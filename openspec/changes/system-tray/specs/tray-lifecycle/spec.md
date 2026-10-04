## ADDED Requirements
### Requirement: Passive tray startup
Sound Snoofer SHALL start in the system tray in live mode with adjacent default configuration and no controls window. Explicit --dry-run SHALL select preview and explicit tui SHALL retain standalone controls.
#### Scenario: Double click
- **WHEN** the packaged executable is launched without arguments
- **THEN** a tray icon appears and the worker runs without a console window
#### Scenario: Preview
- **WHEN** tray mode is launched with --dry-run
- **THEN** it presents preview status and does not write mixer settings
### Requirement: Independent controls lifetime
The tray SHALL own one audio worker independently of its optional controls window.
#### Scenario: Close and reopen
- **WHEN** controls are closed with Q or the window close button and reopened
- **THEN** audio management continues and controls show current authoritative state with working edit acknowledgements
#### Scenario: Slow or crashed controls
- **WHEN** controls stop reading state or exit unexpectedly
- **THEN** worker progress continues and a new controls window can be opened
### Requirement: Instance ownership and explicit quit
A repeated launch for the same user/session/profile/mode SHALL request existing controls without starting another worker. Quit SHALL end the tray worker and its owned controls without changing recorder transport or mixer routes.
#### Scenario: Repeated launch
- **WHEN** the same profile and mode are launched twice
- **THEN** one tray worker remains and at most one controls window is active
#### Scenario: Quit
- **WHEN** Quit is selected
- **THEN** resources are released and no recorder Start or Stop command is issued
### Requirement: Visible failures and recovery
The tray SHALL expose runtime health and retain controls access through connection, configuration and writer ownership errors.
#### Scenario: Device or API disconnect and reconnect
- **WHEN** a device or Voicemeeter disconnects and reconnects
- **THEN** the existing fallback/reconnect policy continues without requiring the controls window
#### Scenario: Ambiguous device match
- **WHEN** routing cannot resolve a device unambiguously
- **THEN** the existing routing safeguards remain effective and controls expose the unresolved state
#### Scenario: Startup failure
- **WHEN** configuration cannot load or tray startup cannot complete
- **THEN** a visible diagnostic explains the failure instead of silently exiting
### Requirement: Readable branded controls
Attached controls SHALL select a readable installed monospace font where supported, and the tray SHALL use a mascot icon derived from the painterly Sound Snoofer asset.
#### Scenario: Controls appearance
- **WHEN** a new controls console is allocated
- **THEN** Cascadia Mono is preferred, Consolas is the fallback, and global console settings remain unchanged
#### Scenario: Small tray rendering
- **WHEN** Windows displays the icon at a small or scaled size
- **THEN** it uses an appropriate resolution of the transparent mascot icon
