## ADDED Requirements
### Requirement: Native stream boundary invalidation
The monitor SHALL notify AEC synchronously of stream start, end and change. AEC SHALL discard old pairing and reference history before processing the next stream and SHALL preserve same-stream failure latching.
#### Scenario: Stream interrupted after microphone input
- **WHEN** a stream lifecycle event occurs between input and output inserts
- **THEN** stale output is ignored and the next input rebuilds AEC without a missing-output failure
#### Scenario: Failure followed by stream boundary
- **WHEN** AEC has latched a failure and a stream lifecycle event arrives
- **THEN** AEC becomes inactive until the next valid input and resumes with fresh framing
#### Scenario: Persistent same-stream fault
- **WHEN** repeated input callbacks occur without output or a lifecycle event
- **THEN** AEC reports the missing output and passes microphone audio through

### Requirement: Bounded reference startup
AEC SHALL pass microphone audio through until its first valid playback reference, tolerate input pre-roll for at most one second of input samples, and enforce strict pairing after reference startup.
#### Scenario: Voicemeeter input pre-roll
- **WHEN** multiple input callbacks precede the first output
- **THEN** the microphone passes through bit-identically without a false engine failure and cancellation begins after the reference arrives
#### Scenario: Reference never arrives
- **WHEN** one second of input samples arrives without a valid output
- **THEN** AEC reports missing output and remains in pass-through
