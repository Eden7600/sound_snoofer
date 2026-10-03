# TUI dashboard refinements

## Purpose

Provide compact daily audio controls with explicit microphone-source selection and a clear separation of recording settings from transport actions.

## ADDED Requirements

### Requirement: Compact dashboard chrome
The dashboard SHALL omit the standalone app-title row, idle instructional notice, second footer row and side-panel recorder-exit reminder. It SHALL retain mode, connection, recorder status, view tabs, relevant errors and one keyboard footer. Reclaimed rows SHALL be available to content without breaking selection visibility or terminal bounds.

#### Scenario: Idle dashboard
- **WHEN** no notice or error is active
- **THEN** the mode/status row and tabs lead directly to the content without a reserved blank notice row or title row

#### Scenario: Resize with a notice
- **WHEN** the terminal shrinks while a notice is visible
- **THEN** the selected control remains visible within bounds or the terminal-too-small view appears

### Requirement: Explicit microphone-source list
Enter or Space on Source SHALL open a list containing Desk, Lav, Webcam and Off, initially highlighting and marking the saved choice. Navigation SHALL not save or apply a choice. Enter SHALL confirm once; Escape SHALL cancel; Tab SHALL cancel and switch view. Confirming the current choice SHALL be a no-op. Other setting/transport shortcuts SHALL not execute behind the open list.

#### Scenario: Browse then cancel
- **WHEN** the user opens Source, highlights another option and presses Escape
- **THEN** Source, saved choices and mixer routing remain unchanged

#### Scenario: Confirm Off
- **WHEN** the user confirms Off in the source list
- **THEN** one normal source edit is submitted, preserving existing save-before-apply and live/preview behavior

#### Scenario: Disconnect or reconnect while browsing
- **WHEN** a microphone disconnects or reconnects with the list open
- **THEN** the four logical choices remain available, browsing does not issue edits, and the highlighted source is not silently replaced

#### Scenario: Ambiguous device match
- **WHEN** the user confirms a logical source whose device matching is ambiguous
- **THEN** the existing routing policy reports ambiguity or fallback without treating the picker choice as proof of a connected device

#### Scenario: Stale selection
- **WHEN** the saved-choice revision changes while the picker is open
- **THEN** the picker closes with a concise reopen message and submits no stale edit

#### Scenario: Save failure
- **WHEN** saving a confirmed choice fails
- **THEN** the previous saved choice remains active, no new routing is applied and the failure is visible

### Requirement: No dedicated microphone-Off shortcut
The TUI SHALL remove the 0 microphone-Off binding and its help text from every layout. Off SHALL remain a source-list option with its existing routing semantics.

#### Scenario: Press zero
- **WHEN** the user presses 0 with the picker closed or open
- **THEN** no source edit or mixer operation is submitted

### Requirement: Recording settings precede actions
Recording controls SHALL appear in this order: Record Computer Audio, Record Microphone, Recording Mic Stage, a nonselectable spacer and Actions heading, Start Recording, Stop Recording. Stage values SHALL display only Pre or Post. Stage labeling SHALL preserve the saved mic_tap field and its meaning. Headers and spacers SHALL not receive focus.

#### Scenario: Navigate between settings and actions
- **WHEN** the user moves down from Recording Mic Stage
- **THEN** focus moves to Start Recording across a visible separator without modifying either setting or transport

#### Scenario: Activate transport
- **WHEN** Start Recording or Stop Recording is focused
- **THEN** only Enter submits the existing guarded one-shot command; Space and selection movement do not submit it

### Requirement: Concise and accurate notices
Routine notices SHALL be concise and reflect current state: Queued, Saved · Preview, Saved · Pending, Applied or Reloaded. Success results SHALL expire after three seconds; pending and errors SHALL remain until resolved or superseded. Errors SHALL outrank success messages, retain full detail in Events and never be hidden by success expiry. Applied SHALL require successful current routing readback without unresolved routes.

#### Scenario: Save in preview
- **WHEN** a choice is saved while preview mode is active
- **THEN** the notice reads Saved · Preview, no mixer write occurs and no Applied claim appears

#### Scenario: Pending live edit resolves
- **WHEN** a saved live edit converges successfully
- **THEN** Saved · Pending becomes Applied and subsequently expires

#### Scenario: Application failure
- **WHEN** a routing or recorder command fails while a prior success notice exists
- **THEN** a concise actionable error replaces the success notice and remains visible after its expiry time
