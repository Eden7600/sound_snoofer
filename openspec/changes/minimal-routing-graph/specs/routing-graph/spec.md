## ADDED Requirements
### Requirement: Minimal navigation
The TUI SHALL expose only Controls and Graph tabs and SHALL remove presentation-only code and stored history backing the removed tabs.
#### Scenario: Cycle tabs
- **WHEN** the user navigates forward or backward from either tab
- **THEN** the other tab is selected without changing controls or submitting audio actions
#### Scenario: Picker open
- **WHEN** Tab is pressed with a device picker open
- **THEN** the picker closes and Graph opens without committing a selection
### Requirement: Observed ASCII routing graph
Graph SHALL display observed sends and device assignments with ASCII nodes and arrows, including mic, playback, recording and Element paths when configured.
#### Scenario: Pending output change
- **WHEN** a different playback output is requested but not applied
- **THEN** the graph retains observed sends and marks changes pending
#### Scenario: Mic Off
- **WHEN** Off is requested but mic sends still exist
- **THEN** those observed sends remain visible until readback confirms disconnection
#### Scenario: Missing observation or disconnected devices
- **WHEN** send values or assignments are missing, or observation fails
- **THEN** unknown/unavailable state is explicit rather than represented as a working path
#### Scenario: Element processing
- **WHEN** B2 sends or AUX returns are observed
- **THEN** their mixer edges are shown and the external Element path is distinguished as unverified
#### Scenario: Recorder
- **WHEN** recorder playback sends or bus capture arms are observed
- **THEN** those paths and the recorder transport state are shown without implying playback or recording has started
#### Scenario: Unresolved or ambiguous match
- **WHEN** a plan has unresolved destinations
- **THEN** concise reasons remain accessible on Graph without inventing a resolved edge
### Requirement: Bounded safe rendering
Graph SHALL remain readable within terminal bounds and SHALL sanitize device names and diagnostics.
#### Scenario: Narrow terminal
- **WHEN** a route does not fit horizontally
- **THEN** its destination is placed on another line and vertical scrolling exposes the whole graph
