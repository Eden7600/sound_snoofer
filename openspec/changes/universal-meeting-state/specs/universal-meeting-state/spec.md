## ADDED Requirements

### Requirement: Shared meeting targets
Snoofer SHALL maintain one mute target and one camera target for every eligible
compatible joined call, with provider observations and intent distinguished.

#### Scenario: External app changes shared state
- **WHEN** an established compatible call reports a fresh known writable value
  transition that is not an expected effect of Snoofer's own command
- **THEN** that value becomes shared intent, using the saved audio preference path
  for mute when available, and propagates to the other eligible calls
- **AND** conflicts are ordered by receipt at Snoofer without claiming physical-click order

#### Scenario: Persistent mute and call-scoped camera
- **WHEN** the last eligible compatible call ends
- **THEN** mute remains retained and camera intent and its old work are discarded
- **AND** the next call period seeds camera from its first known eligible observation

#### Scenario: Another call joins an active period
- **WHEN** a fresh known eligible call joins while another call remains active
- **THEN** it inherits the current shared targets without replacing them from its baseline

### Requirement: Single authority and safe application
The shared coordinator SHALL be the only reconciliation authority. Providers
SHALL retain their exact transport identity and explicit single-action guards.

#### Scenario: Audio save failure
- **WHEN** adoption of an external mute change cannot be saved to audio's preference
- **THEN** Snoofer reports failure and does not claim or propagate an unsaved target

#### Scenario: Audio is absent
- **WHEN** Meet runs without audio
- **THEN** shared mute works from a runtime-only target without starting audio or
  writing its configuration, and camera remains independent

#### Scenario: Unknown write and delayed echo
- **WHEN** application may have started but confirmation is missing
- **THEN** Snoofer shows uncertainty, does not blindly repeat the operation on
  heartbeat or reconnect, and does not adopt a delayed own effect as fresh intent

#### Scenario: Obsolete action
- **WHEN** intent, call lifetime, known state or control revision changes before a write
- **THEN** obsolete unsent work is rejected or canceled without retargeting

#### Scenario: Discord deafen or unknown camera
- **WHEN** Discord reports synthetic mute while deafened or unknown camera state
- **THEN** synthetic mute is excluded from shared adoption and unknown camera cannot
  be changed through a blind toggle

### Requirement: Shared controls and scoped placement
The existing Meetings surface SHALL expose shared Mute/Camera targets with actual
per-app observations and truthful pending, unknown and failure status.

#### Scenario: Existing bindings
- **WHEN** a saved Discord or Meet mute/camera binding is pressed
- **THEN** it requests shared intent while preserving target revision checks
- **AND** Discord-only Deafen, Share and Leave retain their existing scope

#### Scenario: Requested personal layout
- **WHEN** the user approves the deferred live placement session
- **THEN** Like and Dislike use only Plus XL Home r1c9 and r2c9 and shared meeting
  controls reuse its existing Meetings positions
- **AND** smaller decks, shipped defaults and unrelated personal assignments are preserved
