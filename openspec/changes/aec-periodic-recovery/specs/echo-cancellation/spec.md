## ADDED Requirements
### Requirement: Tolerant callback recovery
All AEC bridges SHALL discard mismatched timelines and recover from transient callback gaps, tolerating up to ten seconds of unstable pairing and ending recovery after 250 ms of consecutive pairs. Five seconds without reference SHALL fail open.
#### Scenario: Intermittent callbacks
- **WHEN** callback pairs resume after a transient gap
- **THEN** processing resumes from fresh framing without stale audio
#### Scenario: Prolonged gaps
- **WHEN** gaps exceed the recovery budget
- **THEN** the engine reports failure and passes microphone audio through
### Requirement: Automatic and manual retry
The plugin SHALL retry latched native failure every 15 seconds while active and hooked, and SHALL offer a non-persistent Retry command in the GUI for eligible failures.
#### Scenario: Periodic recovery
- **WHEN** a native failure persists on the same stream
- **THEN** the worker resets it after 15 seconds and waits for observed activity
#### Scenario: User retry
- **WHEN** Retry is pressed for an eligible failed engine or failed load
- **THEN** the worker immediately retries without changing saved preferences or routing
#### Scenario: Inactive engine
- **WHEN** processing is off, inactive, loading or hook removal is unconfirmed
- **THEN** recovery does not reset or replace the engine
