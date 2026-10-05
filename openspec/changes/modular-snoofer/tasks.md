## 1. Baseline and ownership map
- [x] 1.1 Read current implementation and overlapping OpenSpec changes; capture existing audio/profile/deck semantics and test baseline without disturbing live audio.
- [x] 1.2 Map each running goroutine, handle, callback and persistence file to its future owner; record preserved audio invariants and superseded VR/config requirements.
- [x] 1.3 Add focused characterization checks only where existing tests do not protect extraction seams.

## 2. Core host and compiled plugin composition (depends on 1)
- [x] 2.1 Add the minimum public registration/lifecycle contracts and explicit built-in composition; remove startup side effects from plugin imports.
- [x] 2.2 Implement dependency validation, ordered start/stop, partial-start cleanup, independent failure reporting and explicit branch retry.
- [x] 2.3 Implement core config envelope, opaque disabled settings, atomic enablement closure edits and clean full restart.
- [x] 2.4 Keep core tray/TUI/diagnostics usable with no plugins and no Voicemeeter; prove disabled factories and validators are never called.
- [x] 2.5 Test missing/cyclic dependencies, duplicate IDs, save failure, partial cleanup, retry and bounded stop failure without overlapping relaunch.

## 3. Audio extraction without routing changes (depends on 2)
- [x] 3.1 Move audio worker composition, settings, saved state and native resources behind audio ownership; reuse routing/controller/adapters.
- [x] 3.2 Preserve OS-thread-affine single writer, save-before-apply, bounded verification, recorder one-shot behavior and stale-action rejection.
- [x] 3.3 Preserve callback recovery, cleanup uncertainty, cooldown/budget journals and Windows default protection independently of VR.
- [x] 3.4 Remove audio assumptions from core state and startup; test audio-disabled startup and audio start/disconnect/failure/reconnect.
- [x] 3.5 Run existing routing, controller, recording, recovery and native tests supported by the installed environment.

## 4. Semantic controls and integrated TUI (depends on 2 and 3)
- [x] 4.1 Add namespaced descriptors, availability/state and typed operations needed by current controls; preserve authoritative provider ownership.
- [x] 4.2 Update private desktop IPC and UI clients to the shared envelope; preserve revisions, bounded queues and no command replay.
- [x] 4.3 Integrate plugin settings and status into one TUI, including enablement closure/restart and retry; retain no Actions section and System restart confirmation.
- [x] 4.4 Extract Windows media plugin; ensure deck/host can run with media and without audio.
- [x] 4.5 Test provider loss/replacement, incompatible operations, stale submissions and pending/unknown/failed presentation.

## 5. VR dependency and audio profiles (depends on 3 and 4)
- [x] 5.1 Extract SteamVR observation entirely into VR, declare audio dependency and implement the narrow profile-policy API.
- [x] 5.2 Add separate Normal/VR priority lists and saved overrides, mode/monitoring, optional final Normal fallback and shared mute/gain/recording.
- [x] 5.3 Implement known-running/stopped and unknown detection behavior, independent directions and hotplug/ambiguity resolution.
- [x] 5.4 Add separate profile UI sections with subdued yet editable overridden Normal settings and explicit effective-profile status.
- [x] 5.5 Test profile enter/exit, inactive edits, shared mute, Off, fallback, ambiguous endpoints, reconnect and whole-app shutdown suppression.
- [x] 5.6 Verify VR disabled causes no SteamVR observation; VR-only disable/restart returns to Normal without shutdown routing changes.

## 6. Stream Deck surface and layout editor (depends on 4; profile bindings depend on 5)
- [x] 6.1 Remove audio-specific action whitelist/state/rendering coupling; bind semantic controls and preserve Studio presentation.
- [x] 6.2 Add stable pages, Home, shared positions, compatible single-control bindings and reserved sixth dial navigation.
- [x] 6.3 Implement integrated TUI page/binding editor with effective preview, offline editing, Save/Cancel and atomic live apply.
- [x] 6.4 Preserve missing-control bindings/labels as inert Unavailable; invalidate old-generation input across edits, page changes and provider changes.
- [x] 6.5 Seed existing personal/default key and gain assignments; preserve serial-specific layouts and initial-held-key suppression.
- [x] 6.6 Test wrap/Home/reorder/delete, final-page rejection, collision validation, save failure, unavailable restoration and reconnect rendering without replay.

## 7. Rename, external author path and manual changeover (depends on 2-6)
- [x] 7.1 Rename application/commands/build outputs to Snoofer; keep repository folder and tray-first no-console launch.
- [x] 7.2 Build full and core-only compositions; ensure core-only avoids audio native prerequisites and vendor DLL redistribution.
- [x] 7.3 Document and compile a minimal external-plugin composition using public contracts; prove uncompiled and disabled paths.
- [x] 7.4 Back up personal config, saved choices, safety journals and matching old executable/native companion inside the repo.
- [x] 7.5 Manually convert personal settings/layouts and operational state; verify no recovery budget or uncertain command marker was reset.
- [x] 7.6 Update README, application/troubleshooting guides, AGENTS architectural scope and OpenSpec context; reconcile superseded requirements and delete replacement-created dead paths.
- [x] 7.7 Record launch, build and rollback commands; verify the converted config offline before replacing the running application.

## 8. Automated release validation (depends on 7)
- [x] 8.1 Run gofmt, full Go tests and vet; run race checks only if supported and record any limitation.
- [x] 8.2 Run native callback checks for the audio build, full/default and core-only build checks, and strict OpenSpec validation.
- [x] 8.3 Exercise the plugin enablement matrix: none, audio only, media plus deck, full; invalid VR-without-audio closure; unavailable plugin bindings.
- [x] 8.4 Review dependency direction, disabled resource counts, cleanup and new abstractions; remove unnecessary indirection and duplicated behavior.

## 9. Real-device acceptance (depends on 8; never substitute simulated tests)
- [ ] 9.1 Verify tray launch, controls child, settings persistence, clean exit/restart and no blank terminal with the renamed executable.
- [ ] 9.2 Verify Normal routing, Volt channel/A1 reservation, Element processing, mute/Off, gains, playback and Windows defaults.
- [ ] 9.3 Exercise Volt power-off stall, reconnect and permitted recovery; distinguish callback health from audible output and preserve restart limits.
- [ ] 9.4 Start/stop SteamVR and disconnect/reconnect VR audio; verify independent profile choices, Normal edits while overridden, fallback and shared mute.
- [ ] 9.5 Verify deck pages, sixth-dial wrap/Home, shared bindings, offline edits, unavailable controls and reconnect without accidental commands.
- [ ] 9.6 Disable/re-enable each plugin and dependency closure; confirm no disabled polling/handles and no recorder stop or route reset at shutdown.
- [x] 9.7 Verify rollback artifacts and commands, record actual results and leave every unperformed hardware item unchecked.



## 10. Cleanup and commit
- [x] 10.1 Remove the legacy standalone TUI/CLI adapter while retaining audio-worker regression coverage in its owner package.
- [x] 10.2 Remove obsolete loose build artifacts; preserve personal state, current running application and rollback artifacts.
- [x] 10.3 Run automated checks, inspect the staged scope, and commit the accumulated work with a breaking-change description.
- [x] 10.4 Restore Bubble Tea styling; verify resize, picker, confirmation, override and no-color presentation before rebuilding.
