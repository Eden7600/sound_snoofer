## ADDED Requirements
### Requirement: Selectable echo engine
The application SHALL persist a choice of WebRTC AEC3, LocalVQE echo-only, or LocalVQE voice cleanup, defaulting to AEC3 for old configurations. The GUI SHALL expose this choice and state neural mono 16 kHz limitations. Strength SHALL only apply to AEC3.
#### Scenario: Saved engine change
- **WHEN** the user selects a valid engine and persistence succeeds
- **THEN** the old hook is bypassed and confirmed detached before destruction and the new engine is configured and attached
#### Scenario: Save or detach failure
- **WHEN** saving fails or detaching cannot be confirmed
- **THEN** no unsafe replacement occurs and the GUI reports the failure
#### Scenario: Missing neural assets
- **WHEN** the selected model or companion cannot load
- **THEN** the mic passes through with an error and the user can select AEC3
### Requirement: Bounded neural audio processing
Neural inference SHALL run off the audio callback with bounded queues, verified weights, sinc resampling and fresh state after configuration or stream changes. Unprocessed channels and playback SHALL remain untouched.
#### Scenario: Continuous stream
- **WHEN** matched microphone and playback callbacks arrive at 16, 32 or 48 kHz
- **THEN** the selected model processes mono 16 kHz hops and returns audio at the host rate with bounded latency
#### Scenario: Deadline or reference failure
- **WHEN** reference stops, queues overflow or processed output misses its deadline
- **THEN** the engine latches an error and passes through rather than waiting or replaying stale data
#### Scenario: Disconnect and reconnect
- **WHEN** the stream ends, changes or restarts
- **THEN** pending audio is invalidated and fresh processing begins with no previous-stream output
#### Scenario: Ambiguous targets
- **WHEN** the audio plugin cannot resolve microphone or speaker targets
- **THEN** cancellation remains idle and reports the existing target reason
