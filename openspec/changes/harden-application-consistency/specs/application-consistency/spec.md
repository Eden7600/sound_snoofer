## ADDED Requirements

### Requirement: Profile scoped mute ownership
The application SHALL derive mute ownership storage from the effective configuration path in every entry point.

#### Scenario: CLI and tray share a profile
- **WHEN** apply, watch, or tray uses the same configuration from different working directories or DLL overrides
- **THEN** each uses the same config-adjacent mute journal
- **AND** different profiles do not share journals accidentally

### Requirement: Selectable playback preferences survive validation
The application SHALL accept configured VR playback preferences consistently across picker, persistence, and routing while checking fresh eligibility before selection is saved.

#### Scenario: Headset matches only a VR profile
- **WHEN** a uniquely eligible VR playback endpoint is listed and selected
- **THEN** the selection can save and reload without requiring a duplicate ordinary playback matcher

#### Scenario: Saved endpoint disconnects
- **WHEN** a saved endpoint temporarily becomes unavailable
- **THEN** the preference remains saved and existing fallback policy determines effective routing

### Requirement: Complete managed microphone Off
Mic Off SHALL clear all managed microphone and processing-return sends, including the configured VR strip, without operating recorder transport or suppressing computer playback/capture.

#### Scenario: VR capture with recorder conflict
- **WHEN** Source Off is requested with a managed VR B1 send enabled and recorder state unknown or conflicting
- **THEN** microphone disconnection is not blocked solely by that recorder conflict
- **AND** unrelated recorder and computer-audio guards remain enforced

### Requirement: Windows default enforcement follows current permission
Windows-default correction SHALL cease before preview or released writer ownership is reported, and removal of the voice profile SHALL disable enforcement.

#### Scenario: Profile removed during live operation
- **WHEN** configuration reload removes voice intent
- **THEN** the defaults worker receives explicit disabled policy rather than retaining its previous enabled request

#### Scenario: Policy revocation with queued work
- **WHEN** preview, disable, or shutdown supersedes an older enabled request
- **THEN** old queued work cannot authorize a new setter after revocation is acknowledged
- **AND** inability to complete bounded shutdown is reported honestly

### Requirement: Disruptive actions retain their original meaning
Restart confirmation and hardware transport SHALL be scoped to current state and session, with no automatic replay of uncertain commands.

#### Scenario: Stale restart confirmation
- **WHEN** relevant state changes after confirmation was requested
- **THEN** the stale confirmation is rejected and current transport is rechecked before any later restart

#### Scenario: Hardware disconnects with queued transport
- **WHEN** the device session ends before a transport command is dispatched
- **THEN** undispatched old-session work is discarded rather than replayed after reconnect

#### Scenario: Recorder changes during a queue delay
- **WHEN** recorder state changes before a queued recording action executes
- **THEN** the original action is validated or rejected rather than reinterpreted as the opposite transport command

### Requirement: Shared control semantics
TUI and Stream Deck SHALL derive managed mute targets and relevant pending dependencies consistently while retaining their own presentation formats.

#### Scenario: Return mute is pending
- **WHEN** physical mic mute is observed but the managed Element return is not yet muted
- **THEN** neither surface claims the complete requested microphone mute is verified

#### Scenario: Configured VR strip changes
- **WHEN** a relevant send on the configured VR microphone is pending
- **THEN** affected controls show pending and unrelated controls remain unaffected

### Requirement: Accurate observation and diagnostic lifetime
The application SHALL distinguish last successful observation, worker progress, current diagnostic state, and prior action outcome.

#### Scenario: Repeated identical error
- **WHEN** the same reconciliation failure occurs repeatedly
- **THEN** deduplicated logging does not clear the active user-facing error

#### Scenario: Fault after a restart
- **WHEN** new health observations indicate unknown or unavailable stream state after a restart completed
- **THEN** current health is reevaluated independently from the historical restart result

### Requirement: Usable compact controls
Supported terminal sizes SHALL expose essential status and a path to full error details, and unsupported sizes SHALL not allow hidden setting edits.

#### Scenario: Compact supported terminal
- **WHEN** controls run at 42 by 10 or 80 by 24
- **THEN** mode, recorder status, and actionable diagnostics remain accessible without relying on a wide sidebar

#### Scenario: Resize below minimum
- **WHEN** the terminal becomes too small to display controls
- **THEN** setting/transport activation is disabled while close and resize guidance remain available

### Requirement: Explicit destructive preference interactions
Resetting all saved choices and discarding unsent local edits SHALL explain the effect before it happens, without automatically dispatching transport or waiting indefinitely for native calls.

#### Scenario: Reset activation
- **WHEN** the reset shortcut is pressed
- **THEN** the UI requests confirmation describing that all saved choices will reset

#### Scenario: Close with unsent edits
- **WHEN** closing would discard unsent local edits
- **THEN** the user can continue editing or discard them
- **AND** ordinary close without unsent edits remains immediate

### Requirement: Reliable controls focus
Open controls SHALL retain its focus request independently of coalesced state presentation.

#### Scenario: State changes before focus is delivered
- **WHEN** the renderer is slow and newer snapshots replace older pending snapshots
- **THEN** the explicit focus request remains deliverable

### Requirement: Evidence based validation
Each corrected behavior SHALL have focused runnable regression evidence, and hardware acceptance SHALL remain separate from automated checks.

#### Scenario: Standard local checks succeed
- **WHEN** tests, vet, build, and spec validation pass
- **THEN** unperformed native probes, audible checks, and unsupported race checks remain reported as unverified or skipped

#### Scenario: Preview smoke
- **WHEN** development launches a preview
- **THEN** it uses explicit dry-run and isolated config/sidecars and does not imply audible behavior was validated
