## ADDED Requirements
### Requirement: Enumerate media sessions
Snoofer SHALL list every Windows media session and, when the browser extension is connected, every playing browser tab and frame as separate sessions, replacing that browser's single Windows session.

#### Scenario: Two tabs with the extension
- **WHEN** two Brave tabs play media and the extension is connected
- **THEN** two sessions appear with their own titles, and Brave's Windows session is hidden

#### Scenario: No extension
- **WHEN** the extension is not connected
- **THEN** Brave appears as the single session Windows reports

#### Scenario: Tab the extension cannot see
- **WHEN** the extension is connected but does not report the tab that Brave's Windows session describes
- **THEN** Brave's Windows session stays visible

#### Scenario: Media playing before install
- **WHEN** the extension is installed while a tab is already playing
- **THEN** that tab is reported without being reloaded

### Requirement: Per-session control and focus
Snoofer SHALL control each session individually and SHALL direct the media dial and transport keys to the focused session.

#### Scenario: Pausing a background tab
- **WHEN** the user presses the key of a session that is not focused
- **THEN** that session toggles play/pause and becomes focused

#### Scenario: Seeking with the dial
- **WHEN** the media dial turns three detents clockwise in quick succession on a seekable session at 1:00
- **THEN** the dial shows 1:15 immediately and keeps showing it, advancing if playing, until the new position is observed; one seek to 1:15 is sent after the turning stops

#### Scenario: Turning during playback
- **WHEN** the dial is turned while the session plays and its progress advances
- **THEN** every detent is applied; none is rejected as stale

#### Scenario: Hold an app to adjust it
- **WHEN** the user holds the key of an app far down the list
- **THEN** the focused-app dial controls that app's volume and mute, and a tap on the key still mutes

#### Scenario: Hold to focus
- **WHEN** the user holds a session key for half a second
- **THEN** that session becomes focused and does not toggle; a tap still plays or pauses it

#### Scenario: Focus follows playback
- **WHEN** a new session starts playing and no session was pressed in the last 30 seconds
- **THEN** focus moves to the new session

### Requirement: Local browser bridge
The bridge SHALL listen only on localhost, accept only browser-extension origins, require no pairing, and bound message size and session count.

#### Scenario: Web page connection
- **WHEN** a web page tries to open the bridge WebSocket
- **THEN** the connection is refused for its origin

#### Scenario: Store extension connects
- **WHEN** the published extension starts in Chrome or Firefox with Snoofer running
- **THEN** it connects without setup, and its tabs appear as sessions

#### Scenario: Protocol mismatch
- **WHEN** an extension offers a protocol version Snoofer does not support
- **THEN** the connection is refused, and Snoofer and the extension's popup say which side to update

### Requirement: Deck filters for apps and media
The combined Media deck page SHALL offer an Apps filter (All, Pinned, Off) and a Media filter (On, Off) that limit what the deck shows without changing the GUI screens.

#### Scenario: Pinned apps only
- **WHEN** the Apps filter is Pinned
- **THEN** only picked apps fill the app keys and dials, closing ranks, and the App audio screen still lists every app

#### Scenario: Media off
- **WHEN** the Media filter is Off
- **THEN** session keys, transport keys and the media dial are blank on every page, and the Media filter key stays visible

#### Scenario: Apps off frees space for media
- **WHEN** the Apps filter is Off and twelve sessions are playing or paused
- **THEN** sessions fill rows 1 and 2 (and row 3 if needed) instead of stopping at row 1

#### Scenario: Media off frees space for apps
- **WHEN** the Media filter is Off
- **THEN** apps fill rows 1–3 and all five dials, including the media dial's position

#### Scenario: Focused app not repeated
- **WHEN** the focused app is also among the first apps
- **THEN** it appears only on the focused-app dial, and the next app takes the region dial

#### Scenario: Home strip without overflow
- **WHEN** six sessions and six apps are shown
- **THEN** Home's row 3 shows the first four of each, and the page dial gains no extra Home pages
