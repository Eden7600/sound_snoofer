## ADDED Requirements
### Requirement: Link 2 camera controls
The system SHALL control an Insta360 Link 2's privacy, AI tracking, framing style and position through its UVC extension units, without opening a video stream.
#### Scenario: Privacy
- **WHEN** Privacy is toggled
- **THEN** the camera's privacy state changes and is read back within three seconds, otherwise the control shows Failed
#### Scenario: Tracking while private
- **WHEN** privacy is on
- **THEN** tracking, framing and reset are unavailable with the reason Privacy
#### Scenario: Camera absent
- **WHEN** no Link 2 is connected
- **THEN** the controls are unavailable with "No camera" and no command is sent
#### Scenario: Another app streams
- **WHEN** another application is using the camera
- **THEN** the controls still work, because only property requests are made
