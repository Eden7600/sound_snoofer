## ADDED Requirements
### Requirement: Device changes use a one-second default debounce
The system SHALL default device debounce to 1000 ms while retaining 20 ms for send-only changes.
#### Scenario: Stable device choice
- **WHEN** a device change remains stable for 1000 ms
- **THEN** the default controller may apply it and does not apply before that deadline
### Requirement: Verification avoids repeated hardware enumeration
Mixed transactions SHALL use parameter reads for numeric operations and pending readback, retaining full inventory before and after device assignment and at transaction boundaries.
#### Scenario: Playback output changes
- **WHEN** existing sends must be gated around an output assignment
- **THEN** gating uses parameter reads and sends are restored only after full confirmation of the new assignment
#### Scenario: Hardware changes or becomes ambiguous during verification
- **WHEN** a full device confirmation differs from the planned inventory
- **THEN** application aborts before further writes and requires a fresh decision
#### Scenario: Parameter errors or missing optimization support
- **WHEN** a parameter refresh fails
- **THEN** application stops
- **WHEN** a backend lacks parameter-only reads
- **THEN** full snapshot verification is retained
### Requirement: Recorder confirmation stays responsive and bounded
Recorder confirmation SHALL poll at 5 ms without automatic transport retry.
#### Scenario: Uncertain recorder outcome
- **WHEN** a recorder command cannot be confirmed before its deadline
- **THEN** the error is reported and the command is not automatically resent
