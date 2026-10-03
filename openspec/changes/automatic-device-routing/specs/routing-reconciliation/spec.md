# Routing reconciliation

## Purpose

Turn device preferences into bounded, observable mixer changes that tolerate device churn, API failures, and application restarts.

## ADDED Requirements

### Requirement: Explicit execution modes
The application SHALL provide inventory, plan, apply-once, and watch modes. Inventory and plan SHALL make no mixer-setting writes. Watch SHALL default to dry-run; live watch SHALL require an explicit apply option and configuration path.

#### Scenario: Preview a switch
- **WHEN** a dry-run sees AirPods become preferred
- **THEN** it reports the target slot, current assignment, desired device, and reason without applying the switch

### Requirement: Stable observations
Live watch SHALL require a desired routing decision to remain stable for a configurable debounce interval before applying it. Defaults SHALL be one-second polling and two-second debounce. Invalid observations SHALL reset the pending decision and never imply device removal.

#### Scenario: Bluetooth connection flaps
- **WHEN** availability changes repeatedly within the debounce interval
- **THEN** no transient routing decision is applied

### Requirement: Minimal scoped changes
The application SHALL change only managed settings that differ from desired state. Fixed routes manage device assignments. Studio mode additionally manages its four ASIO input patch cells and playback sends. Gains, mutes, inserts, unrelated patch cells and unmanaged outputs SHALL be preserved.

#### Scenario: Existing Element processing
- **WHEN** the managed microphone assignment changes
- **THEN** existing inserts and unrelated virtual cable settings remain untouched; only expressly managed playback sends and ASIO input patch cells may change

#### Scenario: No routing change needed
- **WHEN** inventory is unchanged and managed assignments match the desired state
- **THEN** repeated polling performs no setting writes or engine restart

### Requirement: Verify asynchronous writes
An accepted API write SHALL remain pending until a fresh read confirms the assignment. A failed or timed-out write SHALL be reported as a failure, retain observed state, and stop the remaining writes in that pass. Apply-once SHALL exit nonzero if its plan cannot be verified.

#### Scenario: Second assignment fails
- **WHEN** an input change succeeds but the output write fails
- **THEN** the application reports partial application accurately and does not claim an atomic transaction or successful rollback

### Requirement: Recovery and bounded retries
Watch SHALL continue observing while Voicemeeter is disconnected. After reconnection it SHALL refresh edition, inventory, and assignments before replanning. Failures SHALL use bounded retry delays and SHALL NOT cause a tight loop or automatic engine restart.

#### Scenario: Mixer restarts
- **WHEN** Voicemeeter stops and restarts while watch runs
- **THEN** writes pause during disconnection and resume only after fresh validation and debounce

### Requirement: Single active writer and shutdown
Only one live Voice Snooter writer SHALL manage a Voicemeeter session at a time. Shutdown SHALL release API and process resources without reverting assignments; dry-run processes SHALL remain read-only.

#### Scenario: Second live watcher
- **WHEN** another writer already holds ownership
- **THEN** a new apply or live-watch process exits with an ownership error before writing
