## ADDED Requirements
### Requirement: Tape playback transport
Snoofer SHALL play, pause, stop, rewind and fast-forward the recorder's loaded file from the deck and the GUI, routing it to the Playback destination only while it plays, and SHALL never stop or disturb an active recording through these controls.

#### Scenario: Listen back
- **WHEN** the recorder is stopped and the user presses Play
- **THEN** the tape send to the Playback destination's bus is set and verified, the recorder plays, and Play shows Playing

#### Scenario: Pause and resume
- **WHEN** the tape is playing and the user presses Play
- **THEN** the recorder pauses; pressing again resumes

#### Scenario: Stop clears routing
- **WHEN** the user presses Stop during playback
- **THEN** the recorder stops and the tape send is cleared by the next reconciliation

#### Scenario: Recording protected
- **WHEN** the recorder is recording
- **THEN** Play, Stop, Rew and FF are unavailable and refuse without writing

### Requirement: Processed mic meter
The mic meter SHALL show the processed signal while Element is the effective processing mode, and the source strip otherwise.

#### Scenario: Element running
- **WHEN** voice processing is effectively Element
- **THEN** the mic dial and GUI meter read the AUX return strip, while mic gain still adjusts the source strip
