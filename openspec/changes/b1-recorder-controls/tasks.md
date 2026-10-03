# Tasks

## 1. Recording configuration and saved choices

- [x] 1.1 Add optional recording profile with mic/computer inclusion, Pre/Post tap and explicit virtual computer-source membership; verify default-Off, Potato/voice prerequisites and invalid/duplicate/AUX rejection tests.
- [x] 1.2 Extend saved intent compatibly: valid existing voice-only sidecars retain choices and receive disabled recording defaults; verify save/reload/reset, malformed new fields, missing profile and failed/concurrent save cases.
- [x] 1.3 Add configuration examples and document B1 ownership, source definitions, native directory/format setup and lack of auto-start; verify examples load through plan and TUI configuration paths.

## 2. B1 matrix and transition isolation

- [x] 2.1 Compile all-strip B1 ownership with effective mic Pre/Post and independent computer capture; verify the full inclusion/mode/source matrix including voice disabled, Direct/Post inactive, unavailable mic and absent playback.
- [x] 2.2 Separate recording and voice transition groups; verify recording-only edits preserve B2/B3/monitor sends, while mic/device changes gate affected consumers before repatching and failed disables block new B1 enables.
- [x] 2.3 Add ambiguity, disconnect/reconnect, drift and unchanged-state tests; document live source edits and short recording gaps without transport restart.

## 3. Native recorder observation and transport

- [x] 3.1 Add optional recorder snapshots and strict setter allowlists; verify supported parameters against the installed DLL read-only, test invalid names/ranges and unknown/error state without breaking legacy snapshots.
- [x] 3.2 Implement stopped-only bus/stereo preparation and playback-send protection; verify B1 is solely armed, B1/B2/B3 tape sends are off, format/directory and A sends are preserved, and incompatible active recording blocks preparation.
- [x] 3.3 Implement explicit Start/Stop with writer ownership, fresh-state guards, one in-flight command and bounded confirmation; test repeated Start, pending routing, zero eligible sources, pause/play states, partial preparation and stop despite route conflict.
- [x] 3.4 Inject command timeout, connection loss and external transport changes; verify no automatic retry/restart, stale command rejection and accurate unknown/recovered status. Document parameter names and transport limitations.

## 4. TUI controls

- [x] 4.1 Add recording inclusion/tap rows and explicit Start/Stop controls with inactive/blocked/status feedback; verify keyboard behavior, scrolling/resizing and source-state persistence in dry-run without mixer or transport writes.
- [x] 4.2 Serialize source edits and transport through the existing worker; verify external start/stop observation, save failure, reload, writer conflict and quit/dry-run leaving native transport unchanged with a visible continuation notice.
- [x] 4.3 Update README controls and run an interactive dry-run terminal smoke test covering selection edits, reload, unavailable Start and clean exit; record the results.

## 5. Integration and real-device acceptance

- [x] 5.1 Run go test ./..., go vet ./..., Windows build and strict OpenSpec validation; record passing regression results separately from hardware acceptance.
- [x] 5.2 Back up local config/sidecar and B1/recorder settings, then preview profile adoption with both inclusions off; verify existing voice and playback routes are unchanged.
- [ ] 5.3 Run an explicit short native Start/Stop test and verify the resulting recording file exists and is playable in the configured directory; verify no automatic capture on application restart.
- [ ] 5.4 Listen to mic-only Pre, mic-only Post with an audible Element effect, computer-only and combined recordings; verify Direct/Post becomes inactive without dry substitution and muted speaker playback does not remove computer capture.
- [ ] 5.5 Verify source edits during recording, headphone/Volt disconnects, native external Start/Stop and reconnect behavior; inspect that tap changes do not double the mic or unnecessarily interrupt Discord, and mark unperformed checks pending.
