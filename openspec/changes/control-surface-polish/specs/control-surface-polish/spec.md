## ADDED Requirements
### Requirement: Engaging responsive meters
Dial and GUI meters SHALL show a smoothed level with peak hold, and numeric knobs SHALL show their position within range, updating at least every 100 ms on the deck while a meter is displayed.
#### Scenario: Signal present
- **WHEN** a metered control reports levels
- **THEN** the meter rises immediately, falls smoothly, and shows a peak tick that holds briefly before falling
#### Scenario: Level unknown or expired
- **WHEN** no fresh reading exists
- **THEN** the meter shows LEVEL N/A rather than silence
#### Scenario: Knob position
- **WHEN** a gain dial value changes
- **THEN** its position track reflects the value within −60…+12 dB

### Requirement: Mic controls hidden on the deck when the mic stack is off
When the mic stack is off, the system SHALL blank mic-related Stream Deck keys and dials, keep their bindings and ignore their input, while the GUI keeps showing them.
#### Scenario: Stack turned off
- **WHEN** the mic stack is turned off
- **THEN** mic mute, mic target, processing, monitor, mic gain, record mic and mic stage keys render blank, while the mic stack key stays visible
#### Scenario: Stack turned on
- **WHEN** the mic stack is turned on
- **THEN** the same keys reappear with their bindings
#### Scenario: Press on a blank key
- **WHEN** a hidden key or dial is pressed or turned
- **THEN** no command is dispatched
#### Scenario: GUI unaffected
- **WHEN** the mic stack is off
- **THEN** the GUI still shows and allows the mic controls

### Requirement: Soundboard overlap
The soundboard SHALL offer a persisted Overlap toggle that allows up to eight simultaneous clips.
#### Scenario: Overlap off
- **WHEN** a clip is pressed with Overlap off
- **THEN** playing clips stop before the new clip starts
#### Scenario: Overlap on
- **WHEN** clips are pressed repeatedly with Overlap on
- **THEN** each press plays concurrently, up to eight voices, and a ninth stops the oldest
#### Scenario: Stop
- **WHEN** Stop is pressed
- **THEN** every voice stops
#### Scenario: Deck toggle
- **WHEN** the Overlap key is pressed
- **THEN** the setting toggles, persists and shows On or Off

### Requirement: Focused GUI chrome
The GUI SHALL omit the plugin-host footer and SHALL show audio health only on the Audio and Diagnostics screens.
#### Scenario: Other screens
- **WHEN** the Lights, Soundboard, Stream Deck, Plugins or Third-party apps screen is shown
- **THEN** no audio-health badge appears in the header
#### Scenario: Host connection lost
- **WHEN** the GUI cannot reach the host
- **THEN** the existing error notice reports it
