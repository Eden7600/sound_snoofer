# Validation — 2026-10-06

Implementation commit: 17d2c5d. Design commit: 58ed14c.

- Full Go tests and go vet passed. Native Go ABI test passed against the final DLL, including invalid config and obsolete export checks.
- Strict OpenSpec validation and git diff --check passed.
- Canonical scripts/build.ps1 passed, including monitor ABI/pass-through probe. Prior Snoofer exited normally via scripts/stop.ps1; Voicemeeter and Element were not stopped.
- tools/aec-probe/run.ps1 passed. All runs accounted for exactly 2000 frames over 20 seconds; unrelated channels remained bit-identical. Stereo, phase-opposed stereo, center-only and surround-only: 36.4 dB reduction at 48 kHz. Independent six-channel playback: 35.3 dB. 2048-sample callbacks and extreme 64-to-2048 jitter: 35.8 dB. Normal 256-to-512 transition: 34.5 dB. At 32/16 kHz: 33.5/30.5 dB.
- Local tone in playback pauses changed by about -0.4 dB at 48 kHz. Double-talk at all three strengths: local tone about -1.2 dB; residual reduction about 12.4 dB using tone projection. Assertions require 20 dB echo-only reduction, voice within 6 dB, and 10 dB double-talk residual reduction. These synthetic signals do not certify speech intelligibility or perceptual quality.
- Missing/repeated output callbacks, mismatched size/rate and absent surround reference all latched failure and passed mic audio through. 44.1 kHz and explicit bypass passed all channels unchanged.
- Race detector unavailable: CGO_ENABLED=0. No toolchain changes. No GUI behavior changed; GUI tests were not rerun.
- Restarted the prior no-argument host using bin/snoofer.exe and existing configuration; PID 59660 was running with snoofer-aec.dll and snoofer-audio-monitor.dll loaded. This verifies launch/loading, not audible cancellation or live callback health. No personal configuration writes. Existing scripts/check-desktop.cjs changes and untracked CLAUDE.md were excluded.

## Remaining acceptance
Listen to actual speaker upmix and discrete 5.1 with near-end speech and playback together. Verify real-time CPU/latency under load, nonlinear speaker DSP, room changes and absence of clipping. The fixed shim framing delay is 10 ms; end-to-end device/engine latency is additional. Eight-channel processing costs more than mono. The more permissive delay-confidence setting retains lag aggregation but needs real-world confirmation under unrelated near-end noise.

## Engine alternatives
Reviewed primary sources; no replacement was installed or benchmarked. Retain AEC3 for this change because fixing its inputs and alignment now passes the multichannel workload.

- [SpeexDSP](https://github.com/xiph/speexdsp/blob/master/include/speex/speex_echo.h) exposes multi-microphone/multi-speaker cancellation and a tunable filter tail. It is a feasible comparison target, but its API documentation does not establish better quality for this room or 5.1 setup.
- [DTLN-aec](https://github.com/breizhn/DTLN-aec) supplies pretrained neural echo-cancellation models. Its [reference runner](https://raw.githubusercontent.com/breizhn/DTLN-aec/main/run_aec.py) requires 16 kHz single-channel mic and playback inputs; adopting it directly would sacrifice the independent surround references and full-band input. It is a research comparison candidate, not a drop-in upgrade.
- [Current WebRTC neural residual estimator](https://webrtc.googlesource.com/src.git/+/526e228d25f83b1023760d3835f33d622c7b9f5f/modules/audio_processing/aec3/neural_residual_echo_estimator/neural_residual_echo_estimator_impl.cc) provides an integration path using a TFLite model and runtime. This is a promising follow-up to investigate, but not a switch present in the bundled standalone v2.1 sources; model availability, licensing, integration and measured benefit would need validation.
- [NVIDIA Audio Effects SDK](https://docs.nvidia.com/maxine/afx/latest/index.html) marks its acoustic echo-cancellation effect deprecated. Its room dereverberation is a different operation, so it is not the recommended replacement.
