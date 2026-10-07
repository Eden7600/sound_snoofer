## ADDED Requirements
### Requirement: Optional full-band neural hybrid
The engine selector SHALL offer LocalVQE full-band, using the voice-cleanup neural model for the lower band and WebRTC-processed audio for the upper band at a 48 kHz host rate. Existing engine settings SHALL remain compatible and unchanged by deployment.
#### Scenario: High-frequency voice
- **WHEN** a 48 kHz microphone contains near-end energy above 8 kHz
- **THEN** the hybrid output retains upper-band energy through echo-controlled processing
#### Scenario: Playback echo
- **WHEN** the microphone includes speaker echo
- **THEN** upper frequencies are taken from WebRTC processing, never raw microphone bypass, and lower frequencies retain LocalVQE processing
#### Scenario: Band alignment
- **WHEN** lower and upper branches are combined
- **THEN** measured algorithmic delays are compensated and complementary crossover filters avoid unintended gain doubling
#### Scenario: Unsupported host rate
- **WHEN** the full-band engine receives another sample rate
- **THEN** it passes through and reports Needs 48 kHz rather than claiming full-band processing
#### Scenario: Stream discontinuity or failure
- **WHEN** framing resets or inference fails
- **THEN** both branches reset together or follow the existing bounded recovery/pass-through behavior with no stale upper-band output
#### Scenario: Saved choice
- **WHEN** the user selects the hybrid engine
- **THEN** preference persistence precedes confirmed detachment and replacement through the existing engine lifecycle
