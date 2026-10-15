## ADDED Requirements
### Requirement: Index-based edits are tied to drawn lists
Priority, output-slot and deck-region edits SHALL carry the revision of the list they were drawn from.

#### Scenario: List changed before the click
- **WHEN** a priority list changes after its rows were drawn and before the user clicks a row action
- **THEN** the edit is rejected as stale

### Requirement: Media calls are bounded
Each call into a media application SHALL time out after 2 seconds.

#### Scenario: Frozen player
- **WHEN** a media application stops answering
- **THEN** Now Playing keeps updating other sessions

### Requirement: Builds never mix outputs
A failed build SHALL leave `bin/` unchanged, and an interrupted dependency setup SHALL be repaired by the next build.

#### Scenario: Go build fails
- **WHEN** the Go build fails after native companions were compiled
- **THEN** `bin/` keeps the previous companions

### Requirement: Callback cleanup recognises a clean state
A callback stop that finds nothing registered SHALL count as a clean stop.

#### Scenario: Voicemeeter restarted
- **WHEN** Voicemeeter restarts while Snoofer runs and the callback restarts
- **THEN** the monitor stays enabled

### Requirement: First deck press is delivered
A deck session SHALL start from the deck's current key state when the device provides it.

#### Scenario: First press after connecting
- **WHEN** the user presses a key for the first time after the deck connects
- **THEN** the press is delivered

### Requirement: Every deck open failure is reported
A deck that fails to open SHALL be reported even while another deck runs.

#### Scenario: Second deck refused
- **WHEN** one deck runs and another fails to open
- **THEN** the failing deck's error is shown

### Requirement: Hue room changes isolate writes
Light-write results from before a room change SHALL NOT affect the new room's dials.

#### Scenario: Room changed mid-write
- **WHEN** the user changes room while a light write is in flight
- **THEN** the late result neither releases the new room's queue nor marks its dials

### Requirement: Browser bridge connections cannot be evicted by other extensions
A browser bridge connection SHALL be replaced only by a connection from the same extension origin.

#### Scenario: Other extension connects
- **WHEN** another extension connects with the same browser name while the bridge is connected
- **THEN** it is refused, and the existing connection stays
