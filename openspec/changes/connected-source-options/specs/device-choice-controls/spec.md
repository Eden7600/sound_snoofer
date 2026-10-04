## ADDED Requirements

### Requirement: Microphone choices reflect connected hardware
The source picker SHALL offer Desk/Lav only with unique connected Volt companion and driver matches, Webcam only with an eligible configured fallback input, and Off always.

#### Scenario: ASIO driver remains installed
- **WHEN** Volt's ASIO driver exists but its WDM companion is absent
- **THEN** Desk and Lav are omitted

#### Scenario: Ambiguous or disconnected inventory
- **WHEN** microphone selection is ambiguous or no valid observation exists
- **THEN** unsupported choices are omitted and Off remains available

#### Scenario: Device disappears while choosing
- **WHEN** the highlighted device disconnects
- **THEN** refreshed options omit it and stale confirmation cannot save a newly unavailable choice

### Requirement: Playback preference is persistent and safe
The system SHALL expose Automatic and connected eligible configured outputs as playback choices and persist the preference.

#### Scenario: Manual selection
- **WHEN** a connected playback device is selected
- **THEN** it takes priority and existing output placement and routing migration rules are preserved

#### Scenario: Disconnect and reconnect
- **WHEN** the preferred output disconnects
- **THEN** automatic priority selects a fallback without erasing the preference
- **AND** reconnecting restores preference priority

#### Scenario: Legacy or invalid selection
- **WHEN** saved intent omits playback_device
- **THEN** Automatic is used
- **WHEN** a preference does not match configured playback rules
- **THEN** validation rejects it without changing mixer settings

#### Scenario: Ambiguous device name
- **WHEN** multiple eligible output entries share a name
- **THEN** that name is not offered as a unique manual choice
