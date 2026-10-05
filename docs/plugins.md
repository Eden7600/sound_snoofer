# Snoofer plugins

Snoofer is a Windows host with optional compiled audio, VR, Stream Deck and Windows media plugins. The repository directory remains sound_snoofer and the Go module remains sound-snoofer. No runtime loader, sandbox or process isolation is provided: plugins are trusted Go code.

Build from the repository:

```powershell
./scripts/build.ps1
./scripts/build.ps1 -Tags core -Output bin/core/snoofer.exe
./scripts/build.ps1 -Tags no_audio -Output bin/media-deck/snoofer.exe
```

Available exclusions: no_audio (also excludes VR), no_vr, no_streamdeck and no_media. Core excludes all built-ins and does not build or load the audio companion. The default build includes snoofer-audio-monitor.dll; the Voicemeeter vendor DLL is never distributed.

Launch bin/snoofer.exe. Its adjacent configuration is snoofer.json. A new installation creates disabled plugin entries with inert defaults. Existing personal configurations are converted manually; the application does not attempt migration. --config selects another envelope, --dry-run prevents audio writes, and --check validates the envelope, enabled plugin settings and dependencies without starting devices.

In the integrated controls window, Tab switches between Controls and Plugins. Enter edits selections, text and toggles; +/- adjusts numeric controls. Plugin changes display their dependency closure, then require y to save and restart. r retries failed branches. Closing controls leaves the host running.

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

## Stream Deck layout

Pages contain 36 keys and five assignable dials. The sixth dial always rotates pages and presses Home. Shared bindings reserve positions across every page. Clear a page binding before assigning its position globally; collisions are rejected.

The Stream Deck configurator is part of Controls: select a page, rename/create/reorder/delete, choose Home, choose a key/dial and assign a compatible semantic control. Shared toggles which assignment scope you edit. Effective position previews the merged binding. Save applies the complete draft live; Cancel discards it. Save/Cancel before switching serial layouts.

Missing providers retain their binding IDs and labels and render Unavailable. Hardware may be disconnected while editing. No macros, arbitrary artwork imports, folders or app-specific page profiles are included.



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
