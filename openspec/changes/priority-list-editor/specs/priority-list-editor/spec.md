## ADDED Requirements
### Requirement: Editable priority lists
The GUI SHALL let the user add, remove, reorder and edit interface, playback, webcam and Normal microphone priorities. Each edit SHALL be validated and saved atomically before it is applied without a restart.

#### Scenario: Reorder playback
- **WHEN** the user moves a playback candidate up
- **THEN** the audio settings are saved with the new order before routing changes, and routing re-plans with it

#### Scenario: Invalid pattern
- **WHEN** the user applies a pattern that is not a valid Go regular expression
- **THEN** nothing is saved, the previous configuration keeps running, and the error is shown on the list

#### Scenario: Concurrent external edit
- **WHEN** `snoofer.json` changed on disk after Snoofer loaded it
- **THEN** the save is refused as stale and nothing is applied

#### Scenario: Reload failure
- **WHEN** a saved configuration fails reload validation
- **THEN** the previous configuration keeps running and the failure is shown

### Requirement: Truthful match preview
Each list entry SHALL show its current match using the planner's rules: one match, no match, or an ambiguous count. The entry the planner is using SHALL be marked.

#### Scenario: Ambiguous candidate
- **WHEN** a pattern matches two available devices
- **THEN** the entry shows Ambiguous with both names and is not marked in use

#### Scenario: Device disconnects
- **WHEN** a matched device disconnects
- **THEN** its entry shows No match after the next observation, and the next candidate is marked in use

#### Scenario: Installed ASIO driver without hardware
- **WHEN** an interface's driver is installed but its presence pattern matches nothing
- **THEN** the entry shows No match, and the driver alone is not treated as present

### Requirement: Suggested additions with generated patterns
Connected devices that no entry in a list matches SHALL be offered for that list. Each offer SHALL include an exact pattern and, where the name allows, a device pattern. Snoofer SHALL generate and escape both patterns.

#### Scenario: Add exact
- **WHEN** the user adds the suggested device `Speakers (AirPods Pro)` as exact
- **THEN** the saved pattern is `(?i)^Speakers \(AirPods Pro\)$`, and the entry matches that device

#### Scenario: Add device form
- **WHEN** the user adds the same device in device form
- **THEN** the saved pattern is `(?i)AirPods Pro`

#### Scenario: Name with regex characters
- **WHEN** a device name contains `+`, `.` or `[`
- **THEN** the generated patterns match the name literally
