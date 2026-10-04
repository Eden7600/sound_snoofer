<p align="center">
  <img src="docs/assets/sound-snoofer-painterly.png" alt="Sound Snoofer: golden protogen mascot with headphones and a yellow digital visor" width="360">
</p>

# Sound Snoofer

A Windows Go application that continuously enforces audio-routing rules through the Voicemeeter Remote API. Supports Banana and Potato.

## Run

Double-click `bin/sound-snoofer.exe`. It opens the TUI in live mode using
`bin/config.json` and remembers choices in `bin/config.json.state.json`.
If the config is missing, the app creates one from bundled defaults.
The config path is relative to the executable, independent of working directory.
Your existing personal configuration and choices have been copied there.

For preview, run `sound-snoofer.exe --dry-run`. No other flags are needed for
normal use. Advanced commands below remain available for diagnostics.

## Personal studio rules

- Volt present: A1 uses Universal Audio Volt ASIO. Physical channel 1 feeds both sides of Stereo Input 1; channel 2 feeds both sides of Stereo Input 2. Your lav is on channel 2.
- Volt absent: input 1 falls back to the Insta360 microphone; input 2 has no direct device. The four ASIO input patch cells are reset.
- Playback priorities: AirPods, then SteelSeries Arena, then Volt. Playback occupies the lowest free output (normally A2 with Volt, A1 without). Unrelated occupied buses are preserved.
- Existing playback sends move with playback. The logical rule virtual:1 -> playback continuously keeps primary VAIO routed to the playback device, even after a manual button change. Banana/Potato strip indexes are resolved at runtime.

The active default configuration is bin/config.json (Git-ignored); config.local.json is retained as the pre-migration copy. config.voice.json is a tracked voice/recording example. The original config.example.json demonstrates compatible fixed-slot WDM mode.

## Build and preview

Requires Windows x64, Go 1.26+, and installed Voicemeeter. No C compiler is needed.

~~~powershell
go build -o bin/sound-snoofer.exe ./cmd/sound-snoofer
go test ./...
go vet ./...
.\bin\sound-snoofer.exe devices --json
.\bin\sound-snoofer.exe plan --config config.local.json
.\bin\sound-snoofer.exe watch --config config.local.json
~~~

If Go is missing from PATH, use & 'C:\Program Files\Go\bin\go.exe' instead. The Remote DLL is discovered through the install registry key; --dll accepts an explicit absolute path. No DLL is downloaded or bundled.

TUI and watch are live by default. Live operation:

~~~powershell
.\bin\sound-snoofer.exe apply --config config.local.json
.\bin\sound-snoofer.exe watch --config config.local.json
~~~

Ctrl+C stops watch and releases ownership; it leaves settings in place. Only one writer per Windows session may run. Stop watch to make manual changes without rule enforcement.

## Configuration

Choose either fixed routes or studio, never both. Regexes use Go RE2 semantics: (?i) ignores case; ^ and $ anchor; matching is otherwise a substring search. Backslashes must be escaped in JSON. Invalid or ambiguous patterns do not pick an arbitrary device.

In studio mode, asio_pattern selects the installed driver, presence_pattern selects a WDM input companion as presence evidence (never as the capture path), playback/fallback_mic are ordered WDM candidates, move_playback_routing transfers enabled sends, and playback_sources declares continuously enforced logical sources. Use virtual:1 for primary VAIO, virtual:2 for AUX, or virtual:3 on Potato. Explicit source rules override migrated state.

Studio mode owns inputs 1/2, ASIO A1, Patch.asio[0..3], outputs matching playback patterns, and managed playback sends. Playback regexes also identify old output assignments to release; keep them specific. Unrelated occupied outputs, B buses, gains, mutes, inserts, and other patch cells are preserved. If A1 belongs to an unrelated device, ASIO takeover reports a conflict. If no playback device is available, the topology is left unchanged and reported unresolved.

Defaults: poll_ms=1000, debounce_ms=1000, verify_ms=5000. Each accepts 100–60000. Device assignment changes use debounce_ms; routing-only changes use 20 ms and wake at that deadline instead of waiting for the next normal poll. Native readback verification adds to total apply time. Config loads once at startup; restart watch after edits. Rules repair managed settings continuously, even if devices do not change. Failed writes back off up to 30 seconds. No automatic application launch or engine restart occurs.

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
.\bin\sound-snoofer.exe tui --config config.local.json
~~~

Use --dry-run for preview from startup, or press **l** to toggle live/dry mode. **Tab** switches Controls / Graph, **j/k** or arrows scroll, **r** reloads configuration, **Space** refreshes, and **q** or **Ctrl+C** quits. Invalid reloads preserve the previous config. Mode changes take effect after the current serialized operation; quit cancels pending waits. The dashboard stays open through connection errors and shows attention/stale state until recovery.

This is a persistent foreground terminal session, not a Windows startup service. It needs interactive stdin/stdout; use watch --json for redirected logs. Dry mode changes no audio settings.

## Selectable voice rules (Potato)

Use `config.voice.json` as an opt-in example, or add `"voice": {}` to your studio configuration. Defaults are desk mic, Element mode, voice enabled and monitoring Off. Existing configs without this object keep legacy behavior.

Start with `sound-snoofer tui --config config.local.json`. Controls is the first view:

- Up/down selects a control. Enter or Space on Source opens connected microphone choices and Off; arrows select, Enter confirms, Escape cancels. Browsing does not change audio.
- Enter/Space cycles Processing or Monitor and toggles playback/capture settings. Off is selected through the source list; there is no separate Off hotkey.
- Tab switches Controls / Graph and cancels an open source list. Page Up/Down scrolls details.
- `l` changes live/dry mode, `r` reloads, `f` refreshes, `x` resets saved choices to config defaults, and `q` exits.

Selections are saved beside the config as `<config-filename>.state.json`, including in dry-run. All commands use this same saved intent. Live permission is never saved. An invalid sidecar blocks writes; repair it or use `x` in the TUI. Concurrent external edits require reload. A failed save leaves the previous selection active.

Controls show queued choices immediately while earlier audio changes finish.
You can keep changing settings; ordered edits are saved in batches and then
applied by the audio worker. `Queued` is not confirmation of persistence or mixer
readback. Wait for saved/applied status before quitting if you want all pending
edits retained. Reload, reset, live toggle and recorder actions wait until settings
finish; navigation and quit remain available. A rejected batch restores saved choices.

The profile dedicates B2 to Element and B3 to the app microphone. It owns all strips' B2/B3 sends and all A sends on mic inputs 1/2/3 and AUX. It preserves B1, gains, mutes, effects and inserts. AUX cannot also be an app-playback source. Input 3 is reserved for the configured webcam; unrelated occupants block the plan.

Desk is Volt input 1; lav is always Volt input 2. If Volt disconnects, the selected Volt source falls back to webcam on input 3 and returns when Volt reconnects. Silence does not indicate a dead battery or trigger a mic switch.

Element must use Voicemeeter AUX Virtual ASIO channels 1/2. The route is mic -> B2 -> Element -> AUX -> B3. Direct instead sends mic -> B3. Discord and other voice apps must capture B3 (observed here as Voicemeeter Out 8); ordinary playback must use primary VAIO. AUX -> B2 stays off to prevent feedback. Keep AUX reserved for Element.

Pre monitoring means before Element, not before Voicemeeter's own effects. Post prefers AUX and falls back to Pre in effective Direct mode. Monitoring follows the chosen playback output. Start with Off; test with headphones at low volume. If Element closes or process status cannot be read, Sound Snoofer temporarily routes the microphone directly and restores Element routing when the process returns. Process presence and mixer readback cannot prove plugin/audio health.

Switching can cause a short gap on affected paths: changed sends are disabled and verified before replacement sends are enabled. Unchanged sends stay untouched unless their input device, ASIO patch or output device is being reconfigured. Failed operations stop the transition and remain visible; there is no atomic rollback. Restarting a live session reconciles from observed state. Removing the profile does not restore previous settings; stop enforcement and restore your recorded mixer/config backup if rolling back.
# Recording

The `devices` command omits recognized virtual endpoints such as
VB-CABLE, Voicemeeter virtual ASIO and SteelSeries Sonar. Internal routing retains
the full inventory. Unknown driver identities remain visible.

The Controls dashboard has a **Source** selector showing connected microphones and Off.
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
inclusion controls and a preferred Pre/Post mic stage. Post stays saved when
Direct is effective, while capture temporarily uses Pre. Returning to available
Element processing restores Post. Recording controls are ordered
Record Computer Audio, Record Microphone, Recording Mic Stage. Start/Stop appear
under a separate Actions heading and require Enter in live mode.
Capture never starts automatically. Quit and dry-run leave native recording
running. Configure the recording folder/format in Voicemeeter first.
See [recording setup and behavior](docs/recording.md).

Routine feedback is concise: Queued, Saved · Preview, Saved · Pending, Applied,
or Reloaded. Success notices disappear after three seconds; pending operations
and errors remain until resolved or superseded. Graph shows observed connections and unresolved destinations; `watch --json` remains available for diagnostic logging.

Playback Device offers Automatic plus connected physical WDM outputs matched by the
configured playback regexes. A manual preference overrides automatic priority while
connected; disconnecting uses the automatic fallback and reconnecting restores the
preference. Add devices to the configured playback candidates to make them selectable.
An installed Volt ASIO driver alone does not expose Desk/Lav: its unique WDM
companion must also be connected. Off and Automatic remain available without inventory.

### Tuning Element with a recorded snippet

1. With Recording to VST off, enable Record Microphone, set Recording Mic Stage to Pre, disable Record Computer Audio for a clean voice sample, then Start/Stop Recording.
2. Load that recording in Voicemeeter's native recorder if it is not already loaded.
3. Keep Processing on Element / VST and an active microphone source selected. Enable Recording to VST and select Monitor Post-VST.
4. Enable Loop Snippet if wanted, then Play Snippet. Stop Playback uses the native recorder Stop command. Turn Recording to VST off to restore the live microphone feed.

Rehearsal feeds the tape to B2 and suppresses physical mic sends and the AUX-to-Discord send. Monitor Pre-VST listens to the dry tape; Post-VST listens to Element's return. Off disables listening. Switching source Off or Processing Direct disables rehearsal. Recording cannot start while rehearsal is enabled. Play starts the loaded tape at zero; no file is automatically opened and transport never starts automatically. Tape A/B sends become managed once rehearsal is used; on leaving rehearsal they are cleared, including after an interrupted transition. Other recorder file/format/gain settings are preserved.

Monitor labels are Off, Pre-VST and Post-VST; saved values remain off/pre/post. Recording stage labels remain Pre/Post.

Send-only changes debounce for 20 ms and verify at 5 ms intervals. The controller skips unchanged operations and uses fresh parameter reads for numeric operations and pending verification, with full device inventory checks at the boundaries. Device assignments retain the configured 1-second default debounce. Read-only measurements on this machine found full inventory reads around 78–80 ms; total application latency includes those boundary checks and native readback, so 20 ms is not an end-to-end guarantee.

With `studio.asio_playback: true` (enabled in the default voice profile), the connected Volt appears as **Universal Audio Volt** in Playback Device and is the last automatic fallback. Selecting it uses the existing ASIO A1 output; no second WDM Volt output is opened. Playback and monitoring move to A1, and move back to A2 when a higher-priority WDM device returns in Automatic mode. An explicit Volt preference overrides automatic priority while connected. The driver alone does not establish presence. Mic Off leaves Volt playback working.

Settings latency audit: TUI edits wake the worker immediately. Device selection performs a fresh availability check before saving; durable state saving remains synchronous. Device transactions use full inventory only at the start/end and before/after each changed device assignment, while sends and pending readback use cheap parameter snapshots. Device verification polls at 20 ms; numeric and recorder confirmation at 5 ms. Normal monitor/mode/capture/loop changes use a 20 ms debounce; source Off or a source change that clears/assigns hardware uses the 1-second device debounce. Physical hotplug discovery can additionally take up to the 1-second inventory poll. Native device opening and verification can extend these times. Error backoff remains 1–30 seconds; uncertain transport commands are not retried.

### Preferred settings and Element availability

Sound Snoofer observes `element.exe` without launching or terminating it. Your Processing preference remains Element when the host is closed or its process status is unknown; effective routing falls back to Direct and restores Element when it is observed running again. Idle detection uses the normal inventory poll; availability is also checked during routing transactions.

Monitor Post-VST and Recording Mic Stage Post are persistent preferences. Whenever effective mode is Direct (including selecting Direct yourself), effective monitoring and capture use Pre. Returning to available Element processing restores Post. Preferences are not rewritten by the fallback. Mic Off still disconnects all mic sends.

Requested Recording to VST pauses while Element is unavailable: tape sends are disconnected, direct live voice is restored if enabled, and Play Snippet is rejected. Reopening Element restores requested rehearsal routing but sends no transport command. A running process proves only availability, not working audio or a valid plugin graph.

Controls show **yellow / … pending** while queued or awaiting relevant readback, and **red / requested → effective** for overrides (for example, Post-VST → Pre-VST). Markers remain visible with NO_COLOR. Normal color returns when the preference is met; unrelated satisfied settings are not globally marked pending.

## License

Sound Snoofer is licensed under the [GNU Affero General Public License v3.0](LICENSE) (AGPL-3.0-only).

## Routing graph

Controls and Graph are the only TUI tabs. Graph displays the current mixer sends
as ASCII branches, including hardware outputs, ASIO microphone patches, app audio,
Element sends/returns and recorder paths. Preview and pending edits still show
observed mixer state. Missing observations are marked unknown; arrows describe
configured routing, not measured audio. Dotted arrows identify the unverified
external Element path. Use arrows/j/k or Page Up/Down to scroll.
