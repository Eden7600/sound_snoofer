## ADDED Requirements

### Requirement: Worker exposes only its live action protocol
The worker SHALL remove unused ID acknowledgement/deduplication and batch-edit
paths while retaining callback outcomes, origin notice clearing, scoped feedback,
single-edit validation, save-before-apply, contextual staleness and no-replay behavior.

#### Scenario: An individual edit is stale or cannot be saved
- **WHEN** a live caller submits a stale edit or persistence fails
- **THEN** no partial intent is installed and its callback and feedback report failure

#### Scenario: A subsequent action clears a prior notice
- **WHEN** a production-origin action begins after a previous failure
- **THEN** existing notice clearing remains independent of removed acknowledgements

### Requirement: Shared presentation primitives preserve caller policies
Thumbnail callers SHALL share fit/encode code while retaining their dimensions,
filters, input validation, transparency, GIF behavior and fallback semantics.
Discord and Soundboard SHALL use a common exact On/Off conversion.

#### Scenario: Existing artwork is rendered after consolidation
- **WHEN** either provider processes valid or invalid existing artwork
- **THEN** output pixels, validation and fallback match its previous behavior

#### Scenario: A boolean control is shown or compared
- **WHEN** Discord or Soundboard formats a boolean
- **THEN** true yields On and false yields Off with unchanged action semantics

### Requirement: A live browser connection cannot be displaced
The bridge SHALL reject a second live claimant for an occupied browser name,
including a same-origin claimant, without disturbing incumbent state or commands.
It SHALL allow a new generation after the incumbent disconnects.

#### Scenario: A second socket claims an occupied name
- **WHEN** a second extension connection claims the live browser name
- **THEN** the incumbent keeps its sessions and command ownership
- **AND** the rejected socket cannot publish or receive commands for that name

#### Scenario: The incumbent disconnects before reconnection
- **WHEN** its owned cleanup completes and a fresh socket claims that name
- **THEN** the fresh connection is admitted with a new generation

### Requirement: Acceptance distinguishes evidence and waivers
Audit records SHALL distinguish successful source checks, actual live acceptance,
failed setup and explicit user waivers. Desktop input SHALL require fresh approval.

#### Scenario: Required no-dial hardware is unavailable
- **WHEN** the user directs that those devices remain supported as-is
- **THEN** their physical acceptance is recorded as waived rather than passed

#### Scenario: A test fixture cannot establish a live precondition
- **WHEN** refused-device or frozen-player setup cannot be established safely
- **THEN** the check remains unverified with its concrete blocker and cleanup recorded

### Requirement: Validation fixture ownership
Concurrent test fixtures SHALL establish asynchronous readiness before submitting
actions and initialize shared data before worker startup. Their waits SHALL be
bounded and cleanup SHALL not leave test-owned goroutines blocked indefinitely.

#### Scenario: A remembered deck has not processed its connection event
- **WHEN** an outcome test is preparing its first saved edit
- **THEN** it waits for the actual connected report rather than remembered settings
- **AND** a missing save entry reports a bounded failure with any premature outcome

#### Scenario: A bridge test supplies a native session
- **WHEN** its background media worker begins polling
- **THEN** the fake native session is already initialized without concurrent mutation
