# Validation — 2026-10-16

## Completed without touching live applications

- Guarded desktop shortcut unit tests pass. They inject a fake OS boundary; no Capture/Send native implementation is invoked. Coverage includes all seven balanced key batches, target/process/thread/birth changes, locked/privileged targets, held keys, bounded focus restoration, cancellation, partial-input release-only cleanup and Windows INPUT layout.
- OpenVR adapter unit tests pass using a fake driver. Coverage includes pinned function-table/event layout, raw image packing, installed-runtime path parsing, event conversion/device identity, primary-controller cancellation, bounded queues, thumbnail isolation and cleanup on startup failures. The installed loader path was inspected as a file only; the DLL was not loaded.
- Panel/worker/supervisor tests pass using fake controller events, process factory and runtime detection. Coverage includes disabled defaults, inert preview, exact worker arguments, startup ownership gate, bounded status messages, normal/error/EOF cleanup and suppression of immediate worker respawn.
- Review regressions cover leaving a button and returning, controller switches, stale gestures after slow dispatch, recapture followed by a queued action, and Quit arriving in a discarded batch. The leaving/controller and slow-dispatch/recapture regressions failed before their fixes.
- The 1024 × 512 synthetic panel PNG was rendered and visually inspected. Labels, target, feedback and eight large hit areas fit; hovered Paste has the cyan outline. Artifact: C:/Users/Eden7600/Documents/Codex/2026-10-14/rev/outputs/vr-clipboard-preview.png.
- go test ./internal/desktopinput ./internal/vroverlay ./plugins/vrclipboard -timeout 30s — pass.
- go vet ./internal/desktopinput ./internal/vroverlay ./plugins/vrclipboard ./cmd/snoofer — pass.
- go test -race ./internal/desktopinput ./internal/vroverlay ./plugins/vrclipboard -timeout 30s — pass using the previously approved portable MinGW compiler, with only process-local environment variables.
- The two pure cmd/snoofer registration/argument tests pass: TestVRClipboardIsIndependentAndDisabledByDefault and TestClipboardWorkerInvocation. No main function or app.Run invocation was executed.
- Strict OpenSpec validation passes. New/changed Go sources are gofmt-formatted.

The command dispatch path preserves redirected pipes and skips console creation when valid streams exist. Early worker startup errors use stderr instead of the application's fatal dialog. Package initialization contains no native execution.

## Deferred by the user's current-session restriction

No canonical build, scripts/check.ps1, deployment, host restart, personal configuration edit, visible application launch, desktop input, clipboard read or SteamVR connection was performed. No alternate application executable was produced. The plugin stays disabled in existing configurations.

Task 6 remains open: after permission to interrupt, use the canonical build and restore the exact launch configuration, then opt in and check actual dashboard discovery, texture orientation/colors, controller hit locations, all seven shortcuts, held presses, primary-hand changes, target focus/refusal, runtime exit/reconnect, native job/pipe cleanup and audio-host continuity. User confirmations count as acceptance evidence.

## Platform limits

The native loader/ABI, process job and real focus behavior still need headset acceptance. Windows SendInput cannot atomically bind delivery to a window; simultaneous external focus changes and same-process/thread window-handle reuse remain limits. OpenVR Overlay initialization can start SteamVR if it exits between process observation and loader initialization. These limits are documented in the user guide and design; fake-only checks do not prove hardware compatibility.
