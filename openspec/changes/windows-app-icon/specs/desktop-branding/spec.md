## ADDED Requirements
### Requirement: Windows mascot branding
The Windows executable SHALL embed the existing Sound Snoofer mascot as its application icon. Attached classic-console controls SHALL request matching large and small window icons.
#### Scenario: Executable in Explorer
- **WHEN** Windows extracts the application icon
- **THEN** the built executable supplies the mascot resource at multiple sizes
#### Scenario: Open controls
- **WHEN** a new attached controls console opens
- **THEN** it requests the mascot for its window icons without affecting routing
#### Scenario: Closed controls
- **WHEN** the controls window is closed
- **THEN** only the tray remains; no permanent taskbar button is created
#### Scenario: Unsupported window host
- **WHEN** a console host cannot accept a window icon
- **THEN** controls still function and retain the host icon
