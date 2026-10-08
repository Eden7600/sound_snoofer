## ADDED Requirements
### Requirement: Ready microphones
A microphone marked ready SHALL never be metered or latched silent, so automatic selection never skips it for silence. A ready microphone SHALL still be unavailable when its device is missing or no selected interface maps it.

#### Scenario: Muted lavalier
- **WHEN** the lavalier is ready, first in priority, and silent for longer than the activity delay
- **THEN** Auto keeps the lavalier

#### Scenario: Ready but missing
- **WHEN** a ready device microphone's device disconnects
- **THEN** it is not an option, and Auto moves to the next microphone

#### Scenario: Not ready
- **WHEN** a microphone that is not ready stays silent for the activity delay
- **THEN** Auto skips it, and its row shows Silent

### Requirement: Complete microphone editor
The Routing screen SHALL edit every microphone setting without hand-editing JSON: name, order, source (interface channels or a device), device priority, mono or stereo channels per interface, ready, and activity metering. Every edit SHALL be saved and validated before it is applied.

#### Scenario: Stereo channels
- **WHEN** the user sets a microphone to Stereo with channels 3 and 4 on an interface
- **THEN** the interface maps it to `[3, 4]`

#### Scenario: Change source
- **WHEN** the user switches an interface microphone to a connected input device
- **THEN** it becomes a device microphone with that device, and no interface maps channels for it

#### Scenario: Add interface
- **WHEN** the user adds an interface
- **THEN** it maps no microphone channels

#### Scenario: Activity off
- **WHEN** the user sets metering to Off
- **THEN** `profiles.activity` is removed, and Auto ignores signal
