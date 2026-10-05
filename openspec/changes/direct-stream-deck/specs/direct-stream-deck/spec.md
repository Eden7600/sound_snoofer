## ADDED Requirements
### Requirement: Standalone Stream Deck plus XL
Sound Snoofer SHALL operate the Stream Deck + XL through direct HID without Elgato software and while its controls window is closed.
#### Scenario: Device connects
- **WHEN** a supported device becomes available
- **THEN** its controls initialize and display current application state without sending audio or transport commands
#### Scenario: Disconnect and reconnect
- **WHEN** the device disconnects and returns
- **THEN** audio management continues, the display repaints, and held keys or previous commands are not replayed
#### Scenario: Busy or unsupported device
- **WHEN** access fails or the product is unsupported
- **THEN** the application reports unavailable and continues normal audio management
### Requirement: Physical gain and mute
The default layout SHALL control A1, A2 and active microphone gain and native mute using shared typed actions and authoritative observations.
#### Scenario: Turn active mic knob
- **WHEN** the microphone target and gain are known
- **THEN** the encoder changes that physical strip's gain and displays verified dB
#### Scenario: Press mic mute
- **WHEN** the microphone mute control is pressed
- **THEN** native mute changes without selecting Off, changing device assignments or resetting sends
#### Scenario: Fixed and following mute overlap
- **WHEN** A2 has both explicit bus mute and playback-following mute requested
- **THEN** clearing either request alone cannot unmute A2
### Requirement: Recording and media controls
The default layout SHALL expose recording source and transport controls, rehearsal controls, and system media commands.
#### Scenario: Recording state changes elsewhere
- **WHEN** Voicemeeter recording state changes
- **THEN** hardware feedback reflects observation rather than the last button press
#### Scenario: Invalid rehearsal command
- **WHEN** Element or verified rehearsal routing is unavailable
- **THEN** the play action is rejected with visible feedback using the existing guard
#### Scenario: Media dispatch
- **WHEN** a media key is pressed in live mode
- **THEN** one Windows media key action is dispatched without claiming verified player state
#### Scenario: Preview mode
- **WHEN** hardware controls are operated in dry-run
- **THEN** preview remains visible and no audio, recorder, Windows default or media side effect occurs
### Requirement: Bounded responsive hardware integration
Input, rendering and device I/O SHALL be bounded and cancellable without blocking the audio actor or TUI.
#### Scenario: Slow image transfer
- **WHEN** hardware rendering stalls
- **THEN** audio management and other control surfaces remain responsive
#### Scenario: Corrupt reports
- **WHEN** a report is truncated or contains invalid indices
- **THEN** it is rejected without a command or process crash
