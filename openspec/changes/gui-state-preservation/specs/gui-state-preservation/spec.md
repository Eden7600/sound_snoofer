## ADDED Requirements

### Requirement: Structural updates preserve compatible local presentation state
Automatic structural rebuilds of the current screen SHALL preserve registered focus, scroll,
expanded details and search state by stable target identity. Restoration SHALL recompute local
filtering without sending an action and SHALL NOT restore provider-owned values from stale DOM state.

#### Scenario: Catalogue changes while an existing target remains
- **WHEN** an unrelated app, session, clip or scene appears during work on the current screen
- **THEN** the same surviving target retains its focus and presentation state
- **AND** search results and scroll positions remain consistent with the restored local state
- **AND** restoration sends no request

#### Scenario: A focused target is removed
- **WHEN** a rebuild removes or disables the previously focused target
- **THEN** focus returns to current screen content rather than a different entity
- **AND** removed-target state is discarded rather than transferred or kept for later position reuse

### Requirement: Unapplied edits remain attached to their original context
Dirty fields with explicit commitment SHALL survive compatible same-screen structural rebuilds,
including while unfocused. Their captured revisions and payload identities SHALL remain unchanged.
Context changes, removed targets or missing selected options SHALL NOT silently retarget a draft.

#### Scenario: An unfocused explicit text draft survives a rebuild
- **WHEN** a user types a name or pattern, leaves it unapplied and unrelated structure changes
- **THEN** the same valid editor retains the text and its original edit revision where one was captured
- **AND** ordinary polling does not overwrite the draft or apply it

#### Scenario: An edit context changes
- **WHEN** the owning deck page/profile/serial/slot, app payload name or Routing list configuration changes
- **THEN** incompatible local edits are not restored into the new context
- **AND** existing focused Routing edits retain their stale-list rejection guard
- **AND** no replacement target or device option is automatically committed

#### Scenario: Deferred rebuild runs on an unchanged transport response
- **WHEN** a structural update was deferred for a focused input/select and the user then leaves it
- **THEN** a later no-change poll still rebuilds and restores compatible state
- **AND** dropdown commit/cancel and focus-time revision semantics remain unchanged

### Requirement: Destructive Deck confirmation is nonblocking and target-bound
Delete Page, Forget Deck and disabling Own Layout SHALL use the shared styled dialog while polling
continues. Each local confirmation SHALL retain its original warning, action identity, revision and
target context. Confirmation SHALL dispatch at most once and only while that captured target is valid.

#### Scenario: Cancel or confirm an unchanged target
- **WHEN** a local Deck confirmation is opened
- **THEN** Cancel has initial focus, Escape cancels and cancellation sends no request
- **AND** accepting sends one request with the captured revision
- **AND** Delete Page retains draft-only behavior until Save Layout

#### Scenario: State changes while a confirmation is open
- **WHEN** the captured target disappears, becomes unavailable, changes revision or changes context
- **THEN** the affirmative action is disabled and the user is told the selection changed
- **AND** no current target/revision is substituted for the captured action
- **AND** polling and cancellation remain available without automatic replay

#### Scenario: Rebuild or host confirmation during a local dialog
- **WHEN** content rebuilds while a local dialog is open
- **THEN** the dialog keeps keyboard focus and updates continue behind it
- **AND WHEN** a host confirmation arrives
- **THEN** the local ticket is abandoned without dispatch and the host confirmation takes precedence
- **AND** existing host confirmation payload and save-validation semantics remain unchanged
