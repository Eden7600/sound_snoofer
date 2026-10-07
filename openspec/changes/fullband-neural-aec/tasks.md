# Tasks
- [x] Trace native worker, resampler, AEC3 and GUI engine lifecycle.
- [x] Specify optional 48 kHz hybrid processing and compatibility.
- [x] Measure AEC3 delay and implement/test complementary crossover and alignment.
- [x] Integrate full-band constructor, worker reset/failure behavior and rate guard.
- [x] Add saved GUI engine choice, truthful status and focused tests/render check.
- [x] Run DSP/audio regression probes, Go tests/vet/spec checks, canonical build; commit and restore host.
- [ ] User listening comparison confirms preferable vocal clarity without objectionable echo.

Validation: canonical build and GUI render passed; focused AEC tests, native Go wrapper, go vet, and all 59 OpenSpec items passed. Native probes passed 18 original model/rate/block cases with exact fresh/recovered fixture parity, plus 3 hybrid block sizes, callback recovery, unsupported-rate bypass, and 10 kHz preservation (+0.03 dB). DSP probe measured 430/834 sample branch delays, exact complementary reconstruction, and 17.84 dB synthetic upper-band echo reduction. Tonal double-talk still suppresses 10 kHz by 20.04 dB; subjective improvement remains unverified. Full Go suite has only the pre-existing TestVoiceExample failure caused by the missing config.voice.json. Race testing unavailable with current CGO-disabled toolchain. Host restored with its normal no-argument launch; saved engine selection unchanged.

