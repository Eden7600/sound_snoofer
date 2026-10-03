# Voice Snooter

A Windows Go application that continuously enforces audio-routing rules through the Voicemeeter Remote API. Supports Banana and Potato.

## Personal studio rules

- Volt present: A1 uses Universal Audio Volt ASIO. Physical channel 1 feeds both sides of Stereo Input 1; channel 2 feeds both sides of Stereo Input 2. Your lav is on channel 2.
- Volt absent: input 1 falls back to the Insta360 microphone; input 2 has no direct device. The four ASIO input patch cells are reset.
- Playback priorities: AirPods, then SteelSeries Arena. Playback occupies the lowest free output (normally A2 with Volt, A1 without). Unrelated occupied buses are preserved.
- Existing playback sends move with playback. The logical rule virtual:1 -> playback continuously keeps primary VAIO routed to the playback device, even after a manual button change. Banana/Potato strip indexes are resolved at runtime.

The current personal configuration is config.local.json (Git-ignored); config.studio.json is its tracked example. The original config.example.json demonstrates compatible fixed-slot WDM mode.

## Build and preview

Requires Windows x64, Go 1.26+, and installed Voicemeeter. No C compiler is needed.

~~~powershell
go build -o bin/voice-snooter.exe ./cmd/voice-snooter
go test ./...
go vet ./...
.\bin\voice-snooter.exe devices --json
.\bin\voice-snooter.exe plan --config config.local.json
.\bin\voice-snooter.exe watch --config config.local.json
~~~

If Go is missing from PATH, use & 'C:\Program Files\Go\bin\go.exe' instead. The Remote DLL is discovered through the install registry key; --dll accepts an explicit absolute path. No DLL is downloaded or bundled.

Watch is dry-run unless --apply is provided. Live operation:

~~~powershell
.\bin\voice-snooter.exe apply --config config.local.json
.\bin\voice-snooter.exe watch --config config.local.json --apply
~~~

Ctrl+C stops watch and releases ownership; it leaves settings in place. Only one writer per Windows session may run. Stop watch to make manual changes without rule enforcement.

## Configuration

Choose either fixed routes or studio, never both. Regexes use Go RE2 semantics: (?i) ignores case; ^ and $ anchor; matching is otherwise a substring search. Backslashes must be escaped in JSON. Invalid or ambiguous patterns do not pick an arbitrary device.

In studio mode, asio_pattern selects the installed driver, presence_pattern selects a WDM input companion as presence evidence (never as the capture path), playback/fallback_mic are ordered WDM candidates, move_playback_routing transfers enabled sends, and playback_sources declares continuously enforced logical sources. Use virtual:1 for primary VAIO, virtual:2 for AUX, or virtual:3 on Potato. Explicit source rules override migrated state.

Studio mode owns inputs 1/2, ASIO A1, Patch.asio[0..3], outputs matching playback patterns, and managed playback sends. Playback regexes also identify old output assignments to release; keep them specific. Unrelated occupied outputs, B buses, gains, mutes, inserts, and other patch cells are preserved. If A1 belongs to an unrelated device, ASIO takeover reports a conflict. If no playback device is available, the topology is left unchanged and reported unresolved.

Defaults: poll_ms=1000, debounce_ms=2000, verify_ms=5000. Each accepts 100–60000. Device assignment changes use debounce_ms; routing-only changes use 100 ms and wake at that deadline instead of waiting for the next normal poll. Native readback verification adds to total apply time. Config loads once at startup; restart watch after edits. Rules repair managed settings continuously, even if devices do not change. Failed writes back off up to 30 seconds. No automatic application launch or engine restart occurs.

## Verification and limitations

Automated tests cover transitions in both directions, primary-send drift repair, edition mappings, and failure handling. Read-only Banana discovery and studio preview have run successfully. Physical switching/listening acceptance and Potato hardware testing are still pending. No live settings have been applied by development commands.

Assignment and numeric readback are verified; this does not prove audible sound. A1 changes may interrupt audio. Operations are not atomic, and partial progress remains after failure. After an interrupted migration with several owned playback outputs, enabled sends are unioned before migration. There is no persisted crash-recovery journal or automatic rollback. WDM companion freshness still needs physical unplug/replug verification; an ASIO driver alone is never considered proof of presence. Inventory available=false for non-WDM entries means ineligible as direct presence evidence, not necessarily disconnected hardware.

--json provides structured inventory/plans and watch JSON Lines. Apply emits an initial plan followed by operation events. For manual recovery, record devices --json before a live trial, stop watch, and restore settings in Voicemeeter.

## OpenSpec

OpenSpec 1.14.0 is pinned in package.json/pnpm-lock.yaml and needs Node 20.19+. The Go application does not need Node.

~~~powershell
pnpm install --frozen-lockfile
node node_modules/@fission-ai/openspec/bin/openspec.js instructions apply --change automatic-device-routing --json
node node_modules/@fission-ai/openspec/bin/openspec.js validate --all --strict --no-interactive
~~~

The active change tracks the expanded ASIO/routing scope and remains unarchived until hardware acceptance. See docs/hardware-acceptance.md and docs/remote-api.md.

## Persistent terminal interface

Run from the project directory:

~~~powershell
.\bin\voice-snooter.exe tui --config config.local.json
~~~

Add --apply for live enforcement from startup, or press **l** to toggle live/dry mode. **Tab** switches Routing / Devices / Events, **j/k** or arrows scroll, **r** reloads configuration, **Space** refreshes, and **q** or **Ctrl+C** quits. Invalid reloads preserve the previous config. Mode changes take effect after the current serialized operation; quit cancels pending waits. The dashboard stays open through connection errors and shows attention/stale state until recovery.

This is a persistent foreground terminal session, not a Windows startup service. It needs interactive stdin/stdout; use watch --json for redirected logs. Dry mode changes no audio settings.

## Selectable voice rules (Potato)

Use `config.voice.json` as an opt-in example, or add `"voice": {}` to your studio configuration. Defaults are desk mic, Element mode, voice enabled and monitoring Off. Existing configs without this object keep legacy behavior.

Start with `voice-snooter tui --config config.local.json`. Controls is the first view:

- Up/down selects a control. Enter or Space on Source opens Desk/Lav/Webcam/Off; arrows select, Enter confirms, Escape cancels. Browsing does not change audio.
- Enter/Space cycles Processing or Monitor and toggles playback/capture settings. Off is selected through the source list; there is no separate Off hotkey.
- Tab switches Controls / Routing / Devices / Events and cancels an open source list. Page Up/Down scrolls details.
- `l` changes live/dry mode, `r` reloads, `f` refreshes, `x` resets saved choices to config defaults, and `q` exits.

Selections are saved beside the config as `<config-filename>.state.json`, including in dry-run. All commands use this same saved intent. Live permission is never saved. An invalid sidecar blocks writes; repair it or use `x` in the TUI. Concurrent external edits require reload. A failed save leaves the previous selection active.

The profile dedicates B2 to Element and B3 to the app microphone. It owns all strips' B2/B3 sends and all A sends on mic inputs 1/2/3 and AUX. It preserves B1, gains, mutes, effects and inserts. AUX cannot also be an app-playback source. Input 3 is reserved for the configured webcam; unrelated occupants block the plan.

Desk is Volt input 1; lav is always Volt input 2. If Volt disconnects, the selected Volt source falls back to webcam on input 3 and returns when Volt reconnects. Silence does not indicate a dead battery or trigger a mic switch.

Element must use Voicemeeter AUX Virtual ASIO channels 1/2. The route is mic -> B2 -> Element -> AUX -> B3. Direct instead sends mic -> B3. Discord and other voice apps must capture B3 (observed here as Voicemeeter Out 8); ordinary playback must use primary VAIO. AUX -> B2 stays off to prevent feedback. Keep AUX reserved for Element.

Pre monitoring means before Element, not before Voicemeeter's own effects. Post uses AUX and is inactive in Direct mode. Monitoring follows the chosen playback output. Start with Off; test with headphones at low volume. To recover if Element stops, select Direct explicitly: process presence and mixer readback cannot prove audio is flowing, and Voice Snooter does not automatically expose dry voice.

Switching can cause a short gap on affected paths: changed sends are disabled and verified before replacement sends are enabled. Unchanged sends stay untouched unless their input device, ASIO patch or output device is being reconfigured. Failed operations stop the transition and remain visible; there is no atomic rollback. Restarting a live session reconciles from observed state. Removing the profile does not restore previous settings; stop enforcement and restore your recorded mixer/config backup if rolling back.
# Recording

The Devices tab and `devices` command omit recognized virtual endpoints such as
VB-CABLE, Voicemeeter virtual ASIO and SteelSeries Sonar. Internal routing retains
the full inventory. Unknown driver identities remain visible.

The Controls dashboard has one **Source** selector: Desk → Lav → Webcam → Off.
Press Enter or Space to open the list, then Enter to confirm. Off disconnects
all A1–A5/B1–B3 sends on the desk, lav, webcam and Element return strips. It also
clears microphone input assignments and ASIO input patches. Volt stays on A1;
playback keeps its normal output. Selecting a microphone reconnects its inputs. Computer
audio and recorder transport continue. Choosing a microphone restores your
processing, monitoring and recording choices. The choice persists; dry-run only
previews, and live mode reports pending/errors until routing is verified.
Press **L** to switch between live and preview, Tab to change views, and arrows
or j/k to select a control. The dashboard adapts to terminal size and supports
`NO_COLOR=1`. Start/Stop require Enter; Space cannot start a recording.

The Potato voice profile supports B1 recording with persistent mic/computer
inclusion controls and a Pre/Post mic stage. The stage changes automatically
to Pre when switching to Direct; Post is only selectable in Element.
Returning to Element keeps Pre until explicitly changed. Recording controls are ordered
Record Computer Audio, Record Microphone, Recording Mic Stage. Start/Stop appear
under a separate Actions heading and require Enter in live mode.
Capture never starts automatically. Quit and dry-run leave native recording
running. Configure the recording folder/format in Voicemeeter first.
See [recording setup and behavior](docs/recording.md).

Routine feedback is concise: Queued, Saved · Preview, Saved · Pending, Applied,
or Reloaded. Success notices disappear after three seconds; pending operations
and errors remain until resolved or superseded. Full diagnostics are in Events.
