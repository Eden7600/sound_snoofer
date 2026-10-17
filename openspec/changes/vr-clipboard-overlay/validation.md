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

## Original source-only verification boundary

Before deployment authorization, no canonical build, scripts/check.ps1, deployment, host restart, personal configuration edit, visible application launch, desktop input, clipboard read or SteamVR connection was performed. No alternate application executable was produced. The default configuration keeps the plugin disabled.

Task 7 remains open: check actual dashboard discovery, texture orientation/colors, controller hit locations, all seven shortcuts, held presses, primary-hand changes, target focus/refusal, runtime exit/reconnect, native job/pipe cleanup and audio-host continuity. User confirmations count as acceptance evidence.

## Platform limits

The native loader/ABI, process job and real focus behavior still need headset acceptance. Windows SendInput cannot atomically bind delivery to a window; simultaneous external focus changes and same-process/thread window-handle reuse remain limits. OpenVR Overlay initialization can start SteamVR if it exits between process observation and loader initialization. These limits are documented in the user guide and design; fake-only checks do not prove hardware compatibility.

## Authorized deployment

The user then explicitly requested deployment. The canonical scripts/build.ps1 completed successfully, including native companion self-tests, GUI assets and bin/snoofer.exe. The original tray host and controls child exited normally. The live configuration was backed up under an ignored local deployment folder, VR Clipboard was enabled, and every other plugin setting was checked unchanged. The deployed executable’s configuration check exited successfully.

Snoofer was restored with its exact previous configuration argument and its previously open controls window. A child worker remained running, and SteamVR’s server log confirmed its VRApplication_Overlay connection. No clipboard contents were read and no shortcut was sent during deployment. Physical acceptance has not been inferred from process presence.

Deployment review also corrected scripts/stop.ps1 to skip tray messages for the exact private worker command while still waiting on its held process handle. The fake-only scripts/check-stop.ps1 regression failed before the fix and passes with child-first and host-first enumeration. scripts/check.ps1 runs that fixture in a separate PowerShell process so its fake native type cannot affect real shutdown later in the check. No application rebuild is needed for this script-only correction.

## First live input failure and correction

The user reported that buttons did nothing and Target remained Select a desktop window. An explicitly opted-in read-only native Capture test reproduced GetTokenInformation failing with invalid memory access. Replacing byte-aligned token storage with a typed, aligned token-label buffer made the same probe pass and identify the foreground executable. No focus change, input or clipboard read occurred. The alignment invariant has an ordinary ABI regression; the native probe remains skipped unless explicitly enabled.

Related native review replaced the undefined GetProcessTimes exit-time check with GetExitCodeProcess. The read-only Capture check also passed with that correction. Independent Valve-sample comparison corrected button events to use the last MouseMove position and current pointer ownership; coordinate-free click and pointer-invalidation regressions failed before the fix and pass afterward. Scoped unit tests, vet and race checks pass. Real headset delivery is still awaiting the user's next check.

## Corrective deployment

The canonical build through 65cedf4 completed successfully. The existing host, overlay worker and controls child all exited normally. Configuration validation passed; the same configuration argument and previously open controls were restored, without rewriting the live configuration. The replacement worker remained running and SteamVR logged its new VRApplication_Overlay connection.

No shortcut delivery or focus behavior was verified during deployment. Tasks 7 and 9 remain open for real headset feedback; process presence and the successful read-only native Capture probe do not establish end-to-end input acceptance.

## Live target refresh correction

The user reported that Target now updates when closing/reopening the panel but remains stale while it stays open. The worker was already observing foreground targets on every 50 ms tick but only applied them on activation or explicit recapture. Commit c5834cd applies an eligible target change while active through the existing activation/reset/draw path, then discards queued input after checking for Quit. Failed captures retain the last eligible target; unchanged captures preserve gestures and feedback. No new focus calls or clipboard access were added.

The focused worker regression failed before the fix because releasing a held press dispatched the old target after a new target was captured. It now passes, including discarded queued clicks, fresh gesture recovery on the new target, failed-capture retention, unchanged feedback/redraw behavior and Quit during target change. The VR Clipboard package tests, vet and race checks pass.

The canonical build and configuration check passed. Host, worker and controls shut down normally, then restarted with the same configuration and prior controls window. No live configuration was rewritten. Physical verification of active-panel target refresh and shortcut delivery remains pending; the user's latest report does not by itself confirm Paste success.
