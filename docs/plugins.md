# Snoofer plugins

Snoofer is a Windows host with optional compiled audio, VR, Stream Deck, Windows media and Hue plugins. The repository directory remains sound_snoofer and the Go module remains sound-snoofer. No runtime loader, sandbox or process isolation is provided: plugins are trusted Go code.

Build from the repository:

```powershell
./scripts/build.ps1
./scripts/build.ps1 -Tags core
./scripts/build.ps1 -Tags no_audio
```

Available exclusions: no_audio (also excludes VR and soundboard), no_vr, no_streamdeck, no_soundboard, no_media and no_hue. Core excludes all built-ins and does not build or load the audio companion. The default build includes snoofer-audio-monitor.dll; the Voicemeeter vendor DLL is never distributed.

Launch bin/snoofer.exe. Its adjacent configuration is snoofer.json. A new installation creates disabled plugin entries with inert defaults. Existing personal configurations are converted manually; the application does not attempt migration. --config selects another envelope, --dry-run prevents audio writes, and --check validates the envelope, enabled plugin settings and dependencies without starting devices.

The integrated GUI provides task-oriented screens (Audio, Soundboard, Lights, Stream Deck, Diagnostics) and a Plugins page that only enables, disables and retries plugins; no settings are edited there. Standard Tab navigation, text editing and mouse interaction are supported. Plugin changes show the affected dependencies before Apply & restart. Diagnostics offers Retry plugins. Closing controls leaves the host running. Third-party semantic controls have no generic form; bind them on the Stream Deck, and their statuses appear in Diagnostics. Optional ViewData is an immutable JSON presentation payload owned by its provider.

## Plugin contract

Register snoofer.Plugin descriptors explicitly in a composition that calls app.Run. Descriptors and default settings are inert; no import/init function may open hardware, start workers or acquire native resources. Start runs only when enabled and dependencies have started. Validate runs only for enabled settings. Start must release partially acquired resources on failure.

Start receives core services and a map containing only its declared dependencies. Return a stoppable instance that owns every goroutine, channel and handle. Stop honors the supplied deadline. Cleanup errors prevent automatic relaunch. The host cancels work before stopping dependents in reverse order; a panic in trusted code can still stop the whole application.

Publish namespaced semantic controls through Services.Controls. Handlers run under the registry lock and must enqueue bounded work without device I/O or reentering the registry. Publish new snapshots from the owning worker. Requests carry a control revision; never replay commands on reconnect. Use Services.SaveSettings with the original settings payload and validated replacement for atomic live settings changes.

An external module can import sound-snoofer/app and sound-snoofer/snoofer, plus public built-in factories under plugins/. It must not import internal packages. For local development:

```text
module example.com/my-snoofer
go 1.26.0
require sound-snoofer v0.0.0
replace sound-snoofer => C:/Users/Eden7600/Documents/sound_snoofer
```

Supply the desired descriptors to app.Run in your own main. See snoofer.Plugin, Instance, Services and Control for the small API. Rebuild against the same source version; there is no cross-version binary ABI. The implementation validation includes an external-module build.

## Audio profiles

Audio retains native single-writer ownership, deterministic planning, apply/verify, recorder safeguards and health recovery. Its state_path is the basename of existing .state.json, .mutes.json and .recovery.json sidecars; it need not contain an audio config. Relative paths resolve against the envelope directory. Keep this path unchanged to preserve mute and recovery safety state.

Normal microphone priority is audio.config.profiles.microphones. Normal playback priority remains audio.config.studio.playback. Saved runtime overrides are separate from these configured lists; auto selects microphone priority and the empty playback choice selects playback priority.

VR owns current-session vrserver.exe detection and depends on audio. Its settings contain devices, microphone priorities, playback priorities and default profile choices. An optional last microphone entry normal or playback entry with driver normal delegates that direction to Normal. Without it, no implicit Normal fallback occurs. Unknown detection retains the last known activation and shows a warning. Shared mute/gain/recording preferences survive profile changes.

## Soundboard

Soundboard depends on Audio, not Stream Deck. Enable audio.settings.soundboard_input to reserve Potato VAIO3 (strip 7); remove virtual:3 from ordinary playback_sources. Configure soundboard settings with folder, renderer, microphone and monitor. The renderer must be the exact DirectSound VAIO3 input name; the personal setting is DirectSound: Voicemeeter VAIO3 Input (VB-Audio Voicemeeter VAIO). Both destinations default on: B3 and the current playback output, independent of mic mute, mic stack and Element. Output bus gain/mute still apply. Recording transport and B1 capture choices are unaffected.

MP3 files directly in the folder become semantic clip controls and refresh every five seconds. Playback matches sample peaks to -1 dBFS using one constant gain per clip. Intentional perceived-loudness differences and dynamics remain; there is no loudness leveling or compression. Original MP3s are untouched. The first press prepares a cached WAV (Wait); later presses reuse it. Stop/replacement cancels pending work. Copies live under soundboard-cache/peak-v1 beside the config; changed/deleted sources invalidate obsolete copies. Normalization supports mono/stereo PCM up to 64 MiB decoded; preparation failures do not fall back to raw playback. Matching artwork is optional: clip.mp3 uses clip.png, clip.jpg, clip.jpeg, clip.webp or clip.gif (in that priority, case-insensitive). The decoder recognizes PNG/JPEG/WebP/GIF content even when the extension is misleading. Images must be at most 4096px per dimension and 8 MiB; rectangles fit proportionally without cropping or stretching. GIFs use the first frame as a static icon. They replace the central icon while the name and state remain visible; missing/invalid images use the play icon. Artwork additions, replacements and removals refresh with the catalogue. Pressing any clip replaces the current one unless Overlap is on (soundboard.overlap, persisted as soundboard.overlap in settings, also bindable on the deck): then clips play together, up to 8 at once, and a ninth cuts the oldest. Stop ends every clip. soundboard.volume adjusts VAIO3 gain (-60 to +12 dB) for both destinations; pressing resets to 0 dB without muting. The personal Soundboard page binds it to the first dial. Errors appear in Soundboard controls; there is no automatic retry or default-device fallback. Preview never loads the player. Quitting stops clips and releases playback resources. If soundboard is disabled, keep the audio input reservation so audio clears its sends after restart.

Set a Stream Deck page's auto_controls to soundboard.clip- (Auto controls prefix in the configurator), and bind soundboard.stop to a key. Empty keys fill alphabetically; manual/shared bindings win, and overflow pages repeat the template through the existing page dial. Clear the prefix to use a fully manual page. Home is not altered. Disabling Stream Deck does not disable soundboard's GUI controls.

## Hue

Hue is one plugin with two halves: the room, through a Hue Bridge on the local network (CLIP v2, no cloud account), and sync, through the Hue Sync PC app. Everything is on the Lights screen. If Hue is off, Lights offers Enable Hue.

Pairing: Lights shows the bridge it found, for example "Bridge found at 172.16.102.3". Press Pair, then press the round link button on the bridge within 30 seconds. Pairing saves bridge_id, app_key and certificate_sha256 in hue settings in snoofer.json. The key is a local credential: keep snoofer.json private (it is ignored by git). Then choose the room or zone in the Room picker. Discovery queries mDNS (`_hue._tcp`) on every network adapter, so VPN, WSL and Wi-Fi adapters do not hide the bridge. If none answers, set hue.address to the bridge IP. A paired bridge is followed across DHCP changes by its bridge ID. The certificate is pinned at pairing; if the bridge is replaced or reset, Lights asks you to pair again.

Room: Brightness moves 2% per tick (1–100%). Turning up while off turns the room on, turning down while off does nothing, and a press or On/Off toggles the room. There is no color-temperature control; set white tones with scenes. The Lights screen slider sends the same steps. Fast turns send only the newest value, at most every 250 ms. A value the bridge does not report within 3 seconds shows Error and is not retried.

Scenes: the Lights screen shows the selected room's scenes, with other rooms under Other rooms. For the deck, `hue.room-scene-1` to `hue.room-scene-12` mirror the selected room's scenes by name; changing rooms changes them without editing the layout, and unused slots are blank keys. Every scene also has `hue.scene-<room>-<id>` for `auto_controls` pages (`hue.scene-` or `hue.scene-<room>-`). Recall uses the scene's static colors. Scene keys and cards show a disc of the scene's dominant colors, taken from what the scene sets on each light (or its palette for brightness-only scenes); scenes with no color data keep the bulb icon. The image updates when the scene changes.

Sync: in Hue Sync, open Settings and turn on Third-party control. Snoofer connects to ws://127.0.0.1:24851 (optional hue.sync_port) and reconnects automatically without resending commands. Sync starts or stops with the app's current mode and intensity; Mode (Video, Games, Music) and Intensity (Subtle to Extreme) apply only while syncing, so their keys are blank and their Lights rows hidden until sync starts. The Sync key is blank while the Hue Sync app is not connected.

The halves work together. While syncing, the Brightness dial adjusts the sync brightness (shown as Sync 62%), because the stream controls the lights; a Brightness press still toggles the room. Pressing a scene while syncing stops sync first and recalls the scene once Hue Sync confirms; if it does not stop within 3 seconds, the scene shows Error and nothing is recalled.

## Stream Deck layout

Pages contain 36 keys and five assignable dials. The sixth dial shows previous/current/next page names, highlights the current page, rotates pages and presses Home. Shared bindings reserve positions across every page. Clear a page binding before assigning its position globally; collisions are rejected.

The Stream Deck configurator is part of Controls: select a page, rename/create/reorder/delete, choose Home, choose a key/dial and assign a compatible semantic control. Shared toggles which assignment scope you edit. Effective position previews the merged binding. Save applies the complete draft live; Cancel discards it. Save/Cancel before switching serial layouts.

Missing providers retain their binding IDs and labels and render Unavailable. Hardware may be disconnected while editing. No macros, arbitrary per-binding artwork imports, folders or app-specific page profiles are included. Providers may supply an immutable base64 Artwork PNG thumbnail (square, at most 64px); soundboard uses matching sidecar images.



## Minimal external composition

With the go.mod above, this main.go uses only public contracts. Enable Example in the Plugins tab after the first launch:

```go
package main

import (
    "context"
    "encoding/json"
    "os"

    "sound-snoofer/app"
    "sound-snoofer/snoofer"
)

type example struct{ controls *snoofer.Controls }
func (e example) Stop(context.Context) error {
    e.controls.Remove("example")
    return nil
}
func main() {
    p := snoofer.Plugin{ID: "example", Label: "Example", Start: func(_ context.Context, s snoofer.Services, _ json.RawMessage, _ map[string]snoofer.Instance) (snoofer.Instance, error) {
        err := s.Controls.Publish("example", []snoofer.Control{{ID: "example.status", Label: "Example", Group: "Example", Kind: "status", Value: "Ready", Available: true}}, nil)
        if err != nil { return nil, err }
        return example{s.Controls}, nil
    }}
    if err := app.Run(context.Background(), os.Args[1:], p); err != nil { app.ShowError(err) }
}
```

Build on Windows with `go build -ldflags "-H=windowsgui" -o snoofer.exe .`. Add the built-in factories you want to app.Run; omitting them also omits their native requirements. Plugins that start workers must additionally cancel and join those workers in Stop.

Mic stack enablement (audio.mic-stack) is shared across Normal and VR, using the existing enabled saved choice. The audio.source binding edits the active profile target; fixed-profile source IDs remain unchanged. Off is no longer a target option. Disabling retains both targets and shared mute, and no target/profile transition implicitly enables the stack. Existing saved Off choices load safely as disabled with Automatic as the target. Default/user-owned deck layouts are not rewritten; assign the new enablement control in the configurator if desired.

Controls may publish optional Meter telemetry (Present, Known, DB in dBFS, At timestamp). Meter-only updates do not invalidate command revisions. Publish fresh immutable samples, distinguish unknown from silence, and leave Meter absent for non-metered controls. The Stream Deck expires readings after 500 ms and preserves existing knob bindings.

All application builds use scripts/build.ps1 and replace bin/snoofer.exe. Exit Snoofer first. No alternate executable names or output directories. scripts/check.ps1 runs validation and calls the same build script. Every external integration a plugin talks to (app, device, OS service, companion DLL, network peer) publishes a connection report: a status-only control with Kind connection, ID <plugin>.app-<name>, Group set to the plugin display name, Label set to the app name, Value set to a short state word, SurfaceOnly and no operations, plus Connection {State, Endpoint, Since, LastActivity, LastError, LastErrorAt, Details}. Set Since when the state changes, keep LastError after recovery, add an Interval detail for polled peers (enables the stale marker) and a Required: Yes detail when the peer must be present, and never include credentials. The Third-party apps screen renders reports without plugin-specific code. Controls may provide ShortLabel for icon-bearing surfaces; keep the full Label meaningful without an icon. Set Hidden while a control has no useful place on control surfaces: the Stream Deck shows a blank key, keeps the binding and ignores input on it. Hidden does not change availability; GUI screens decide for themselves whether to show it (the audio mic controls stay in the GUI while hidden on the deck).
