## ADDED Requirements
### Requirement: Deck editor supports every model
The Stream Deck editor view SHALL publish key and dial lists for every model, empty when the model has none.

#### Scenario: Deck without dials
- **WHEN** the edited deck is a model without dials
- **THEN** the editor renders its keys, and the dial list is empty

### Requirement: Disconnected devices are not saved
A microphone or playback selection for the Normal or VR profile SHALL be refused when the chosen device is not currently available.

#### Scenario: Playback unplugged before selection
- **WHEN** the user selects a Normal or VR playback device that is no longer connected
- **THEN** the edit is refused with "no longer connected", and nothing is saved

### Requirement: Soundboard readiness follows the plan
Soundboard readiness SHALL compare the planned soundboard strip routing with the observed mixer state.

#### Scenario: Output slot carries the soundboard
- **WHEN** an output slot receives the soundboard and the mixer matches the plan
- **THEN** the soundboard is ready

### Requirement: Callback registration retries
A failed audio callback registration SHALL be retried on the next registration request.

#### Scenario: Voicemeeter starts after Snoofer
- **WHEN** registration fails because Voicemeeter is not running, and Voicemeeter starts later
- **THEN** the next registration request registers the callback

### Requirement: Deck key reports are not lost
A completed Stream Deck input report SHALL be delivered even if it completes at a read deadline.

#### Scenario: Release at the deadline
- **WHEN** a key release report completes at the moment a read would time out
- **THEN** the release is delivered, and no hold action fires

### Requirement: Browser bridge tolerates bad reports
The browser bridge SHALL skip malformed reports without closing the connection, and the extension SHALL fetch artwork only over http(s) without credentials.

#### Scenario: Malformed page report
- **WHEN** a page dispatches a report with wrongly typed fields
- **THEN** the connection stays open, and other tabs still appear

#### Scenario: Artwork URL with another scheme
- **WHEN** a report names artwork that is not an http(s) URL
- **THEN** the extension does not fetch it

### Requirement: Every interface mic is gated during hardware switches
The routing transition SHALL gate the sends of every interface microphone affected by a patch or A1 change, whatever its strip position.

#### Scenario: Interface mic in position three
- **WHEN** an interface microphone on strip 2 has its patch or the A1 interface changed
- **THEN** its sends are gated before the hardware write

### Requirement: Recording pause is independent of tape ownership saves
Tape-routing ownership SHALL be saved only when Play starts tape playback.

#### Scenario: Pause while saved choices are broken
- **WHEN** a recording is running, saved choices cannot be written, and the user presses Play/Pause
- **THEN** the recording pauses

### Requirement: Audio settings edits are validated before saving
An audio settings edit SHALL be refused, and not saved, when the saved choices would not load with the edited settings.

#### Scenario: Removing the microphone in use
- **WHEN** the user removes a microphone that the saved choices select
- **THEN** the edit is refused with the reason, and the settings on disk are unchanged

### Requirement: GUI polling survives render errors
The GUI SHALL continue polling after an exception while rendering a state.

#### Scenario: Render throws
- **WHEN** rendering one state throws
- **THEN** the next poll is still scheduled

### Requirement: Integration reconnects back off
Hue reconnect backoff SHALL reset only after the event stream delivers a message, and replacing a Hue connection SHALL end its event stream.

#### Scenario: Stream refused at once
- **WHEN** resources load but the event stream fails immediately
- **THEN** each retry waits longer than the last, up to the maximum

#### Scenario: Re-pairing
- **WHEN** the user pairs again while connected
- **THEN** the old event stream is closed

### Requirement: Discord keeps one token exchange
Only one Discord token exchange SHALL run at a time.

#### Scenario: Pipe reconnects during an exchange
- **WHEN** the Discord pipe reconnects while a refresh is in flight
- **THEN** no second exchange starts, and the saved refresh token is not erased

### Requirement: Now Playing stops cleanly
Now Playing SHALL report stopped only after its listener has closed, and SHALL retry a busy listening port.

#### Scenario: Restart with the port still busy
- **WHEN** Now Playing starts while its port is briefly in use
- **THEN** it listens once the port is free, without a restart
