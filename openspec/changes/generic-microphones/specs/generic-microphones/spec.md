## ADDED Requirements
### Requirement: User-defined microphones
Snoofer SHALL route microphones defined in configuration by ID and name. Each microphone SHALL be an interface microphone (one channel mono or two channels stereo per interface) or a device microphone (a priority list of Windows input devices). No microphone role SHALL be built into the code.

#### Scenario: Single USB microphone
- **WHEN** the only microphone is a device microphone named "Yeti"
- **THEN** it feeds input 1, Auto selects it when connected, and surfaces label it "Yeti"

#### Scenario: Stereo interface pair
- **WHEN** an interface maps a microphone to channels 3 and 4
- **THEN** that microphone's input receives channel 3 on the left and channel 4 on the right

#### Scenario: Microphone not on this interface
- **WHEN** the selected interface does not map a microphone
- **THEN** that microphone is not an option, and its patch cells are 0

#### Scenario: Device microphone disconnects
- **WHEN** a device microphone's device disconnects
- **THEN** it is no longer an option, and Auto moves to the next microphone in priority

#### Scenario: Too many microphones
- **WHEN** more microphones are defined than the edition's inputs available beside VR
- **THEN** validation rejects the configuration against that edition

#### Scenario: Occupied input
- **WHEN** a device microphone's input holds a device that matches none of its candidates
- **THEN** planning stops with an error, and nothing is overwritten

### Requirement: Legacy microphones
A configuration without `microphones` SHALL behave as before, as desk and lav interface microphones plus a webcam device microphone from `fallback_mic`.

#### Scenario: Existing configuration
- **WHEN** the current studio configuration loads
- **THEN** inputs, patches, options, labels and saved choices are unchanged

### Requirement: Microphone editor
The Routing screen SHALL add, rename, reorder and remove microphones, edit device microphone priorities and set interface channels, saving before applying.

#### Scenario: Reorder microphones
- **WHEN** the user moves a microphone up
- **THEN** after saving, the microphone takes the earlier input, and routing re-plans
