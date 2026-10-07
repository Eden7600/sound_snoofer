# Echo cancellation probe

Run ./tools/aec-probe/run.ps1 from the repository root after scripts/build.ps1. Requires MSVC and the Windows SDK. Test executable and objects stay in .local/aec-probe; no audio device or Voicemeeter registration is used.

The V2 native config carries all eight bus slots. Tests include stereo, phase-opposed stereo, center-only, surround-only, independent six-channel playback with different room responses, 16/32/48 kHz, 256/512/2048-sample buffers and an ordinary buffer-size transition. Synthetic room delay is 40 ms plus device buffering and per-channel offsets; this models output-before-echo causality. Echo reduction must reach 20 dB after convergence; a local 440 Hz tone must stay within 6 dB. This is an acoustic regression test, not perceptual speech-quality certification.

Double-talk runs all three strengths and measures tone preservation via sine/cosine projection, reporting residual energy after removing that tone. The summary's overall energy reduction during double-talk includes the wanted voice and must not be read as echo reduction.

An adversarial per-callback 64-to-2048 size-jitter case verifies exact frame counts, preservation of non-mic channels and the same 20 dB echo reduction threshold. Double-talk also requires at least 10 dB residual reduction in the tone-based estimate. All cases are deterministic synthetic regression checks, not a guarantee under arbitrary device jitter or nonlinear speaker processing.

Missing/repeated output, mismatched size/rate and missing surround reference checks require latched failure and exact mic pass-through. Unsupported 44.1 kHz and explicit bypass preserve all channels. ABI validation is also exercised by SNOOFER_AEC_DLL=bin/snoofer-aec.dll go test ./internal/aec.

Startup coverage reproduces four input callbacks before the first output. Until a real reference arrives, mic output is bit-identical (including aliased buffers); after one second without output, the engine reports missing output. Steady-stream pairing checks remain strict. The suite also calls the production monitor dispatch with stream start/end/change between inserts, then verifies clean recovery. Synthetic surround and double-talk scenarios include startup pre-roll, and the reset case allows the same 20 seconds of adaptation as the baseline.
