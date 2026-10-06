# Design

One compiled plugin, `hue`, with no dependency on other plugins. It has two halves: the room (Hue Bridge, CLIP v2) and sync (the Hue Sync PC app's third-party control socket). A single worker goroutine owns both connections and all their state, which lets the two halves coordinate directly. It serializes writes, publishes control snapshots and joins its child goroutines in Stop. Control handlers only enqueue into a bounded queue (capacity 16) and reject when full. Preview (`Services.Live == false`) reads and observes but publishes every write control unavailable. Nothing is replayed on reconnect and no uncertain command is retried.

Revision 4: scene artwork from scene colors, Home brightness on dial 5 (index 4), and a core `Hidden` control flag so sync-only keys disappear when unusable.

Revision 3 (after use): the color-temperature dial is removed entirely, by user decision. That covers `hue.temperature`, the `neutral_kelvin` setting, color-temperature observation, the GUI Temperature readout, the `hue-temperature` icon and the Home dial 4 binding. Saved settings containing `neutral_kelvin` are converted manually, with no migration code (the plugin was never released); the personal config is edited while Snoofer is stopped.

Revision 2 (after first use): Hue Sync merged into `hue`, mDNS fixed for multi-adapter hosts, a dedicated Lights GUI screen, a Plugins page reduced to enable/disable, room scene slots, and Hue bindings on the personal Home page. The separate `huesync` plugin and its `no_huesync` tag are removed; it was never released beyond this change.

## Settings
```json
"hue": {"address": "", "bridge_id": "", "app_key": "", "certificate_sha256": "",
        "group": "", "sync_port": 24851}
```
- `address` optionally overrides discovery (IP or host name).
- Pairing writes `bridge_id`, `app_key` and `certificate_sha256`; the Lights screen writes `group`. All writes use `Services.SaveSettings` with the settings payload last read, so concurrent edits are rejected rather than lost.
- `sync_port` is optional: absent means 24851, and a present value must be 1–65535. Absent and zero stay distinct.
- Empty strings mean not configured. `app_key` never appears in control values, statuses or diagnostics.

## Room half (Hue Bridge)

### Discovery and identity
Without `address`, the worker sends an mDNS PTR query for `_hue._tcp.local` (via `golang.org/x/net/dns/dnsmessage`) on every up, multicast-capable, non-loopback IPv4 interface. Each query uses a socket bound to that interface's address with the multicast interface set explicitly (`golang.org/x/net/ipv4`). The worker collects unicast replies from all interfaces for 3 s and treats each reply's source address as a candidate. Querying only the OS default multicast route missed the bridge on a host with Tailscale, WSL and several Wi‑Fi adapters (observed 2026-10-05). An interface that fails to send is skipped; discovery fails only if every interface fails. Each candidate is confirmed with unauthenticated `GET https://<ip>/api/0/config`, which supplies `bridgeid`. Outcomes stay distinct:
- None found: status `No bridge`, rediscover every 60 s. The diagnostic suggests setting the address.
- One found: use it. While unpaired, status `Not paired` and the diagnostic `Bridge <ip>`, so the pairing UI can name it.
- Several found and none matches the saved `bridge_id`: status `Multiple bridges`, with candidates listed. There is no automatic pick.
- Paired: only a candidate with the saved `bridge_id` is accepted, so DHCP address changes are followed. With `address` set, a different `bridgeid` is a hard error.

### TLS
The client pins on first use. Pairing records the SHA-256 of the leaf certificate (DER encoding), and every later connection requires that leaf. A mismatch sets status `Error` (certificate changed) and requires pairing again. Before pairing, only `/api/0/config` and the pairing POST run unpinned.

### Pairing
`hue.pair` (press) opens a 30 s window that posts `{"devicetype":"snoofer#<hostname>"}` to `/api` once per second. Hue error 101 shows `Press button`. Success saves `bridge_id`, `app_key` and the pin atomically, then reconnects. Other errors, or the window expiring, end pairing with a local error. Pressing Pair again during the window has no effect. Pairing a different bridge clears `group`.

### Model and observation
Resource payloads differ by type. For example, `status` is an object on scenes but a string on `zigbee_connectivity`. The decoder reads only the fields this plugin uses, accepts a non-object `status` as absent, and decodes each resource separately. A malformed resource of a type the plugin does not use (anything other than room, zone, device, light, grouped_light and scene) is skipped, so unrelated bridge data never blocks the load or the event stream. A malformed resource of a used type is still an error. (Found on the real bridge: "cannot unmarshal string into ... sceneStatus".)

After connecting, the worker loads all CLIP v2 resources, then follows `/eventstream/clip/v2` (SSE), applying `update`/`add`/`delete` events. Every (re)connect reloads fully. Reconnects back off from 1 s to 30 s. A stream gap marks observed values `N/A`.

The configured group resolves to a room or zone. Its `grouped_light` supplies observed `on` and average brightness. Member lights are the room's device light services or the zone's lights.

### Knob semantics
- Brightness: one tick is 2%, clamped to 1–100%. Rotation never turns the room off. Rotating up while off sends `on:true` with the brightness; rotating down while off is ignored. A press toggles on/off from observed state, and with unknown state the press is rejected.

### Write coalescing and verification
Each knob keeps one requested target, so ticks never queue. At most one `PUT grouped_light` is outstanding, at least 250 ms apart, always carrying the latest targets. HTTP 429/503 doubles the gap (up to 2 s) and resends the latest value. While pending, dials show the requested value (`Subdued` in the GUI). If the bridge reports nothing within one step of the target within 3 s, the dial shows `Error` and later ticks start from the observed value.

### Scenes
`hue.scene-<group-slug>-<id8>` (press) exists for every scene. Label is `<Room> <Scene>`, ShortLabel is the scene name, Value is `Active`/`Ready`. A press recalls with `{"recall":{"action":"active"}}` and shows `Wait` until the bridge reports the scene active, or `Error` after 3 s.

`hue.room-scene-1` … `hue.room-scene-12` are stable slots for the selected group's scenes, sorted by name. Slot *n* mirrors the *n*th scene's label, state and recall, so deck bindings survive room changes and renames. Slots beyond the scene count, or without a room, publish as unavailable with empty Label, ShortLabel and Icon. The deck renders such a slot as a blank key (see Presentation). Recalling through a slot targets the scene shown when the input was generated: the request carries the slot's revision, which changes whenever the mapped scene changes, so a stale press is rejected.

### Scene artwork
Scene and slot controls carry a generated `Artwork` thumbnail: a square transparent PNG at most 64 px (the existing artwork contract) showing a disc split into wedges, one per dominant scene color.
- **Color source.** Per-light `actions` of lights the scene turns on. `color.xy` uses the Hue wide-gamut conversion to sRGB, normalized to full brightness. `color_temperature.mirek` uses a blackbody approximation. Each `gradient` point counts as an equal share of that light.
- **Fallback.** If no lit action has color data (for example, a dimming-only scene), the scene `palette` colors and temperatures are used. Without either, no artwork is generated and the bulb icon remains.
- **Brightness.** Wedge brightness follows the action dimming, scaled to 55–100% so dim scenes stay visible.
- **Merging.** Colors closer than a small RGB distance merge. At most five wedges are drawn, largest first, sized by light count.
- **Caching.** Rendering is deterministic, and results are cached by scene ID and color content, so publishing does not re-encode. Scene data changes refresh the image.
- **Surfaces.** On the deck, artwork replaces only the central symbol; label and state badge keep semantic colors. The Lights screen scene cards show the same thumbnail.
- **Model.** The resource model reads scene `actions` (target, on, dimming, color, color_temperature, gradient) and `palette` for this purpose only. No color-temperature control returns.

### Hidden controls
`snoofer.Control` gains `Hidden bool`: a provider-owned signal that the control currently has no useful place on surfaces. The Stream Deck renders a bound hidden control as a blank key, ignores input on it, and keeps the binding. GUIs omit it. Hidden controls also publish `Available: false`, so dispatch is already rejected. Hue sets it as follows:
- `hue.sync-mode` and `hue.sync-intensity` while Hue Sync is not syncing.
- `hue.sync` while the Hue Sync app is not connected (closed, third-party control off, or no state yet).

Status and the Lights screen still explain how to start sync. The configurator still lists hidden controls by label, so bindings can be made at any time. Empty room scene slots keep the existing blank rule.

## Sync half (Hue Sync PC app)

### Protocol (from the Hue Sync binary and the Elgato plugin source)
`ws://127.0.0.1:<sync_port>/` exists only when Hue Sync *Settings → Third-party control* is on. It has no authentication and listens on loopback only. Commands are sent as `{"command":C,"data":{...}}`:
- `start_sync` (optional `mode`, `intensity`)
- `stop_sync`
- `set_intensity`, `set_app_mode` (effective only while syncing)
- `inc_bri {step}` (relative, 0–100 scale)

The app pushes `app_state_update` with `state` (`bridge_connected`/`bridge_disconnected`/`syncing`), `mode`, `intensity` and `bri`. Unknown fields and events are ignored. The client uses `github.com/gorilla/websocket`, already in the module graph through Wails.

### Connection
Dial timeout is 3 s; reconnects back off from 1 s to 30 s. A refused connection is `N/A`, and the diagnostic names both remedies (start Hue Sync, enable Third-party control). State is unknown until the first event. A disconnect drops pending commands without resending them. When the app reports `bridge_disconnected`, every sync control is unavailable.

### Sync controls
| ID | Kind / ops | Behavior |
| --- | --- | --- |
| `hue.sync-status` | status | `Syncing`, `Ready`, `No bridge`, `N/A` |
| `hue.sync` | toggle / press | `On`/`Off`. Sends `start_sync` (keeps the app's mode and intensity) or `stop_sync`. `Wait` until confirmed, `Error` after 3 s. |
| `hue.sync-mode` | selection / set | Video, Games, Music. Available only while syncing. |
| `hue.sync-intensity` | selection / set | Subtle, Moderate, High, Extreme. Available only while syncing. |

## Joining the halves
- **One brightness dial.** `hue.brightness` follows whatever drives the lights. While Hue Sync reports `syncing`, rotation sends coalesced `inc_bri` (2 per tick, 100 ms minimum gap) and the value shows the sync brightness as `Sync 62%`. Otherwise it controls the room as above. A press always toggles the room on/off, and Sync has its own key.
- **Scenes win over sync.** Pressing a scene (or slot) while syncing sends `stop_sync` first. The recall is sent only after the app confirms the sync stopped, within 3 s; otherwise the scene shows `Error` and nothing is recalled. Without a sync connection, scenes recall directly.
- **Status:** `hue.status` reports the bridge and `hue.sync-status` reports the app. The Lights screen shows both side by side.

## Controls summary (group `Hue`)
- **Room half:** `hue.status`, `hue.pair`, `hue.group` (selection; room/zone IDs with name labels, zones suffixed "(zone)"), `hue.brightness`, the scene controls and the slots.
- **Sync half:** `hue.sync-status`, `hue.sync`, `hue.sync-mode`, `hue.sync-intensity`.

## GUI
The **Lights** screen sits in the sidebar after Soundboard. It is built from the controls above, never writes settings directly, and does not appear in the Plugins page.
- **Setup banner** (only while something is missing):
  - Disabled plugin: an Enable button that dispatches the existing plugin selection (confirmation dialog).
  - `No bridge`: the diagnostic, plus a note that `hue.address` can be set.
  - Unpaired: "Bridge <ip> found" with Pair. During the pairing window: "Press the link button on the bridge" with a countdown hint from the `Press button` state. Then success or the error.
  - Certificate error: Pair again.
- **Room card (left):** the room selector, then two large readouts with −/+:
  - Brightness, with an On/Off button.
  - A note "Sync controls brightness" while syncing.
  - Below them, a scene grid for the selected room (Active highlighted, Wait/Error per card). An "Other rooms" disclosure lists the remaining scenes grouped by room.
- **Sync card (right):** a large Sync toggle, segmented Mode and Intensity buttons (disabled with "Start sync to change" while not syncing), and the app status. When unreachable it shows: "Open Hue Sync and turn on Settings → Third-party control".

The **Plugins** page becomes enable/disable only: plugin cards with status and Enable/Disable, plus Retry. The generic fallback forms move off this page. Built-in plugins have their own screens. Third-party plugin controls remain available as deck bindings, and their statuses still appear in Diagnostics. `docs/plugins.md` changes accordingly.

## Presentation (deck)
- **Icons:** `hue-scene` (bulb), `hue-brightness` (sun), `hue-pair` (link), `huesync-sync` (screen with rays), `huesync-mode` (segments), `huesync-intensity` (wave). Scene `Active` uses the Active color.
- **Deck font:** gains a `%` glyph.
- **Dial text:** non-page dials no longer print a control's Icon identifier as strip text.
- **Empty slots:** a control that is unavailable and has no Label, ShortLabel or Icon renders as a blank key instead of `N/A`. Normal unavailable controls still show `N/A`.

## Personal layout (bin/snoofer.json, user-owned and not in git)
Home keeps every existing binding.
- **Rightmost four columns** (zero-based keys 5–8, 14–17, 23–26, 32–35), left open for Hue:
  - Row 0: `hue.sync`, `hue.sync-mode`, `hue.sync-intensity`, `hue.brightness` (press = lights on/off).
  - Rows 1–3: `hue.room-scene-1` … `-12`.
- **Dials:** index 4 (dial 5, beside pagination) is `hue.brightness`; indexes 2 and 3 stay free; dial 6 remains pagination.
- **Contract wording:** "index 2 is empty" only described the result of moving Mic in `e4ea932`; it was never a reservation, and the contract is corrected.
- **Config cleanup:** the obsolete `huesync` config entry is removed. No Lights deck page is added.

## Out of scope
Per-light control, color (xy), dynamic scene playback, Entertainment API streaming, scene editing, the cloud API and the HDMI Sync Box.
