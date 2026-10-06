## ADDED Requirements
### Requirement: Connection reports
Plugins SHALL publish one read-only connection report per external integration with a typed state, endpoint, state-entered time, last activity, last error and ordered details, and SHALL never include credentials.
#### Scenario: Peer connects
- **WHEN** an integration establishes its connection
- **THEN** its report shows Connected with the time the state was entered and refreshes last activity on each successful exchange
#### Scenario: Peer lost
- **WHEN** the peer disconnects or fails
- **THEN** the report shows the new state and since time, and keeps the error with its time
#### Scenario: Recovery
- **WHEN** the peer recovers after a failure
- **THEN** the report shows Connected while still showing the previous error as a past error
#### Scenario: Feature off
- **WHEN** an integration is disabled by configuration or intent
- **THEN** its report shows Off rather than an error
#### Scenario: Unknown
- **WHEN** a provider cannot determine the state
- **THEN** the report shows Unknown, distinct from Disconnected
#### Scenario: Not bindable
- **WHEN** the Stream Deck configurator lists bindable controls
- **THEN** connection reports are not offered

### Requirement: Third-party apps screen
The GUI SHALL provide a Third-party apps screen that renders all connection reports generically, grouped by plugin, with relative times, a summary and copyable details.
#### Scenario: Reports present
- **WHEN** plugins publish reports
- **THEN** each appears as a card with state, endpoint, since, last activity, last error and details, without screen-specific code per plugin
#### Scenario: Times advance
- **WHEN** time passes without new reports
- **THEN** relative times update in place without rebuilding the screen
#### Scenario: Copy details
- **WHEN** Copy details or Copy all is pressed
- **THEN** a plain-text report with absolute timestamps is copied, or a local error is shown if the clipboard is unavailable
#### Scenario: Plugin disabled
- **WHEN** a plugin is disabled or failed to start
- **THEN** it is listed as not monitored with its plugin status

### Requirement: Complete integration coverage
Every external integration owned by a plugin SHALL publish a report: Voicemeeter Remote API, audio-callback monitor, Voicemeeter recorder, ASIO interface, Element, Windows default-device protection, SteamVR, Stream Deck hardware, soundboard playback, Hue Bridge, Hue Sync and Windows media keys.
#### Scenario: Stream Deck device versus layout
- **WHEN** the layout editor reports a save or validation message
- **THEN** the Stream Deck device report is unaffected
#### Scenario: Voicemeeter not running
- **WHEN** Voicemeeter is closed while Snoofer runs
- **THEN** the Voicemeeter report shows Disconnected with the read error, and returns to Connected with a new since time when Voicemeeter returns
