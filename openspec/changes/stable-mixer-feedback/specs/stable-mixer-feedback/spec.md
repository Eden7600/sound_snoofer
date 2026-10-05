## ADDED Requirements
### Requirement: Asynchronous mixer verification
Mixer controls SHALL tolerate bounded native readback delay without classifying it as an immediate failure.
#### Scenario: Gain propagation
- **WHEN** a gain write becomes visible within 100 ms
- **THEN** the change succeeds after readback without repeating the write
#### Scenario: Gain cannot be verified
- **WHEN** verification times out or the API fails
- **THEN** the affected gain control reports failure while retaining available observed dB
#### Scenario: Mute propagation
- **WHEN** a mute setter succeeds but readback has not caught up
- **THEN** dependent routing remains blocked and the next observation is scheduled without a connection error
### Requirement: Connection status reflects native observation
Audio availability SHALL be independent of routing and write diagnostics.
#### Scenario: Routing failure with healthy observation
- **WHEN** a routing or setter error occurs but native observation succeeds
- **THEN** audio controls remain connected and the diagnostic remains available
#### Scenario: Actual read failure
- **WHEN** native observation fails
- **THEN** audio availability becomes unknown until observation recovers
