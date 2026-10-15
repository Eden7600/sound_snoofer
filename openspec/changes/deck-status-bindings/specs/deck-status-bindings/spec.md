## ADDED Requirements

### Requirement: Status controls are valid display-only key bindings
Stream Deck SHALL accept a `Kind: status` control on a key and SHALL reject it on a dial.
Text controls SHALL remain incompatible with all deck positions. Layout validation SHALL
continue to reject incompatible known controls, preserve missing-provider bindings, and
validate complete layouts at startup, assignment and Save.

#### Scenario: Audio publishes before Stream Deck starts
- **WHEN** fixed audio controls, including Playback Device and output-slot statuses, are
  published before Stream Deck starts with its shipped layouts
- **THEN** the plugin starts and publishes its editor without rejecting those status keys
- **AND** saved/default positions and binding identities are unchanged

#### Scenario: Provider appears after startup
- **WHEN** audio publishes after Stream Deck starts
- **THEN** the same status bindings remain valid and show their current state
- **AND** saving another compatible binding does not fail because status keys are present

#### Scenario: Unsupported positions remain rejected
- **WHEN** a status is assigned to a dial, a text control to either position, or a connection
  report without supported operations to a position
- **THEN** validation rejects the assignment without changing the active saved layout

### Requirement: Status keys never dispatch user actions
A displayed status binding SHALL remain read-only. Hardware key input SHALL NOT create,
queue or dispatch a semantic action for it, and SHALL NOT generate an unsupported-action
error. Existing state rendering, hidden/unavailable treatment and stale-input guards SHALL
remain in effect.

#### Scenario: Status key is tapped or held
- **WHEN** an available status key receives key down, release or a long hold
- **THEN** no provider action is invoked and no layout error is produced
- **AND** the key continues to display provider state updates

#### Scenario: Actionable neighbor still works
- **WHEN** the user presses an actionable control beside a status key
- **THEN** the existing supported action is dispatched with its displayed revision

### Requirement: Editor compatibility matches layout validation
The provider's binding options and the GUI's binding list SHALL apply the same status,
text and operation compatibility rules as layout validation. Existing filters for internal
editor controls and automatic-region membership SHALL remain unchanged.

#### Scenario: Choose a display-only status binding
- **WHEN** the editor selects a key
- **THEN** eligible status controls appear in both the provider options and GUI binding list
- **AND** assigning and saving one preserves the remaining layout

#### Scenario: Choose a dial
- **WHEN** the editor selects a dial
- **THEN** status and text controls are absent from its compatible choices
- **AND** supported adjustable controls remain available
