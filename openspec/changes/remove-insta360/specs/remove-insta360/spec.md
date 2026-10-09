## ADDED Requirements
### Requirement: No Insta360 integration
Snoofer SHALL NOT include, build or load any Insta360 camera control. The camera's microphone SHALL remain usable as an audio device.

#### Scenario: Build
- **WHEN** `scripts/build.ps1` runs
- **THEN** no camera DLL is built, and no `insta360` plugin is compiled in

#### Scenario: Meetings screen
- **WHEN** the GUI shows the Meetings screen
- **THEN** only the Discord card appears

#### Scenario: Default deck layout
- **WHEN** a new layout is created
- **THEN** the Meetings page has no camera keys

#### Scenario: Old configuration
- **WHEN** a saved configuration still names the `insta360` plugin
- **THEN** Snoofer starts, and reports that plugin as not compiled into this build

#### Scenario: Webcam microphone
- **WHEN** the Insta360 Link 2's microphone is connected
- **THEN** it is still offered and chosen by the microphone priorities
