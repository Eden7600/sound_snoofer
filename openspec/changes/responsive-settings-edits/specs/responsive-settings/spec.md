## ADDED Requirements

### Requirement: Controls remain editable during application
The TUI SHALL immediately display queued choices and accept further setting edits while the worker applies earlier choices.

#### Scenario: Rapid edits
- **WHEN** settings are changed repeatedly before a worker reply
- **THEN** controls reflect the latest draft and ordered edits are delivered after acknowledgement without stale-revision loss

#### Scenario: Periodic refresh
- **WHEN** worker status refreshes before edit acknowledgement
- **THEN** queued control values remain visible and navigation and source selection remain usable

### Requirement: Queued edits preserve persistence and ownership guarantees
The system SHALL save each ordered batch before applying it and SHALL retain serialized native operations.

#### Scenario: Save or revision failure
- **WHEN** a batch is rejected
- **THEN** the UI restores authoritative values, clears unconfirmed edits and displays the failure

#### Scenario: Conflicting actions and bounded queue
- **WHEN** non-setting commands are requested during pending edits or the local edit limit is exceeded
- **THEN** the UI explains the pending state without blocking navigation or silently losing accepted edits

#### Scenario: Quit
- **WHEN** the user quits with unsent local edits
- **THEN** shutdown cancels work without automatically executing recorder commands or claiming those edits were saved
