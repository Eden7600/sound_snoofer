# Device discovery

## Purpose

Expose enough reliable device and mixer information to configure routing without guessing endpoint identities or treating enumeration failures as unplug events.

## ADDED Requirements

### Requirement: Read-only inventory
The system SHALL list input and output devices with direction, driver type, public name, and available hardware identity, without changing audio settings. Device indexes SHALL NOT serve as persistent identities.

#### Scenario: Multiple representations of one device
- **WHEN** an interface appears under multiple driver types
- **THEN** inventory distinguishes each representation and direction rather than merging by display name

#### Scenario: Missing stable identifier
- **WHEN** a device has no hardware identifier
- **THEN** inventory marks the identifier unavailable without preventing a unique device-name regex match

### Requirement: Availability and enumeration errors
The system SHALL distinguish a successful inventory from an enumeration failure. Eligibility SHALL require current availability evidence; an installed driver alone SHALL NOT establish connected hardware.

#### Scenario: Device disconnects and returns
- **WHEN** a WDM endpoint disappears from a successfully refreshed inventory and later returns
- **THEN** its availability follows those observations while its configured preference remains intact

#### Scenario: Enumeration fails halfway
- **WHEN** any device description or count fails
- **THEN** the snapshot is marked invalid and cannot trigger assignment changes

### Requirement: Edition and connection reporting
The system SHALL detect the running Voicemeeter edition and expose disconnected, unsupported, and connected states distinctly. Banana and Potato SHALL be supported without recompilation.

#### Scenario: Voicemeeter is stopped
- **WHEN** its DLL is installed but its engine is unavailable
- **THEN** the application reports disconnected and performs no mixer writes or automatic launch

#### Scenario: Edition changes
- **WHEN** Potato replaces Banana during a later connection
- **THEN** the application refreshes edition capabilities and validates configured slots before applying changes

#### Scenario: Missing or incompatible DLL
- **WHEN** the installed Remote DLL cannot be loaded
- **THEN** the application reports the path and actionable load error without downloading a replacement
