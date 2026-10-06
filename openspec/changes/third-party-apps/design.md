# Design

## Connection report (core)
```go
// snoofer/connection.go
type ConnectionState string // Typed state; drives tone. Control.Value carries the short display word.
const (
    ConnectionConnected    = "connected"     // Working.
    ConnectionReady        = "ready"         // Stateless integration usable (e.g. SendInput).
    ConnectionConnecting   = "connecting"    // Attempt or recovery in progress.
    ConnectionAttention    = "attention"     // Working but degraded (stall, suspension, partial).
    ConnectionDisconnected = "disconnected"  // Expected peer absent (app closed, device unplugged, process not running).
    ConnectionOff          = "off"           // Feature disabled or not in use; not a fault.
    ConnectionUnconfigured = "unconfigured"  // Needs setup (unpaired, no renderer configured).
    ConnectionError        = "error"         // Failure needing action.
    ConnectionUnknown      = "unknown"       // Cannot determine.
)
type Connection struct {
    State        ConnectionState
    Endpoint     string     // What is talked to: path, address, URL, device, process name.
    Since        time.Time  // When State was entered; zero if unknown.
    LastActivity time.Time  // Last successful exchange; zero if none.
    LastError    string     // Most recent failure, kept after recovery for diagnosis.
    LastErrorAt  time.Time
    Details      []ConnectionDetail // Ordered, provider-chosen facts.
}
type ConnectionDetail struct{ Label, Value string }
```
- `Control.Connection *Connection` is optional. Reports use `Kind: "connection"`, `Group` = the owning plugin's display name, `Label` = the app name, `Value` = a short state word, no Operations, nil invoke and `SurfaceOnly: true`. They are never Stream Deck bindable, because `compatible()` needs press/set/adjust.
- **Status rule:** `Status` stays empty; errors live in `Connection`. Diagnostics notices therefore do not duplicate reports, and the deck never shows them.
- **Ownership:** `Controls.Publish` and `Snapshot` deep-copy `Connection` (including `Details`), so providers never share mutable slices with readers. Publishing is otherwise unchanged. A changed timestamp is a changed control, and revision churn on status-only controls is harmless.
- **Timestamps:** wall-clock instants, which marshal as RFC 3339. The GUI computes "ago" against its own clock (same machine).
- **Credentials:** providers never put them in reports (the Hue app key, for example). Endpoint and details are safe to copy.

`snoofer.ConnectionTracker` (owned by the publishing goroutine) implements these semantics so every provider handles Since, activity and retained errors identically.

### `Since` semantics
Set when the reported State changes, and kept otherwise. `LastError` and `LastErrorAt` survive recovery until a newer failure replaces them; the card marks a recovered error as "Last error" in muted tone.

## GUI: Third-party apps screen
- **Sidebar:** "Third-party apps" between Plugins and Diagnostics, eyebrow `SYSTEM`.
- **Summary bar:** counts per tone (Connected/Ready, Attention/Connecting, Error/Disconnected, Off/Not configured) and Copy all.
- **Cards:** one per report, grouped by `Group`, in a responsive grid.
  - **Header:** app name, state badge and tone (mapping below).
  - **Body:** endpoint (monospace), "Since 3m ago", "Last activity 2s ago" (with an amber "stale" marker when older than 3× the provider's stated interval, if a detail named `Interval` is present), and the last error with its age.
  - **Details:** a definition list.
  - **Copy details:** a plain-text block with app name, state, ISO timestamps, endpoint, error and details.
- **Disabled plugins:** shown as a compact "Not monitored" list (plugin ID and status) so absence is explicit.
- **Refresh:** relative times update in place every poll (200 ms) without rebuilding. The screen rebuilds only when the report set changes (the existing `layoutKey`).
- **Copy:** `navigator.clipboard.writeText`, falling back to a hidden textarea with `execCommand("copy")`, with a transient "Copied" confirmation or a local error.
- **Tone mapping:**
  - connected and ready: active (cyan);
  - connecting and attention: attention (amber);
  - error: critical;
  - disconnected: critical if the provider marks the peer as required (detail `Required: Yes`), otherwise neutral;
  - off, unconfigured and unknown: neutral.

## Integrations
Each report is built by the plugin that owns the integration, from state it already owns, published with its other controls by the same goroutine. Gaps found during inventory are filled with tracking only; no behavior changes. Report IDs use the form `<plugin>.app-<name>`.

| ID | Endpoint | State sources | Details |
| --- | --- | --- | --- |
| `audio.app-voicemeeter` | Remote API DLL path in use | Connected/readError, open error, Live | Edition (Basic/Banana/Potato), Voicemeeter version (`VBVMR_GetVoicemeeterVersion`, optional export), Login result (0 = running, 1 = launched/not running), Poll interval, Health, Recovery, Writer (live owner / preview) |
| `audio.app-callback` | `snoofer-audio-monitor.dll` | `Snapshot.Callback` nil → Off; Active → Connected; stall health → Attention; Error → Error | Buffers, Synced, Starting, Ending, Changes, Recovery outcome |
| `audio.app-recorder` | Voicemeeter Recorder | not configured → Off; `Recorder.Error` → Error; else Connected | Transport state, Armed sources, Conflict |
| `audio.app-asio` | ASIO name (or "None") | `ASIOActive` → Connected; unavailable → Disconnected; ambiguity → Error | Configured priority match, Sample rate (`Bus[0].device.sr`), A1 device |
| `audio.app-element` | `element.exe` | `Snapshot.Element` Known/Running/Error | Processing reason, Effective mode |
| `audio.app-windows-defaults` | Windows Core Audio (MMDevice/IPolicyConfig) | Result Kind: Disabled → Off, Preview → Off, Verified → Connected, Attention → Attention, Unknown → Unknown | Playback target, Capture target, Suspended, Last correction |
| `vr.app-steamvr` | `vrserver.exe` | Known/Running | Last check, Profile (VR/Normal) |
| `streamdeck.app-device` | Stream Deck + XL (USB 0FD9:00C6) | device connect/disconnect/error events, separated from layout status | Serial, Connections this session, Last disconnect reason |
| `soundboard.app-playback` | renderer name + `snoofer-soundboard.dll` | readiness, player loaded, last native error | Renderer, Companion loaded, Routing ready, Last clip result |
| `hue.app-bridge` | bridge address | worker status | Bridge ID, Name, Software version and API version (from the `/api/0/config` probe), Event stream (open/closed), Last event, Reconnects, Paired |
| `hue.app-sync` | `ws://127.0.0.1:<port>/` | sync link | App state, Mode, Intensity, Last event, Reconnects |
| `media.app-keys` | Windows SendInput | Ready; last send failure → Error until the next success | Last command, Last sent |

### Tracking added
- **Voicemeeter (worker-owned, copied into `control.State`):**
  - `Native.DLLPath`, `LoginCode` and `Version`, read once after open;
  - `ConnectedSince`, set on Connected transitions;
  - `LastError`/`LastErrorAt` kept across recovery;
  - Windows defaults `Suspended` and resolved target names added to `windowsaudio.Result`.
- **SteamVR:** check time and state-entered time in the VR goroutine.
- **Stream Deck:** a separate `device` struct (connected, serial, since, connections, last error) in the plugin goroutine. Layout status stops receiving device text.
- **Soundboard:** companion-loaded flag, last native error and time, last clip result and time.
- **Hue:**
  - `connectedSince`, `lastEvent`, `reconnects`, plus bridge name and versions from the probe;
  - Hue Sync `connectedSince`, `lastEvent`, `reconnects`;
  - the most recent error retained.
- **Media:** last command, time and error.

## Out of scope
Reconnect actions (diagnostics only, by user decision), core host facilities (tray icon, single-instance event, controls process) and the CLI maintenance path, which runs outside the host.
