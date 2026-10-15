## ADDED Requirements

### Requirement: Output destination details do not become errors
Audio output-slot controls SHALL publish their destination state in Value and their descriptive device name in structured ViewData. A healthy device name SHALL NOT occupy Status or turn a display-only destination key into Error. Genuine pending, unavailable and error handling for other controls SHALL remain unchanged.

#### Scenario: Connected output has a device name
- **WHEN** the provider publishes a Music slot with a connected speaker device and In use state
- **THEN** the actual binding-to-native-renderer path displays Music and In use with the active state color
- **AND** Routing continues to show the device name and select the current device

#### Scenario: Output is absent or cannot acquire a bus
- **WHEN** a configured slot has No device, Disconnected or No output state
- **THEN** its destination key retains that state word and existing neutral or attention color
- **AND** an unconfigured hidden slot remains blank

### Requirement: Presentation changes preserve output editing
Device reassignment, rename, removal and source toggles SHALL retain their existing payloads, saved settings, stable slot identity and stale-list rejection. Generated fixtures SHALL stay aligned with the actual provider.

#### Scenario: User chooses another output device
- **WHEN** the user explicitly commits another device from Routing
- **THEN** the request still names the same stable slot ID and selected raw device identity
- **AND** the current descriptive name is excluded from duplicate choices

#### Scenario: An old output list receives an edit
- **WHEN** configuration changes while an older output list is being edited
- **THEN** the existing stale configuration check rejects the edit instead of targeting a different slot
