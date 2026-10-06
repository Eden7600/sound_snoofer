# Design

Two compiled plugins, `hue` and `huesync`, with no dependencies on other plugins. Each owns one worker goroutine that holds all network state, serializes writes, publishes control snapshots and joins in Stop. Control handlers only enqueue into a bounded queue (capacity 16) and reject when full. Preview (`Services.Live == false`) reads and observes but publishes every write control unavailable. Neither plugin replays commands on reconnect or retries an uncertain command.

## Hue Bridge (`hue`)

### Settings
```json
"hue": {"address": "", "bridge_id": "", "app_key": "", "certificate_sha256": "",
        "group": "", "neutral_kelvin": 4000}
```
`address` optionally overrides discovery (IP or host name). Pairing writes `bridge_id`, `app_key` and `certificate_sha256`; the GUI writes `group`. All writes use `Services.SaveSettings` with the settings payload last read, so concurrent edits are rejected rather than lost. `neutral_kelvin` is validated to 2000–6500. Empty strings mean not configured. `app_key` never appears in control values, statuses or diagnostics.

### Discovery and identity
Without `address`, the worker sends an mDNS PTR query for `_hue._tcp.local` (via `golang.org/x/net/dns/dnsmessage`, IPv4 multicast, 3 s window) and resolves each answer's A record. It confirms each candidate with unauthenticated `GET https://<ip>/api/0/config` and reads `bridgeid`. Outcomes stay distinct:
- None found: status `No bridge`. Rediscover every 60 s.
- One found: use it.
- Several found and none matches the saved `bridge_id`: status `Multiple bridges`, with the candidate IPs and IDs in the diagnostic. Set `address` to choose; there is no automatic pick.
- Paired and `bridge_id` known: only a candidate with that ID is accepted, so a DHCP address change is followed automatically. With `address` set, a different `bridgeid` is a hard error and never used silently.

### TLS
Bridges present either a Signify-CA certificate or a self-signed one. The client does not embed a CA; it pins on first use. Pairing records the SHA-256 of the leaf certificate's DER encoding, and every later connection requires a matching leaf. A mismatch sets status `Error` (certificate changed) and requires pairing again. Before pairing, only `/api/0/config` and the pairing POST run without a pin, and pairing pins the certificate from that same connection.

### Pairing
`hue.pair` (press) opens a 30 s window that posts `{"devicetype":"snoofer#<hostname>"}` to `/api` once per second. Hue error type 101 (link button not pressed) shows `Press bridge button`. Success saves `bridge_id`, `app_key` and the pin atomically. Any other error, or the window expiring, ends pairing with a local error. Pressing again restarts the window. Pairing a different bridge replaces the stored identity and clears `group`.

### Model and observation
After connecting, the worker loads `room`, `zone`, `grouped_light`, `light` and `scene` from `/clip/v2/resource/...`. It then follows `/eventstream/clip/v2` (SSE, `hue-application-key` header) and applies `update`/`add`/`delete` events to an immutable model it owns. Each event-stream (re)connect does a full reload before publishing, because missed events are possible. SSE reconnects back off from 1 s to 30 s. A stream gap or bridge loss marks observed values unknown (`N/A`) without changing settings.

The configured group resolves to a room or zone by resource ID. Its `grouped_light` service supplies observed `on` and average `dimming.brightness`. Member lights are the room's device light services or the zone's light services. Color temperature is observed from member lights that are on and have `color_temperature.mirek_valid`. If all of them agree within one dial step, the value is their mean in kelvin. Otherwise it shows `Mixed`, and with none valid (all off or in color mode) it is unknown. The temperature range is the intersection of the members' `mirek_schema`. An empty intersection uses the union, and the bridge clamps each light.

### Controls (group `Hue`)
| ID | Kind / ops | Value |
| --- | --- | --- |
| `hue.status` | status | `Connected`, `No bridge`, `Multiple bridges`, `Not paired`, `Disconnected`, `Error` |
| `hue.pair` | command / press | `Ready`, `Press bridge button`, `Paired`, `Error` |
| `hue.group` | select / set | room/zone ID options with name OptionLabels |
| `hue.brightness` | numeric / adjust, press | `62%`, `Off`, `N/A` |
| `hue.temperature` | numeric / adjust, press | `4000K`, `Mixed`, `N/A` |
| `hue.scene-<group-slug>-<id8>` | command / press | `Active`, `Ready` |

Scene IDs combine the slug of the owning room/zone name with the first 8 hex digits of the scene UUID. This lets a Stream Deck page use `auto_controls: hue.scene-` for every scene or `hue.scene-studio-` for one room. Renaming a room changes its scene IDs, which leaves manual bindings showing Unavailable (existing behavior) while automatic pages refill. Label is `<Room> <Scene>` and ShortLabel is the scene name. Scenes are listed sorted by label. A scene is `Active` when `status.active` is not `inactive`. Pressing recalls it with `{"recall":{"action":"active"}}`: the key shows `Wait` until a matching status event arrives, and `Error` if none arrives within 3 s.

Without a configured group, both knobs are unavailable with status `Choose room`. If the group disappears from the bridge, they are unavailable with `Room missing`; the setting is kept.

### Knob semantics
- Brightness: one tick is 2%, clamped to 1–100%. Rotating never turns the room off; only a press does. Rotating up while the room is observed off sends `on:true` along with the brightness. Rotating down while off is ignored. A press toggles `on` based on observed state; with unknown observed state the press is rejected.
- Temperature: one tick is 100 K, converted to mirek (`round(1e6/K)`) and clamped to the group range. Rotating while off or with no CT-capable member is rejected locally. A press sets `neutral_kelvin`. From `Mixed`, the first tick starts from the mean.

### Write coalescing and verification
Hue limits group commands (about one per second per group, according to Signify guidance). Each knob keeps one requested target. Ticks update the target from the pending target if there is one, otherwise from the observed value. The worker sends at most one `PUT /clip/v2/resource/grouped_light/<id>` at a time, at least 250 ms apart, always carrying the latest targets, so no backlog builds. HTTP 429/503 doubles the gap up to 2 s, and a success restores it. While a write is pending, the dial shows the requested value with `Subdued` set. The GUI marks it pending, and the deck shows the number rather than `Wait` so the dial stays readable while turning. Observed events confirm the target within one step tolerance. If no confirmation arrives within 3 s of the last send, the control shows `Error` with an observed/requested diagnostic and the pending target is dropped. Later ticks start from the observed value.

A Hue Sync entertainment stream overrides bridge commands for its lights. The plugin reports what the bridge observes and does not stop syncing.

## Hue Sync PC app (`huesync`)

### Protocol (from the Hue Sync binary and the Elgato plugin source)
Hue Sync exposes a WebSocket at `ws://127.0.0.1:<port>/`, default port 24851, only when Hue Sync *Settings → Third-party control* is on. It has no authentication and listens on loopback only. Clients send `{"command":C,"data":{...}}`:
- `start_sync` with optional `mode` (`video`/`games`/`music`) and `intensity` (`subtle`/`moderate`/`high`/`extreme`)
- `stop_sync`
- `set_intensity {intensity}`
- `set_app_mode {mode}` (these two take effect only while syncing)
- `inc_bri {step}`, a relative brightness change on a 0–100 scale

The app pushes `{"event":"app_state_update","data":{"state","mode","intensity","bri"}}`, where `state` is `bridge_connected`, `bridge_disconnected` or `syncing`. Unknown events and fields are ignored. The client uses `github.com/gorilla/websocket`, already in the module graph through Wails, rather than implementing framing by hand.

### Settings
`{"port": 24851}`, validated to 1–65535.

### Connection
The worker dials with a 3 s timeout and reconnects with backoff from 1 s to 30 s. A refused connection cannot tell "app closed" apart from "third-party control off", so the status is `N/A` and the diagnostic names both remedies. Observed state is unknown until the first `app_state_update` after connecting. A disconnect drops pending commands and does not resend them.

### Controls (group `Hue Sync`)
| ID | Kind / ops | Behavior |
| --- | --- | --- |
| `huesync.status` | status | `Syncing`, `Ready`, `No bridge`, `N/A` |
| `huesync.sync` | toggle / press | `On`/`Off`. Sends `start_sync` without data (the app keeps its current mode and intensity) or `stop_sync`. `Wait` until the state event; `Error` after 3 s. |
| `huesync.brightness` | numeric / adjust, press | Adjust sends `inc_bri` with step = ticks × 2, coalesced: ticks accumulate while one command is outstanding (100 ms minimum gap). Press toggles sync, as in the Elgato dial. Value is observed `bri%`. |
| `huesync.mode` | select / set | `video`/`games`/`music` with Video/Games/Music labels. Available only while syncing. |
| `huesync.intensity` | select / set | `subtle`/`moderate`/`high`/`extreme`. Available only while syncing. |

When Hue Sync reports `bridge_disconnected`, every control except status is unavailable.

## Presentation
New code-drawn icons: `hue-scene` (bulb), `hue-brightness` (sun), `hue-temperature` (thermometer), `huesync-sync` (screen with light rays), `huesync-mode` and `huesync-intensity` (wave). `huesync.brightness` reuses `hue-brightness`. The UI contract gains vocabulary rows; scene `Active` uses the Active color. No artwork is generated from scene palettes. The GUI shows both plugins through the existing grouped fallback forms under Plugins; there are no new GUI screens.

## Out of scope
Per-light control, color (xy) control, dynamic scene playback, Entertainment API streaming, scene editing, cloud/remote API, Hue Sync Box (HDMI), and changes to personal deck layouts. Users add a Lights page with the configurator.
