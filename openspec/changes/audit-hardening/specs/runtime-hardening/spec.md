## ADDED Requirements
### Requirement: Index-based edits are tied to drawn lists
Priority and output-slot edits SHALL be rejected as stale when the configuration changed after their rows were drawn.

#### Scenario: List changed before the click
- **WHEN** a priority list changes after its rows were drawn and before the user clicks a row action
- **THEN** the edit is rejected as stale

### Requirement: Media calls are bounded
Each asynchronous WinRT media operation SHALL stop waiting after 2 seconds, request cancellation and return a timeout error if it is still running. Synchronous property reads remain unchanged; a multi-operation snapshot has no aggregate two-second deadline.

#### Scenario: Frozen player
- **WHEN** a media application stops answering
- **THEN** Now Playing keeps updating other sessions

### Requirement: Build preparation preserves deployed outputs
Dependency preparation, native compilation, GUI asset generation and Go compilation/link failures before output deployment SHALL leave existing deployed file contents unchanged. Companions, models and licenses SHALL be staged afresh and published only after esbuild, Tailwind and the Go build succeed. Build tags SHALL retain their existing selection behavior. The application SHALL continue to build only through `scripts/build.ps1`, with `bin/snoofer.exe` as its sole application output.

This guarantee excludes I/O failures while Go writes the final executable or while staged companions are copied into `bin/`. Such failures SHALL fail the build, report that deployed output may be incomplete and require a successful rebuild; rollback is not promised.

#### Scenario: Compilation or GUI generation fails
- **WHEN** native compilation, esbuild, Tailwind or Go compilation/linking fails before output deployment
- **THEN** the existing executable, companions, models and licenses keep their previous contents

#### Scenario: Tagged build follows a different build
- **WHEN** the selected build tags exclude a companion produced by an earlier build
- **THEN** no stale staged file is published and unrelated deployed files remain untouched

#### Scenario: Successful build publishes nested assets
- **WHEN** the application and its selected companions build successfully
- **THEN** the selected companions, models and licenses are copied from the fresh stage into their existing `bin/` locations

#### Scenario: Deployment write fails
- **WHEN** writing the final executable or copying a staged companion fails
- **THEN** the build reports failure and incomplete deployed output rather than reporting success

### Requirement: Interrupted neural dependency setup recovers
The next build SHALL repair an incomplete CMake extraction or missing/wrong pinned LocalVQE or GGML checkout. Verified model downloads SHALL be reused across fresh build stages.

#### Scenario: CMake extraction was interrupted
- **WHEN** the CMake cache exists but has no completed-extraction marker
- **THEN** the verified archive is extracted completely before that cache is used

#### Scenario: Dependency checkout was interrupted
- **WHEN** the LocalVQE checkout or its GGML submodule is missing the pinned revision
- **THEN** the next build restores and verifies the pinned revisions before compiling

### Requirement: Callback cleanup recognises a clean state
A callback stop that finds nothing registered SHALL count as a clean stop.

#### Scenario: Voicemeeter restarted
- **WHEN** Voicemeeter restarts while Snoofer runs and the callback restarts
- **THEN** the monitor stays enabled

### Requirement: First deck press is delivered
A deck session SHALL initialize keys and dial buttons as released on each connection. Its first valid down report SHALL emit one press, without an initial current-input query, startup delay or first-report suppression. A button already held while connecting MAY therefore activate once when first reported; repeated down reports SHALL NOT repeat that press. Existing release handling, stale-generation rejection and dark/inert behavior SHALL remain unchanged.

#### Scenario: First key or dial-button press after connecting
- **WHEN** the first valid report shows a key or dial button down after the deck connects
- **THEN** exactly one press is delivered
- **AND** the same rule applies independently to Plus and Plus XL

#### Scenario: Repeated down and release
- **WHEN** a down report repeats, then the button is released and pressed again
- **THEN** the repeated down emits no additional press and the later down emits one new press
- **AND** key release events retain their existing behavior

#### Scenario: Invalid report before first input
- **WHEN** a malformed or zero-length key report precedes the first valid down report
- **THEN** the invalid report emits no event and does not consume the first valid press

#### Scenario: Reconnect with a button already held
- **WHEN** a fresh deck session first reports a button that was already held during connection
- **THEN** that down state may activate once under the user's released-at-connect policy
- **AND** queued commands from the previous session are not replayed

### Requirement: Every deck open failure is reported
A deck that fails to open SHALL be reported even while another deck runs. Its temporary path-only report SHALL be retired when that path opens or disappears; remembered serial-based reports SHALL be retained.

#### Scenario: Second deck refused
- **WHEN** one deck runs and another fails to open
- **THEN** the failing deck's error is shown

#### Scenario: Failed path recovers or disappears
- **WHEN** a previously failing HID path opens successfully or disappears from discovery
- **THEN** its temporary error report is removed without removing remembered serial-based deck reports

### Requirement: Hue room changes isolate writes
Light-write results from before a room change SHALL NOT affect the new room's dials.

#### Scenario: Room changed mid-write
- **WHEN** the user changes room while a light write is in flight
- **THEN** the late result neither releases the new room's queue nor marks its dials

### Requirement: Browser bridge connections cannot be evicted by another connection
A browser bridge connection SHALL retain an occupied browser name until its owned cleanup
completes. Every second live claimant SHALL be refused regardless of extension origin,
without changing the incumbent's generation, sessions or command ownership. This admission
rule SHALL NOT be described as authentication of the first claimant.

#### Scenario: A second connection claims an occupied name
- **WHEN** another connection claims the same browser name while the bridge is connected
- **THEN** it is refused, including when its origin matches the incumbent
- **AND** the existing connection keeps its sessions, generation and commands

#### Scenario: A browser reconnects after cleanup
- **WHEN** the incumbent has disconnected and its owned cleanup completes
- **THEN** a fresh connection may register with a new generation
- **AND** commands captured for the old generation cannot reach it

### Requirement: Shutdown deadline failures identify the awaited plugin
When the host's cleanup deadline expires while awaiting a plugin, the returned
error SHALL identify that plugin, retain the context error for `errors.Is`, and
preserve any earlier cleanup failures. The diagnostic SHALL NOT change shutdown
order, deadlines, error presentation, build refusal or process termination policy.

#### Scenario: A plugin exceeds the host cleanup deadline
- **WHEN** the host deadline expires while a plugin's Stop call has not completed
- **THEN** the returned error identifies the awaited plugin
- **AND** `errors.Is` still recognizes `context.DeadlineExceeded`

#### Scenario: Another plugin already reported a cleanup failure
- **WHEN** one plugin reports an error before the deadline expires awaiting another
- **THEN** the returned error retains both the earlier failure and the named timeout
- **AND** no successful cleanup or confirmed hardware cause is inferred from the diagnostic

### Requirement: Initial-input diagnostics are opt-in and read-only
The native initial-report probe SHALL run only with its dedicated environment opt-in, report query and decoding results, and issue no device-output or application actions. Default tests SHALL skip it.

#### Scenario: Probe initial input on connected decks
- **GIVEN** the production host is stopped and the dedicated probe opt-in is set
- **WHEN** the probe opens each supported connected deck
- **THEN** it reports the current-input result and decode result without changing layouts, controls or playback
