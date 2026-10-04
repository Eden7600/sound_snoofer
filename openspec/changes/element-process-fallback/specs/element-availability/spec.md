## ADDED Requirements
### Requirement: Unavailable Element falls back without changing preferences
The system SHALL retain saved Element settings while deriving Direct routing when element.exe is absent or cannot be observed.
#### Scenario: Host closes
- **WHEN** Element preference is selected and element.exe is absent
- **THEN** the mic routes directly to B3, Element feed/return sends disconnect, Post monitor/capture effectively use Pre and the TUI indicates fallback
#### Scenario: Host reopens
- **WHEN** element.exe becomes available again
- **THEN** the original processing, monitoring and capture preferences resume without rewriting saved choices
#### Scenario: Observation error
- **WHEN** process enumeration fails or no process observation exists
- **THEN** Direct fallback applies and status identifies unknown availability
#### Scenario: Off overrides fallback
- **WHEN** the microphone source is Off
- **THEN** no mic or return sends are enabled regardless of process availability
### Requirement: Rehearsal respects host availability
The system SHALL pause effective snippet routing and reject Play while Element is unavailable, retaining the requested rehearsal setting.
#### Scenario: Host exits during rehearsal
- **WHEN** Element closes while the tape is playing
- **THEN** tape-to-Element and managed tape-monitor sends disconnect, direct live voice is restored if enabled, and no transport command is sent
#### Scenario: Host returns
- **WHEN** Element returns with rehearsal still requested
- **THEN** rehearsal routing resumes without automatically issuing Play
### Requirement: Host changes invalidate pending routing
The controller SHALL revalidate Element availability during transactions and abort stale transitions.
#### Scenario: Host changes during a write
- **WHEN** observed Element availability changes before a transaction completes
- **THEN** the controller requires a fresh plan before further writes
### Requirement: Post preferences survive Direct mode
Monitor and recording-stage Post preferences SHALL be retained for automatic and explicitly selected Direct mode, with effective Pre routing until Element processing is active.
#### Scenario: Direct selected then Element selected
- **WHEN** Direct becomes effective while Post is preferred
- **THEN** effective monitoring/capture uses Pre without changing saved Post
- **AND** returning to effective Element restores Post
### Requirement: Controls indicate unmet preferences
Setting controls SHALL use yellow for pending application and red for an effective override, retaining readable text indicators and selection visibility.
#### Scenario: Queued or unverified control
- **WHEN** a control's preference is queued or its relevant operations have not converged
- **THEN** that control is yellow with a pending marker while unrelated satisfied controls remain normal
#### Scenario: Overridden control
- **WHEN** effective behavior differs from a preference
- **THEN** that control is red and displays the effective value
#### Scenario: Preference becomes satisfied
- **WHEN** readback confirms the preferred behavior
- **THEN** the pending or override indication clears
