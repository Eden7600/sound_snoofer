## ADDED Requirements
### Requirement: Lower latency with observable worker timing
Neural processing SHALL use event-driven wake-ups and a tested reduced buffer, report its processing latency estimate, and expose queue/worker timing and interruption counts.
#### Scenario: Audio arrives
- **WHEN** a microphone/reference packet is published
- **THEN** the worker is signaled without callback waiting or polling sleep
#### Scenario: Timing observation
- **WHEN** neural timing is displayed
- **THEN** peak queue age and packet processing duration are distinguished from end-to-end latency and gap/underrun counts remain available across resets
### Requirement: Neural-controlled hybrid upper band
The hybrid SHALL smoothly gate its echo-controlled upper band according to neural output activity while preserving the neural low band.
#### Scenario: Neural silence
- **WHEN** neural output becomes quiet
- **THEN** upper-band gain fades closed without a raw audio bypass
#### Scenario: Neural voice
- **WHEN** neural output exceeds the gate opening threshold
- **THEN** the upper band fades open without additional lookahead latency
