## ADDED Requirements
### Requirement: Independent playback references
Snoofer SHALL preserve all eight slots of the selected playback bus independently for echo cancellation, without averaging away channel content, and SHALL preserve the user's current mode and strength.
#### Scenario: Discrete surround or stereo speaker upmix
- **WHEN** the playback bus contains stereo, center-only, surround-only or discrete 5.1 audio
- **THEN** every bus channel is available to the echo canceller independently, including when front left and right are identical or silent
### Requirement: Lossless bounded reframing
The shim SHALL process supported callback sizes incrementally with fixed 10 ms capture framing latency and no silently dropped samples. It SHALL validate paired input/output timing and native processing results.
#### Scenario: Non-frame-aligned callback buffers
- **WHEN** valid paired callbacks use supported rates and buffer sizes not divisible by 10 ms
- **THEN** all complete frames are processed and non-mic channels remain bit-identical
#### Scenario: Invalid reference or callback discontinuity
- **WHEN** a reference channel is missing, callbacks are unpaired, rates or sizes disagree, or processing fails
- **THEN** failure is reported and subsequent microphone audio passes through without modifying playback
