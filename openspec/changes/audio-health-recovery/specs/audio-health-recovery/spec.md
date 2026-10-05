## ADDED Requirements
### Requirement: Evidence-based health observation
The system SHALL distinguish endpoint presence, API connectivity and validated stream-fault evidence without treating silence or unchanged meters as a fault.
#### Scenario: Silent or muted source
- **WHEN** levels are zero or unchanged without an independently validated fault signature
- **THEN** no automatic recovery is triggered
#### Scenario: Sustained validated failure
- **WHEN** the same expected-active connected target presents a validated fault for three samples spanning at least one second outside transition grace
- **THEN** an incident becomes eligible for the optional recovery policy
#### Scenario: Device disconnect or VR exit
- **WHEN** a target becomes absent or ineligible
- **THEN** ordinary selection fallback applies and the old fault evidence is discarded
#### Scenario: Unknown backend behavior
- **WHEN** a readback lacks validated fault semantics
- **THEN** its health is unknown and it cannot authorize an automatic restart
### Requirement: Bounded optional engine recovery
The system SHALL offer opt-in automatic engine restart and an explicit manual restart action, serialized through the sole native writer.
#### Scenario: Eligible automatic incident
- **WHEN** auto recovery is enabled live, transport is stopped and the budget permits
- **THEN** one standalone restart command is submitted after fresh precondition checks
#### Scenario: Recording or unknown transport
- **WHEN** a fault is detected during active or unknown transport
- **THEN** automatic restart is deferred and manual restart requires an interruption confirmation
#### Scenario: Uncertain command outcome
- **WHEN** restart submission has an ambiguous result
- **THEN** it consumes the attempt and is not automatically replayed
#### Scenario: Repeated failures
- **WHEN** recovery recurs
- **THEN** a sixty-second cooldown and maximum two automatic attempts per ten minutes prevent a restart loop across app relaunches
#### Scenario: Dry-run
- **WHEN** a recovery action is requested in preview
- **THEN** no restart or other audio mutation occurs
### Requirement: Preserve intent and verify recovery
Recovery SHALL preserve saved preferences and reconcile only managed settings from fresh observations without issuing recorder transport commands.
#### Scenario: Settings edited during recovery
- **WHEN** the user changes a preference while recovery is in progress
- **THEN** the UI remains responsive and reconciliation uses the latest saved preference
#### Scenario: Successful setter only
- **WHEN** restart submission succeeds but valid health observations do not return
- **THEN** recovery is reported as unverified or failed rather than successful
#### Scenario: Muted microphone
- **WHEN** recovery occurs while the microphone is muted
- **THEN** reconciliation preserves mute intent and does not select Off or unmute to test audio
#### Scenario: Native call hangs
- **WHEN** the audio actor is blocked inside a DLL call
- **THEN** the UI reports unavailable recovery and no second native writer is started

### Requirement: Engine-wide incident deduplication
Multiple affected targets SHALL be treated as a single engine incident rather than triggering independent device restarts.
#### Scenario: All audio stalls
- **WHEN** several targets show fault evidence during the same engine stall
- **THEN** one recovery action and one global attempt budget cover the incident without device reselection
