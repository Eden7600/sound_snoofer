## ADDED Requirements

### Requirement: Runtime call focus
Snoofer SHALL provide a focus button for each eligible joined compatible call.
Focus SHALL control Deafen, Share and Leave while Mute/Camera remain universal.

#### Scenario: Calls arrive without explicit focus
- **WHEN** another eligible joined call is first observed
- **THEN** automatic focus selects the most recently observed new call
- **AND** unchanged reports do not change that ordering

#### Scenario: User selects focus
- **WHEN** the user presses a call's focus button
- **THEN** focus remains on that exact call while it is eligible
- **AND** after it disappears the newest eligible remaining call is selected

### Requirement: Scoped one-shot actions
Focused actions SHALL capture exact focus, call lifetime and control revision,
perform at most one operation, and report submission separately from observation.

#### Scenario: Focus or call changes before execution
- **WHEN** a captured action becomes obsolete before sending
- **THEN** it is rejected or canceled without reaching another call

#### Scenario: Unsupported deafen
- **WHEN** no joined call supports Deafen or the focused call is Meet
- **THEN** Deafen is N/A and cannot change playback or another call

#### Scenario: Discord call leaves and rejoins
- **WHEN** Deafen input was captured before a call lifetime change
- **THEN** Discord rejects it before RPC even if the channel ID is reused

### Requirement: Meet presentation and departure
Meet SHALL expose guarded presentation and Leave actions through the companion.
Older helpers without these capabilities SHALL remain unavailable for them.

#### Scenario: User starts presentation workflow
- **WHEN** Share opens Meet's normal menu or source chooser
- **THEN** Snoofer reports submission and the user chooses what to share
- **AND** no source is automatically selected and sharing is not falsely shown On

#### Scenario: User leaves a Meet call
- **WHEN** Leave is requested for a fresh eligible call
- **THEN** the captured unique Leave button is invoked at most once
- **AND** successful departure requires observation, not only transport receipt

#### Scenario: Ambiguous controls or uncertain result
- **WHEN** controls are missing, replaced, ambiguous, unavailable or unconfirmed
- **THEN** stale work is refused or reported unknown without automatic replay

### Requirement: Focus presentation and preserved layout
GUI and Stream Deck SHALL show the focused call and truthful capability state.

#### Scenario: User's Plus XL placement
- **WHEN** the requested Meetings layout update is applied
- **THEN** existing Deafen/Share/Leave positions use focused actions and spare
  positions hold call focus buttons
- **AND** smaller decks and unrelated user bindings remain unchanged
