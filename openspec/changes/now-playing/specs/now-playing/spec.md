## ADDED Requirements
### Requirement: Enumerate media sessions
Snoofer SHALL list every Windows media session and, when the browser extension is connected, every playing browser tab and frame as separate sessions, replacing that browser's single Windows session.

#### Scenario: Two tabs with the extension
- **WHEN** two Brave tabs play media and the extension is connected
- **THEN** two sessions appear with their own titles, and Brave's Windows session is hidden

#### Scenario: No extension
- **WHEN** the extension is not connected
- **THEN** Brave appears as the single session Windows reports

### Requirement: Per-session control and focus
Snoofer SHALL control each session individually and SHALL direct the media dial and transport keys to the focused session.

#### Scenario: Pausing a background tab
- **WHEN** the user presses the key of a session that is not focused
- **THEN** that session toggles play/pause and becomes focused

#### Scenario: Seeking with the dial
- **WHEN** the media dial turns three detents clockwise on a seekable session at 1:00
- **THEN** the session seeks to 1:15, and the dial shows Pending until the new position is observed

#### Scenario: Focus follows playback
- **WHEN** a new session starts playing and no session was pressed in the last 30 seconds
- **THEN** focus moves to the new session

### Requirement: Secure browser bridge
The bridge SHALL listen only on localhost, accept only extension origins presenting the install token, and bound message size and session count.

#### Scenario: Web page connection
- **WHEN** a web page tries to open the bridge WebSocket
- **THEN** the connection is refused for its origin

#### Scenario: Wrong token
- **WHEN** an extension connects with a stale token
- **THEN** the connection is closed, and the GUI asks to save and reload the extension
