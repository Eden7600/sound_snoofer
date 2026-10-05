## ADDED Requirements
### Requirement: Focused control sections
The TUI SHALL display one control group at a time with keyboard section navigation, preserving existing actions and diagnostics.
#### Scenario: Browse groups
- **WHEN** the user presses Left/Right or brackets in Controls
- **THEN** the current section changes cyclically without dispatching control actions
- **AND** all non-surface-only controls remain reachable
#### Scenario: Live updates
- **WHEN** controls change
- **THEN** the selected control is preserved by ID when still present in its section and removed sections fall back safely
#### Scenario: Responsive presentation
- **WHEN** a terminal is wide enough
- **THEN** a section rail appears beside the active form
- **AND** narrower terminals retain section navigation and bounded content
#### Scenario: Color preference
- **WHEN** NO_COLOR is explicitly set
- **THEN** rendering remains plain and keyboard focus remains visible
#### Scenario: Diagnostics and editing
- **WHEN** a row is selected or edited
- **THEN** its status remains available and existing apply, cancel and confirmation semantics are preserved

