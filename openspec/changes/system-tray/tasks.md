## 1. Implementation
- [x] 1.1 Separate worker lifetime and attached TUI rendering; preserve standalone TUI.
- [x] 1.2 Implement bounded child IPC, independent controls lifecycle and Windows console integration.
- [x] 1.3 Implement tray menu/status, default launch and instance signaling.
- [x] 1.4 Add GUI build script and user documentation.
## 2. Verification
- [x] 2.1 Add lifecycle, IPC, command and failure-path tests; run tests and vet.
- [x] 2.2 Validate OpenSpec and build Windows executable; attempt race validation.
- [x] 2.3 Verify preview launch and controls reopening; verify native tray registration and context shutdown using the opt-in desktop integration test.
- [x] 2.4 User confirmed tray visibility and colour rendering.
- [ ] 2.5 Manually verify the tray menu Quit click.
- [x] 2.6 Add readable attached-console font and mascot ICO; retain original artwork.
## 3. Hardware acceptance
- [ ] 3.1 Confirm live routing and native recording continue while controls are closed.
- [ ] 3.2 Confirm tray restores after Explorer restart and real hotplug recovery while hidden.

## Evidence
- Full Go test suite and vet passed; GUI-subsystem executable built with scripts/build.ps1.
- Strict OpenSpec validation passed.
- Opt-in desktop smoke test ran for 3 seconds against a distinct .local/tray-lifecycle profile: actual tray initialization, Shell_NotifyIconGetRect registration check, worker cancellation and exit passed.
- Separate .local/tray-smoke preview launched with controls absent; duplicate launch opened the TUI; a later launch reopened controls while the same tray parent remained alive.
- Attached-Q unit test confirms it cancels only the view and the worker continues publishing; actor cancellation closes its backend once.
- Race test attempted but unavailable: CGO is disabled. No toolchain installed.
- Development shell supplies NO_COLOR=1. Relaunched final preview without that override; deliberate user NO_COLOR remains supported.

- User confirmed both the tray icon and TUI colours were visible. Added explicit Cascadia Mono / Consolas font selection in response to font feedback.
