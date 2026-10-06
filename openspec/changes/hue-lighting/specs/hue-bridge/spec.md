## ADDED Requirements
### Requirement: Bridge discovery and pairing
The system SHALL provide an optional Hue plugin that locates a Hue Bridge on the local network or at a configured address and obtains an application key through the bridge link button.
#### Scenario: Single bridge discovered
- **WHEN** no address is configured and exactly one bridge answers discovery
- **THEN** that bridge is used and its bridge ID is confirmed before pairing
#### Scenario: Several network adapters
- **WHEN** the host has VPN, virtual and wireless adapters besides the bridge network
- **THEN** discovery queries every multicast-capable IPv4 adapter and finds the bridge on its own network
#### Scenario: No bridge
- **WHEN** no bridge answers and no address is configured
- **THEN** status shows No bridge and discovery repeats without writing settings
#### Scenario: Ambiguous bridges
- **WHEN** several bridges answer and none matches the saved bridge ID
- **THEN** no bridge is chosen automatically and the candidates appear in diagnostics
#### Scenario: Link button
- **WHEN** Pair is pressed and the bridge link button is pressed within 30 seconds
- **THEN** the bridge ID, application key and certificate fingerprint are saved atomically
#### Scenario: Pairing timeout
- **WHEN** the link button is not pressed within 30 seconds
- **THEN** pairing ends with a local error and existing settings are unchanged
#### Scenario: Address change
- **WHEN** a paired bridge reappears at a new address without a configured override
- **THEN** it is accepted only if its bridge ID matches the saved ID

### Requirement: Pinned bridge connection
The Hue plugin SHALL connect only to the paired bridge certificate and SHALL NOT expose the application key in controls or diagnostics.
#### Scenario: Certificate changed
- **WHEN** the bridge presents a certificate that does not match the saved fingerprint
- **THEN** no authenticated request is sent and status shows an error requiring re-pairing
#### Scenario: Bridge disconnects and reconnects
- **WHEN** the event stream or bridge connection is lost and later restored
- **THEN** observed values show N/A while disconnected, the full model is reloaded on reconnect, and no command is replayed

### Requirement: Scene recall controls
The Hue plugin SHALL publish one press control per bridge scene, with stable room-prefixed IDs suitable for automatic Stream Deck pages.
#### Scenario: Recall scene
- **WHEN** a scene key is pressed
- **THEN** the bridge recalls that scene and the key shows Wait until the bridge reports it active
#### Scenario: Recall unconfirmed
- **WHEN** no active status is observed within 3 seconds
- **THEN** the key shows Error with a local diagnostic and is not retried
#### Scenario: Scenes change on the bridge
- **WHEN** scenes are added, renamed or deleted in the Hue app
- **THEN** the published scene controls and automatic page contents refresh from bridge events
#### Scenario: Scene active elsewhere
- **WHEN** a scene is activated by another app
- **THEN** its key shows Active from observed bridge state

### Requirement: Room scene slots
The Hue plugin SHALL publish twelve stable slot controls that mirror the selected room's scenes in name order.
#### Scenario: Room changes
- **WHEN** a different room is selected
- **THEN** the slots show that room's scenes without changing any deck binding
#### Scenario: Fewer scenes than slots
- **WHEN** the room has fewer than twelve scenes or no room is selected
- **THEN** the remaining slots are unavailable and render as blank keys
#### Scenario: Stale slot press
- **WHEN** a slot press refers to a slot revision whose scene has since changed
- **THEN** the press is rejected and no scene is recalled

### Requirement: Room brightness knob
The Hue plugin SHALL expose a brightness control for one configured room or zone, coalescing rapid adjustment into bounded group writes, and SHALL NOT expose color-temperature control.
#### Scenario: Turn brightness
- **WHEN** the brightness dial turns
- **THEN** the requested brightness changes 2% per tick, clamped to 1–100%, and only the latest value is sent
#### Scenario: Press brightness
- **WHEN** the brightness dial is pressed with known observed state
- **THEN** the room toggles on or off
#### Scenario: Turn up while off
- **WHEN** the brightness dial turns up while the room is observed off
- **THEN** the room turns on at the requested brightness; turning down while off does nothing
#### Scenario: Fast rotation
- **WHEN** many ticks arrive faster than the group write interval
- **THEN** at most one write is outstanding and no backlog of stale values is sent
#### Scenario: Write not observed
- **WHEN** the bridge does not report the requested value within 3 seconds
- **THEN** the dial shows Error and later ticks start from the observed value
#### Scenario: No room configured or room deleted
- **WHEN** no group is selected or the selected group no longer exists
- **THEN** the brightness dial is unavailable and the saved selection is retained

### Requirement: Preview safety
The Hue plugin SHALL perform no bridge writes or pairing in preview.
#### Scenario: Dry run
- **WHEN** Snoofer runs with --dry-run
- **THEN** scenes and knobs show observed state but are unavailable for input
