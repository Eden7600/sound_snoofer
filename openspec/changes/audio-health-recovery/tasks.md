## 1. Feasibility
- [x] Reproduce the confirmed whole-engine audio stall, for which the user reports engine restart fixes audio; determine whether Remote API calls remain responsive and capture clock-device evidence. See incident-2026-10-05.md; recovery and other backend comparisons remain unverified.
- [ ] Compare connected, silent, muted, failed and recovering readbacks for WDM and ASIO; document which signatures are reliable and which failures remain undetectable.
- [ ] Validate restart effects on Element, recorder transport and owned mute; keep unproven signatures in observation-only mode.
- [x] Build and self-test the opt-in native output callback probe; record a bounded live capture without enabling recovery. Working baseline and user-confirmed disconnect/stall/reconnect-plus-restart captures completed; see incident-2026-10-05.md. Broader health qualification remains outstanding.
## 2. Implementation
- [x] Add health evidence/state and transition grace without slowing numeric setting edits.
- [x] Add narrow standalone engine restart action, policy toggle, recorder guards and persistent attempt budget.
- [x] Add verification/reconciliation with latest intent and ambiguous-result handling.
- [x] Add Controls/tray actions and optional Stream Deck binding with pending/error status.
## 3. Automated verification
- [ ] Cover silence/mute/gates, missing devices, SteamVR exit, ASIO-patched strips, observation failures and target changes.
- [ ] Cover cooldown, restart budgets, app restart, clock changes, duplicate actions, unknown outcomes and recording deferral.
- [ ] Cover concurrent edits, dry-run, preserved assignments/sends/mute, no transport replay and a simulated stuck actor without a second writer.
- [ ] Run tests, vet, Windows build, strict validation and isolated UI smoke.
- [x] Validate callback qualification, reconnect/resume grace, identity ambiguity, live-owner lifecycle, recorder deferral, persisted uncertainty and callback-based recovery verification with targeted tests.
- [x] Exercise the production native monitor against Voicemeeter: register, observe advancing buffers, disable/re-enable and clean up without setters.
## 5. Generalized A1 detection (2026-10-06)
- [x] Capture the SteelSeries A1 stall and post-restart baseline with the callback probe (incident-2026-10-06.md).
- [x] `fix(audio)`: monitor whenever live; target any present A1 device (ASIO presence rules kept); explicit Stalled state; health/report/tone updates. Tests cover WDM/ASIO targets, absent/ambiguous hardware, alert-only versus Auto-recover dispatch and preview.
- [x] `feat(gui)`: Restart audio engine (and Confirm) on Audio and Diagnostics; stall badge tone. Update the GUI check.
- [x] Validate, build, enable Auto-recover through the GUI, relaunch and record. 
- [ ] Hardware: the next real A1 stall is detected and automatically restarted.

## 4. Hardware acceptance
- [ ] Verify actual incident detection and engine recovery on the affected hardware, separately from mock success.
- [ ] Check audible output, Element return, mic mute, Volt A1 and recorder behavior after recovery.
- [ ] Check prolonged silence, ordinary hotplug, VR transitions and suspend/resume cause no spurious restarts.

## 6. Boot without outputs and monitor continuity (2026-10-07)
- [x] `fix(routing)`: under the voice profile, no eligible playback device is a resolved no-output plan that still applies the mic stack and keeps A1; reasons stay diagnostic. Tests cover voice and non-voice profiles.
- [x] `fix(audio)`: automatic restart ignores unresolved routing, defers on pending changes for at most ten seconds after qualification, and names the pending target. Tests cover the deferral, its bound and unresolved items.
- [ ] `fix(audio)`: restart the callback monitor after a stream end or change, at most every five seconds, on the live owner. Tests cover end, change, rate limit and preview.
- [ ] Validate, build and relaunch.
- [ ] Hardware: booting without outputs, then a stall, is restarted automatically; health stays monitored after a device switch.
