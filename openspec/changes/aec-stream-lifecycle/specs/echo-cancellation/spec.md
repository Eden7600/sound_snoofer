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
