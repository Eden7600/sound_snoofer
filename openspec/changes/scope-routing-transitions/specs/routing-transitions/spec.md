## ADDED Requirements

### Requirement: Unrelated sends remain stable
The system SHALL leave unchanged sends untouched unless they depend on a device or input patch being reconfigured.

#### Scenario: Monitoring changes
- **WHEN** the monitor is toggled or changes between Pre and Post
- **THEN** only changed monitor sends are written and voice delivery and recording sends remain stable

#### Scenario: Recording changes
- **WHEN** mic capture, computer capture or the mic tap changes
- **THEN** only changed capture sends are written and unrelated capture, playback and voice sends remain stable

#### Scenario: Voice source or mode changes
- **WHEN** source or processing mode changes without hardware changes
- **THEN** unchanged playback, computer capture and return sends are not toggled

#### Scenario: Hardware changes or reconnects
- **WHEN** input devices, output devices or ASIO input patches change
- **THEN** only dependent managed sends are temporarily disconnected, with downstream AUX sends included when an affected input feeds Element
- **AND** unrelated computer capture remains stable

### Requirement: Route transitions remain verified and ordered
The system SHALL disconnect superseded routes before adding replacements and SHALL verify each required write before continuing.

#### Scenario: Source or capture tap switches
- **WHEN** the selected microphone or recording tap changes
- **THEN** old sends are disabled before replacement sends are enabled without an intermediate duplicate or feedback path

#### Scenario: Failure, drift or disconnect
- **WHEN** a required write fails, readback fails, or concurrent inventory/settings drift occurs
- **THEN** the transition stops visibly, does not enable dependent replacement routes and can recover from a fresh plan

#### Scenario: Ambiguous device selection
- **WHEN** inventory matches ambiguously
- **THEN** existing selection and ownership safety rules remain in effect

#### Scenario: Converged state
- **WHEN** observed routing already matches the desired state
- **THEN** no send writes are issued
