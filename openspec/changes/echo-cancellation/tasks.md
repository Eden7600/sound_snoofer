# Tasks
Each numbered block is one reviewable commit.

## Implementation
- [x] 1. `docs(aec)`: design the echo cancellation plugin, vendored engine and shared callback.
- [x] 2. `chore(third_party)`: vendor webrtc-audio-processing v2.1 and the abseil subset, with licences and `third_party/README.md`.
- [x] 3. `build(aec)`: `scripts/build-aec.ps1` (explicit source list) and the C++ shim producing `bin/snoofer-aec.dll`; wired into `scripts/build.ps1`.
- [x] 4. `test(aec)`: offline probe with synthetic echo (ERLE, pass-through, re-framing, 44.1 kHz bypass).
- [x] 5. `feat(voicemeeter)`: monitor insert hook and modes; `SetCallback(monitor, insert)` with re-registration. Includes tests.
- [x] 6. `feat(audio)`: `EchoTargets` and `SetEchoInsert`. Includes tests.
- [x] 7. `feat(aec)`: the plugin (modes, Auto, strength, status, meter) with a fake engine in tests; deck icon.
- [x] 8. `feat(gui)`: Echo cancellation card on the Audio screen. The GUI check covers it.
- [x] 9. Validate: Go, offline probe, GUI and desktop checks, OpenSpec, the canonical build; enable for the personal config in Auto; relaunch.

## Hardware acceptance
- [ ] 10. With speakers and a call or recording:
  - no echo is audible at the far end;
  - local speech stays natural;
  - headphones in Auto pass through;
  - Off restores the exact original path.

## Recovery on a new audio stream (2026-10-07)
- [x] 11. `docs(aec)`: design one reset per new stream and the failure reason.
- [x] 12. `fix(aec)`: native `AECResetFailure` and `AECReadFailure`, the output-insert generation guard, the Go binding and the offline probe. The probe and the opt-in native test cover reset and reason.
- [x] 13. `fix(aec)`: publish `EchoTargets.Stream`; the plugin resets once per new stream and shows the reason. Fake-engine tests cover same-stream latching, a new-stream reset and a failure seen before the new count.
- [x] 14. Validate, build and relaunch. Go tests and vet pass; the native ABI test and the offline probe pass against the new DLL (each fault reports its reason, then a reset resumes processing; echo reduction unchanged at 30.5–36.4 dB). Snoofer relaunched with no arguments.
- [ ] 15. Hardware: after an engine restart or device switch, echo cancellation returns to Active.
