## ADDED Requirements
### Requirement: Lights screen
The GUI SHALL provide a Lights screen that presents the room and sync halves of the Hue plugin together, including setup, without writing settings except through plugin controls.
#### Scenario: Plugin disabled
- **WHEN** the Hue plugin is disabled
- **THEN** the Lights screen offers Enable, which uses the existing confirmed plugin selection
#### Scenario: Bridge found but unpaired
- **WHEN** a bridge is found and no key is stored
- **THEN** the screen names the bridge and offers Pair, then shows the link-button instruction until pairing succeeds or fails
#### Scenario: No bridge
- **WHEN** discovery finds no bridge
- **THEN** the screen shows the diagnostic and how to set a bridge address
#### Scenario: Paired
- **WHEN** the bridge is connected
- **THEN** the room selector, brightness and the selected room's scenes are shown, with other rooms' scenes in a disclosure
#### Scenario: Hue Sync unreachable
- **WHEN** the sync socket is unavailable
- **THEN** the sync card shows how to enable Third-party control while the room card stays usable

### Requirement: Enable-only Plugins page
The Plugins page SHALL only enable, disable and retry plugins.
#### Scenario: Plugin with controls
- **WHEN** a plugin publishes controls
- **THEN** none of its controls appear on the Plugins page

### Requirement: Blank empty slots
The Stream Deck SHALL render an unavailable control with no label and no icon as a blank key.
#### Scenario: Empty scene slot
- **WHEN** a key is bound to a scene slot with no scene
- **THEN** the key is blank rather than showing N/A
#### Scenario: Hidden control
- **WHEN** a bound control is published hidden
- **THEN** the key is blank, input is ignored and the binding is kept
#### Scenario: Ordinary unavailable control
- **WHEN** a labelled control is unavailable
- **THEN** it still shows N/A
