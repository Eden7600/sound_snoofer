## ADDED Requirements
### Requirement: Bounded neural callback resynchronization
The neural bridge SHALL discard unmatched audio and stale queued results after an isolated missing or extra playback callback, pass the microphone through during fresh warm-up, and resume processing only with new matching reference audio. This recovery SHALL replace permanent failure on the first callback discontinuity for both LocalVQE engines.
#### Scenario: Missing output midstream
- **WHEN** another input arrives before the previous input received its output callback
- **THEN** pending audio and processing history are invalidated, the microphone passes through, and fresh valid pairs restore Active without user intervention
#### Scenario: Extra output midstream
- **WHEN** an output has no matching input or mismatched rate or size
- **THEN** that output is discarded and the next valid input starts fresh framing
#### Scenario: Persistent discontinuities
- **WHEN** a recovery episode lasts two seconds of input audio without one continuous second of valid pairs
- **THEN** missing-output failure latches and the mic continues passing through
#### Scenario: Reference disappears
- **WHEN** reference remains absent for one second after restart
- **THEN** the existing missing-output timeout latches
#### Scenario: Independent engine fault
- **WHEN** neural inference, output deadlines, sample validity or queue capacity fails
- **THEN** the existing latched failure behavior remains in effect
#### Scenario: Explicit stream boundary
- **WHEN** configuration changes or a stream lifecycle event occurs
- **THEN** stale recovery counters and queued audio cannot cross into the new stream
