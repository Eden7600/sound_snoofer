# Neural AEC probe

Build the application with scripts/build.ps1, then run tools/neural-aec-probe/run.ps1.
The probe does not connect to Voicemeeter or change audio routing. It uses both
hash-verified LocalVQE models, nine rate/block combinations each, paced at real
time, and tests pre-roll, aliased storage, untouched channels, stream reset,
missing callbacks, accelerated-input deadline failure, and unsupported-rate/bypass.
It also compares the callback bridge with direct streaming inference on the
upstream fixture using identical downmix arithmetic; max error must be <1e-4.
Reported tone attenuation is diagnostic only, not a speech quality benchmark.

2026-10-07: both models passed all cases; fixture error was 0.00000000 for both.
Worst observed callback time was 20.4 microseconds. Neural work runs on a separate
thread with 64 ms plus one device block of scheduling delay, followed by the
model's 16 ms hop delay and sinc resampling delay. The GUI reports the total
(approximately 93 ms for a 512-sample/48 kHz device). The neural path is mono,
16 kHz, with an averaged eight-channel reference; it does not independently model
surround speakers. Real-room double-talk and voice-quality listening remain
necessary before choosing it over AEC3.
