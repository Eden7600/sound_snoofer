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
- [ ] 9. Validate: Go, offline probe, GUI and desktop checks, OpenSpec, the canonical build; enable for the personal config in Auto; relaunch.

## Hardware acceptance
- [ ] 10. With speakers and a call or recording:
  - no echo is audible at the far end;
  - local speech stays natural;
  - headphones in Auto pass through;
  - Off restores the exact original path.
