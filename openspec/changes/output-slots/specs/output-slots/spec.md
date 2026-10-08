## ADDED Requirements
### Requirement: Named output slots
Snoofer SHALL support up to three named output slots, each with one static device chosen by exact name. Each slot's device SHALL be placed on its own A bus, and Snoofer SHALL never treat that device as Playback.

#### Scenario: Speakers for music, headphones for monitoring
- **WHEN** a "Music" slot names the speakers, and Playback resolves to the headphones
- **THEN** the speakers keep or receive their own bus, the headphones take the lowest other free bus, and neither device is cleared

#### Scenario: Slot device disconnects
- **WHEN** a slot's device becomes unavailable
- **THEN** the slot shows Missing, its bus stays reserved, Playback does not take it, and nothing becomes unresolved

#### Scenario: Slot device reconnects
- **WHEN** the device returns
- **THEN** the slot resumes on the same bus without moving Playback

#### Scenario: Buses exhausted
- **WHEN** Playback needs a bus and every free bus is held by slots
- **THEN** Playback takes the last slot's bus, and that slot shows No output

#### Scenario: Slot device also matches a playback pattern
- **WHEN** a slot device also matches a playback candidate pattern
- **THEN** the slot keeps the device, and Playback never selects, offers or clears it

#### Scenario: Slot removed
- **WHEN** the user removes a slot
- **THEN** its device and sends are left as they are and are no longer managed

### Requirement: Intent routing to slots
Each slot SHALL have saved per-source switches for Computer sources, Mic monitor, Soundboard and Tape. Snoofer SHALL own only those sends on the slot's bus.

#### Scenario: Monitor on headphones only
- **WHEN** monitoring is Pre and the Music slot's Monitor switch is off
- **THEN** the mic monitor reaches the Playback bus and not the Music bus

#### Scenario: Monitor on a slot
- **WHEN** the Monitor switch is on for a slot
- **THEN** the same tap that feeds Playback also feeds the slot's bus

#### Scenario: Tape to a slot
- **WHEN** the user presses Play and a slot's Tape switch is on
- **THEN** the tape is routed to Playback and to the slot before playback starts, kept there while it plays, and cleared after Stop

#### Scenario: Manual send preserved
- **WHEN** the user enables a send from an unmanaged strip to a slot's bus in Voicemeeter
- **THEN** Snoofer leaves it unchanged

#### Scenario: Mic stack disabled
- **WHEN** the mic stack is disabled
- **THEN** mic and monitor sends to every slot are off, while Computer, Soundboard and Tape slot routing continue

#### Scenario: Save before apply
- **WHEN** a route switch is toggled and saving choices fails
- **THEN** no send changes and the error is shown

### Requirement: Routing surfaces
The GUI Routing screen SHALL show sources against destinations as a toggle matrix with each destination's device and state. The default deck layout SHALL include a Routing page with the same switches.

#### Scenario: Deck page with fewer slots
- **WHEN** only one slot is configured
- **THEN** rows for slots 2 and 3 are blank keys and their bindings are kept

#### Scenario: Selecting never dispatches
- **WHEN** the user moves focus across matrix cells
- **THEN** nothing changes until a cell is activated

### Requirement: Slots without a device
An output slot SHALL be valid without a device. Its routing switches SHALL stay editable, and it SHALL hold no bus and plan no sends until a device is assigned.

#### Scenario: Set up routing before the device
- **WHEN** the user creates a "Monitor output" slot without a device and turns on Monitor
- **THEN** the slot shows No device, the switch is saved, and no bus or send changes

#### Scenario: Assign the device later
- **WHEN** a connected device is then assigned to the slot
- **THEN** the slot takes a bus and receives the switched-on sources

#### Scenario: Device disconnected
- **WHEN** the assigned device disconnects
- **THEN** the slot shows Disconnected and keeps its routing and its reserved bus, like a slot without a device
