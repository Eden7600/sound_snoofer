## ADDED Requirements
### Requirement: Independent desktop controls
Snoofer SHALL provide a graphical controls child while the tray host retains plugin ownership.
#### Scenario: Close and reopen
- **WHEN** controls closes and is reopened
- **THEN** audio and plugin workers continue in the host and the GUI reconnects without replaying actions
#### Scenario: Host exits
- **WHEN** the host disconnects
- **THEN** its controls child exits and releases resources
### Requirement: Task-oriented screens
The GUI SHALL provide Audio, Soundboard, Stream Deck, Plugins and Diagnostics screens with standard keyboard and pointer interaction.
#### Scenario: Profile override
- **WHEN** VR owns audio
- **THEN** Normal settings remain editable with a visible override indication
#### Scenario: Live updates
- **WHEN** a user edits or searches while snapshots arrive
- **THEN** focus, unsubmitted text and browsing position are retained
#### Scenario: State feedback
- **WHEN** a control is pending, failed, unavailable, muted or overridden
- **THEN** its state is visible using text as well as semantic styling
### Requirement: Spatial deck editor
The GUI SHALL present the physical key and dial positions with a binding inspector and explicit draft save/discard.
#### Scenario: Select a position
- **WHEN** a user clicks or keyboard-selects a key or dial
- **THEN** only editor selection changes and the assigned action is not invoked
#### Scenario: Inspect ownership
- **WHEN** a position is shared or automatically populated
- **THEN** its ownership is visible alongside the effective binding
#### Scenario: Save edits
- **WHEN** a user saves a draft
- **THEN** existing layout validation and persistence govern application of the layout
### Requirement: Validated commands
The GUI SHALL preserve revision-checked semantic requests and show failures without automatic retries.
#### Scenario: Read-only state
- **WHEN** a user inspects status
- **THEN** no control operation is dispatched
#### Scenario: Stale edit
- **WHEN** the provider rejects an outdated request
- **THEN** the rejection is visible and the GUI does not silently replay it

