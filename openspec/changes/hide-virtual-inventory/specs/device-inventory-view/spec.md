## ADDED Requirements

### Requirement: Inventory omits recognized virtual endpoints
The user-facing inventory SHALL omit recognized virtual audio endpoints consistently in the TUI and CLI text/JSON output while preserving physical devices and original order.

#### Scenario: Mixed inventory
- **WHEN** discovery contains VB-CABLE, Voicemeeter virtual ASIO, physical Volt ASIO and physical microphones/speakers
- **THEN** inventory displays the physical entries and omits the virtual entries

#### Scenario: Truncated virtual name
- **WHEN** a cable endpoint name is truncated or renamed but has a recognized virtual hardware ID
- **THEN** the endpoint is omitted

#### Scenario: Empty, disconnected or virtual-only inventory
- **WHEN** no entries remain after filtering
- **THEN** the TUI displays an empty-inventory message and CLI JSON contains an empty device array
- **AND** existing connection/error status is preserved

### Requirement: Display filtering preserves routing data
Filtering SHALL NOT mutate internal device discovery, assignments or routing state.

#### Scenario: Routing uses virtual endpoints
- **WHEN** the filtered inventory is rendered
- **THEN** full internal inventory remains available for existing routing and ambiguity checks without mixer writes

#### Scenario: Unknown device
- **WHEN** an endpoint lacks recognized virtual identifiers
- **THEN** it remains visible rather than being classified solely by its ASIO driver type
