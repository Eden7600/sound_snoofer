## ADDED Requirements
### Requirement: Devices by identity
Priority entries, interfaces and output slots SHALL be able to name a device by its Windows endpoint ID (or ASIO driver CLSID), resolved to the device's current name for matching. Regular-expression patterns SHALL remain available.

#### Scenario: Renamed endpoint
- **WHEN** Windows renames an endpoint chosen by identity, for example from `Speakers (2- Arena)` to `Speakers (3- Arena)`
- **THEN** the entry still matches the device under its new name

#### Scenario: Disconnected device
- **WHEN** a device chosen by identity is not active
- **THEN** the entry shows its stored label as Disconnected, is not selected, and still owns a bus showing its stored name

#### Scenario: Label taken by another device
- **WHEN** a disconnected identity entry's label equals the name of a different connected device
- **THEN** the entry matches neither device

#### Scenario: Two endpoints with one name
- **WHEN** two active endpoints share the resolved name
- **THEN** the entry is ambiguous and skipped, as with patterns

#### Scenario: No Windows endpoint inventory
- **WHEN** the endpoint inventory is unavailable
- **THEN** WDM identity entries are unresolved, pattern entries work, and ASIO identity entries resolve from Voicemeeter

### Requirement: Add devices by identity
The priority editor SHALL add suggested devices by identity, keeping pattern forms as an advanced choice.

#### Scenario: Add a suggestion
- **WHEN** the user adds a suggested connected device
- **THEN** the saved entry holds its driver, endpoint ID and name, and matches it
