## ADDED Requirements
### Requirement: Independent button status
Each Stream Deck button SHALL show only status relevant to its own binding and SHALL NOT inherit a global notice or global routing-pending flag.
#### Scenario: Unrelated global error
- **WHEN** a global error notice exists while a button's state remains valid
- **THEN** that button retains its normal presentation
#### Scenario: Monitor route pending
- **WHEN** a monitor send changes while other settings remain satisfied
- **THEN** monitor shows pending and recording, mute, media and controls buttons remain unchanged
#### Scenario: Failed action and retry
- **WHEN** a bound action fails
- **THEN** only that binding shows the action error, cleared by retry/success or bounded expiry
#### Scenario: Audio disconnected
- **WHEN** native audio observation is unavailable
- **THEN** audio-dependent controls show unavailable while media and Open controls remain usable
### Requirement: Complete presentation caching
The renderer SHALL invalidate its cache from per-control visible state and upload only changed images.
#### Scenario: Different pending setting
- **WHEN** pending work moves from monitor to recording loop while the global pending boolean stays true
- **THEN** monitor returns to its normal display and only loop becomes pending
#### Scenario: Unrelated notice changes
- **WHEN** global notice text changes without changing a button's presentation
- **THEN** its encoded image remains unchanged
