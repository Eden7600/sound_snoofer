## ADDED Requirements
### Requirement: Minimal default layout
The default deck SHALL omit separate stop-record, tape playback/stop, Loop, To VST and Stop Media controls and SHALL place previous, play-pause and next at zero-based positions 27, 28 and 29.
#### Scenario: Removed controls
- **WHEN** the default deck is rendered
- **THEN** removed positions are blank and pressing them submits no action
### Requirement: Combined recording toggle
A single recording button SHALL start or stop recording based on observed transport state.
#### Scenario: Stop active recording
- **WHEN** recording is active or recording-paused and the button is dispatched
- **THEN** it submits stop without changing capture preferences
#### Scenario: Start stopped recorder
- **WHEN** the recorder is observed stopped
- **THEN** the button submits start
#### Scenario: Unavailable transport
- **WHEN** recorder observation is missing, audio is disconnected or tape playback is active
- **THEN** the recording toggle submits nothing
### Requirement: Icon-based controls
Common controls SHALL display recognizable icons with concise labels and scoped status.
#### Scenario: Recording state changes
- **WHEN** recording starts
- **THEN** the recording circle changes to a stop square while unrelated keys remain unchanged
### Requirement: Shared playback mute interaction
The playback button and a knob targeting the active playback bus SHALL toggle the same effective Snoofer mute preference without leaving an overlapping request active.
#### Scenario: Alternate mute controls
- **WHEN** the playback button mutes and the corresponding bus knob is clicked
- **THEN** both overlapping requests are cleared together and unmute is requested
#### Scenario: Other bus
- **WHEN** a knob controls a bus other than active playback
- **THEN** its click changes only that bus preference
