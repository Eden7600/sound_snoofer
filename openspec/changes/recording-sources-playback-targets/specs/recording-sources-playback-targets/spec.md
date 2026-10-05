## ADDED Requirements
### Requirement: Independent capture sources and playback targets
The application SHALL present and persist recording sources separately from playback targets.
#### Scenario: Change clip destinations
- **WHEN** the user changes playback targets
- **THEN** recording source preferences remain unchanged
#### Scenario: Combined targets
- **WHEN** several supported targets are enabled
- **THEN** the same loaded clip is sent directly to each selected destination without feeding B1, B2 or Element
### Requirement: Named soundboard clips
Each configured clip SHALL have one explicit action that loads its file and plays it through eligible selected targets.
#### Scenario: Successful clip action
- **WHEN** a valid clip is selected while recording is inactive
- **THEN** the app verifies preparation and loads the requested file before submitting playback
#### Scenario: Invalid file
- **WHEN** a clip is missing or loading cannot be verified
- **THEN** the action reports failure and does not play the previously loaded file
#### Scenario: Recording active
- **WHEN** a clip is selected during recording or recording pause
- **THEN** recording continues and the clip action is rejected
#### Scenario: Repeated clip
- **WHEN** a clip is selected during soundboard playback
- **THEN** verified stop precedes loading and playing the requested clip from its beginning
### Requirement: Playback leaves live voice processing unchanged
Soundboard playback SHALL use recorder sends and SHALL NOT change live mic routes or pass clips through Element.
#### Scenario: Play while Element processes voice
- **WHEN** a clip starts, stops or changes
- **THEN** live microphone sends, source selection, mute and Element processing remain unchanged
#### Scenario: Clip with microphone muted
- **WHEN** a clip plays while the mic is muted
- **THEN** selected clip sends operate independently and mic mute remains in force
#### Scenario: Playback completion
- **WHEN** a clip finishes
- **THEN** no capture, playback or voice-route action is automatically restarted
