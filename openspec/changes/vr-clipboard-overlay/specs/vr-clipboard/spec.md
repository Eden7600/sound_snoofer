## ADDED Requirements

### Requirement: Optional VR clipboard dashboard
Snoofer SHALL provide an optional independent VR Clipboard dashboard panel, disabled by default, using existing SteamVR installation and plugin lifecycle facilities.

#### Scenario: Existing configuration
- **WHEN** a configuration predating this feature is loaded
- **THEN** no overlay starts and audio policy, desktop focus and clipboard remain unchanged

#### Scenario: User opens the panel
- **WHEN** the enabled plugin has connected to running SteamVR and the user selects its dashboard tile
- **THEN** large Copy, Cut, Paste, Select All, Enter, Tab and Backspace buttons and the target application are visible
- **AND** no desktop window is created or clipboard contents exposed

### Requirement: Captured desktop input
Each gesture SHALL capture the displayed desktop target and action. Only one fresh matching press and release SHALL dispatch input after fresh target, focus, desktop, privilege and modifier validation.

#### Scenario: Paste
- **WHEN** the user presses and releases Paste over the same button with a valid target
- **THEN** exactly one Ctrl+V batch is submitted to Windows
- **AND** Snoofer does not read, replace, log or persist the clipboard contents

#### Scenario: Stale or interrupted gesture
- **WHEN** the panel hides, target changes, controller ownership changes, event is stale or cancellation occurs between press and release
- **THEN** no shortcut is sent and the old gesture is not replayed later

#### Scenario: Invalid desktop target
- **WHEN** the recorded target identity changes, foreground restoration is denied, another modifier is held, the desktop is locked, or privileges are insufficient
- **THEN** the action is rejected with feedback and is not redirected to another window

#### Scenario: Partial insertion
- **WHEN** Windows accepts only part of a key batch
- **THEN** Snoofer reports an uncertain failure and attempts only key-release cleanup
- **AND** the action is never retried automatically

### Requirement: Isolated overlay lifetime
The overlay SHALL run outside the audio-host process, own and release its resources, and avoid automatic input or immediate restart after runtime shutdown.

#### Scenario: Preview
- **WHEN** Snoofer runs in preview mode
- **THEN** no overlay process is launched, runtime loaded or keyboard input sent

#### Scenario: Runtime exits
- **WHEN** SteamVR quits or the overlay worker fails
- **THEN** only the owned worker and overlay are cleaned up, the audio host stays running, and no old input is replayed

#### Scenario: Host exits
- **WHEN** Snoofer stops or loses the worker connection
- **THEN** the worker stops accepting input and closes its own overlay within bounded cleanup

### Requirement: Non-interrupting development
Implementation and verification SHALL respect the user's current prohibition on restarting Snoofer or interrupting their active VR session.

#### Scenario: Before interruption approval
- **WHEN** source development or automated checks run
- **THEN** tests use fakes and do not launch the app, contact SteamVR, send input, read clipboard or change personal configuration
- **AND** canonical deployment and hardware acceptance remain explicitly pending

### Requirement: Simulated pointer position
VR Clipboard SHALL use the last MouseMove position for matching button transitions, preserving pointer ownership and lifecycle guards independently of button-event coordinates.

#### Scenario: Button reports omit position
- **WHEN** the current controller points at a button and reports matching fresh down/up events without meaningful coordinates
- **THEN** the selected button is hit-tested at the last pointer position and acts once

#### Scenario: Pointer position is unavailable
- **WHEN** no current pointer position exists or focus, visibility, ownership or target invalidation cleared it
- **THEN** a button transition cannot use a previous pointer position to send a shortcut
