# Implementation record

All code, builds, test executables, external-module fixtures and backups are inside the existing repository.

## Ownership

| Resource / state | Owner | Release / persistence |
| --- | --- | --- |
| Tray event, icon and controls child | app | Child pipes close on detach; child joined or terminated; event released before restart |
| Plugin graph and core config envelope | snoofer.Host | Cancellation before reverse-order Stop; cleanup error prevents relaunch |
| Semantic snapshots and requests | snoofer.Controls | Mutex-serialized registration/dispatch, bounded provider queues, revisions invalidate stale input |
| Native audio login, writer lease, controller and callback bridge | audio / existing control worker | Single pinned OS thread; callback teardown before ownership release; uncertain cleanup is returned to host |
| Windows defaults COM worker | audio / windowsaudio | Cancel/revoke/ack; no stale-default restoration at exit |
| SteamVR process enumeration timer | vr | Only enabled VR starts it; cancellation joins it; no policy withdrawal during whole-app shutdown |
| Deck HID read/render workers | streamdeck surface | Cancellation joins reader, bounded queues, device close, bounded display clear |
| Deck draft and active layout | streamdeck plugin | Draft is independent; validated atomic Save precedes live replacement |
| Windows media input queue | media | Bounded queue, cancellation and join; no inferred playback state |
| Static plugin settings | core envelope / enabled validator | Atomic replace and revision comparison; disabled payloads stay opaque |
| Audio saved choices | explicit audio state_path + .state.json | Existing save-before-apply and state revision checks |
| Mute and recovery journals | same state_path + .mutes.json / .recovery.json | Paths preserved across rename; existing recovery limits and uncertain-command markers retained |
| Passive VR slot ownership | audio state_path + .vr-ownership.json | Persisted before policy activation, allows Normal reconciliation to clear old monitoring sends after VR is disabled |

The old cmd/sound-snoofer and internal/desktop composition are replaced by cmd/snoofer and app. Old audio-specific deck dispatch, queue and renderer adapters are removed; the device protocol and Studio raster drawing remain. The audio maintenance CLI retains devices/plan/apply/watch. Its retired TUI adapter and internal/tui are removed; the worker tests now live in internal/control. The unified Bubble Tea app is the sole interactive UI.

## Concrete API choices

Public contracts are in snoofer; built-in factories are in plugins/audio, plugins/vr, plugins/streamdeck and plugins/media. Core is independent of these packages. Explicit composition files use build tags core, no_audio, no_vr, no_streamdeck and no_media. A no_audio build also omits VR.

An optional plugin-owned Command callback retains audio maintenance commands behind the audio plugin. It is dispatched only for an enabled provider. The normal controls child remains generic and uses inherited private pipes.

Profile choices are resolved for each fresh routing observation. VR status comes through audio.SetVRPolicy, never native process observation. Normal and VR preferences remain separate; shared mute/gain/recording state remains outside profile selection.

Passive slot ownership is audio cleanup data, not executable VR functionality. Changing a previously owned VR hardware strip is rejected with an explicit cleanup instruction instead of forgetting the old assignment and monitor sends. Current UI does not offer a hardware-strip editor.

## Validation recorded

- Initial regression suite passed before extraction.
- Full Go tests passed after composition and device-driver cleanup.
- go vet passed on the modular composition.
- Default, core-only and no_audio (media/deck) builds succeeded.
- Core-only dependency inspection contains no audio, Voicemeeter, Stream Deck or built-in plugin packages.
- An external Go module built successfully against public app/snoofer contracts.
- Core-only Windows tray startup/shutdown smoke passed in preview mode.
- Real-device preview lifecycle passed with audio, media and Stream Deck enabled and VR disabled; reported no observed engine fault.
- Native callback lifecycle passed: processing and synchronization counts advanced from 0 to 101, with stop/re-enable cleanup verified.
- Native C self-check passed ABI, pass-through, aliasing, synchronization, invalid buffers and lifecycle cases.
- Plugin matrix validation covers none, audio, media+deck, full and invalid VR-without-audio.
- New tests cover lifecycle order, independent failure, closure, cleanup deadlines, stale controls, atomic config, profiles/fallback/ambiguity, shared mute, inactive edits, layout collision/Home/wrap and generic rendering/transport.
- Race detector is unavailable with the installed CGO_ENABLED=0 environment. No toolchain change was made.

Physical Volt disconnect/reconnect, audible routing, SteamVR transitions with a configured headset and manual deck navigation remain real-device acceptance tasks. Preview/API evidence does not claim those trials passed.

## Personal changeover

New executable: bin/snoofer.exe.
New envelope: bin/snoofer.json.
Audio state_path: config.json relative to bin, retaining existing sidecar identities.
Enabled: audio, streamdeck, media.
VR is compiled but disabled because the existing personal configuration contains no headset matchers. Its prepared settings explicitly fall back to Normal; no endpoint names were guessed.

Rollback bundle: .local/rollback/modular-20261005-110252.
The bundle contains the prior executable and matching native companion, personal configs and state snapshots, hashes, and superseded source files. Original bin/config.json and its sidecars were not replaced during conversion.

To launch the new application, run bin/snoofer.exe. To roll back, quit Snoofer cleanly, restore the prior executable and matched companion from the bundle, and launch sound-snoofer.exe with its original config. If new-version choices have since been saved, restore the compatible saved-choice snapshot deliberately. Keep the latest compatible recovery and mute journals so rollback does not reset cooldowns, uncertain-command markers or native ownership evidence. Back up current state before any restoration.


## Final checks (2026-10-05)

- scripts/check.ps1 passed the full Go suite, vet, default GUI/native build and all 31 strict OpenSpec changes. CGO race checking remains unavailable.
- The full four-plugin composition started and stopped successfully in preview using copied state under .local/modular-preview; all four branches reported Running. This does not establish VR device routing.
- Controls-console regression passed with CREATE_NEW_CONSOLE and DETACHED_PROCESS, including repeated console acquisition and preserved inherited IPC pipes.
- The core tray startup/shutdown smoke passed after the startup watchdog was restored.
- Default, core, no_audio and external-module builds passed against the final public API.
- Deck actor checks cover failed Save retaining active layout/draft, successful live apply, stable page reorder, Home, stale page input, provider loss/restoration and no replay. HID decoding separately covers held keys.
- Profile regression checks cover independent microphone failure reporting, exhausted output slots, rejection of unowned playback overrides and passive output/input ownership after VR is disabled.
- Every rollback-manifest hash matched. Original bin/config.json, .state.json, .mutes.json and .recovery.json still match their backup hashes.

Implementation and automated tasks 1–8 are complete. Tasks 9.1–9.6 remain unchecked: the automated console/tray and preview checks are supporting evidence, not substitutes for the complete interactive and audible acceptance procedures.

## Commands

Run these from C:\Users\Eden7600\Documents\sound_snoofer:

```powershell
.\scripts\build.ps1
.\bin\snoofer.exe
```

For binary rollback, first quit Snoofer from the tray and confirm its process has exited. Then restore the matched pair:

```powershell
Copy-Item .local/rollback/modular-20261005-110252/bin/sound-snoofer.exe bin/sound-snoofer.exe -Force
Copy-Item .local/rollback/modular-20261005-110252/bin/snoofer-audio-monitor.dll bin/snoofer-audio-monitor.dll -Force
.\bin\sound-snoofer.exe
```

These copy commands were checked against existing source/destination paths but were not executed over the new build. Original config/state remains in place. After subsequently saving new-version choices, back up current state and deliberately restore only incompatible saved choices; retain current compatible safety journals. Never reset recovery budgets to make rollback work.

## Cleanup and visual restoration

Removed the retired internal/tui package and CLI launch adapter. Moved its domain regression checks (lifecycle, choices, recording, batch saves, disconnected selection, reload, latency and notices) into internal/control. The core Bubble Tea shell now restores blue section headings, a framed viewport, aligned controls, highlighted tabs/selection and persistent status/help. Presentation tests cover narrow terminals, scrolling pickers, overrides, NO_COLOR and hidden-edit prevention.

After the user exited Snoofer, the default executable and native companion were rebuilt in bin. Full tests, vet, native self-checks, all 31 strict OpenSpec validations, core/no_audio builds and the controls-console regression passed. No dependency versions changed; the retired direct terminal dependency is now transitive.

Removed old loose executables, test binaries, smoke directories, temporary modules/scripts, generated previews, build products and Go caches from .local. Only the rollback bundle remains. Original stall capture logs were retained inside its probe-evidence directory, and the incident report paths were updated. Personal configs and operational sidecars were not removed or restored. Hardware acceptance remains separate and unchecked.
