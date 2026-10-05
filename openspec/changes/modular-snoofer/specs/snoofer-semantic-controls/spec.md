## ADDED Requirements

### Requirement: Domain-neutral controls
Core SHALL expose namespaced semantic control registration, state and actions for command, toggle, selection, numeric and status controls. Providers SHALL own behavior and authoritative state; surfaces SHALL own rendering.

#### Scenario: Non-audio surface
- **WHEN** media and Stream Deck are enabled without audio
- **THEN** media controls work without constructing audio state or loading its native libraries.

#### Scenario: Invalid operation
- **WHEN** a binding requests an operation unsupported by its semantic control
- **THEN** validation rejects it with an actionable reason.

### Requirement: Stale and unavailable control handling
The control service SHALL reject stale requests, bound queues and distinguish pending, observed, unknown and failed states. One-shot commands SHALL NOT replay after provider or transport reconnection.

#### Scenario: Provider replacement
- **WHEN** a provider fails or restarts while commands are queued
- **THEN** old-generation commands are invalidated and restoration requires fresh input.

#### Scenario: Uncertain native command
- **WHEN** dispatch succeeds but native observation is unavailable
- **THEN** presentation remains unknown or pending rather than asserting successful application.

### Requirement: Unified UI
Plugin settings and diagnostics SHALL integrate into the core TUI without separate plugin applications or a generic Actions section. Required external engine-restart confirmation SHALL remain available under System.

#### Scenario: Disruptive request
- **WHEN** a surface requests an engine restart requiring confirmation
- **THEN** the System confirmation is reachable without restoring the removed Actions section.

### Requirement: Configuration ownership
Core SHALL validate the configuration envelope and enabled plugins SHALL validate their own settings. Disabled and uncompiled payloads SHALL be retained as opaque JSON without executing plugin code. Saves SHALL be atomic and revision checked.

#### Scenario: Disabled configuration
- **WHEN** settings exist for a disabled or uncompiled plugin
- **THEN** they survive unrelated edits without invoking that plugin's validators.

#### Scenario: Ambiguous or malformed configuration
- **WHEN** a document has duplicate keys, malformed JSON or unknown core fields
- **THEN** loading fails visibly without partial activation or overwriting the file.

#### Scenario: Stale settings editor
- **WHEN** settings changed after an editor opened
- **THEN** a stale save is rejected rather than losing the newer choices.


### Requirement: Styled responsive Bubble Tea shell
The unified TUI SHALL use Bubble Tea with a consistent framed layout, visible selected tab and control, grouped settings, readable values and a persistent status/help footer. It SHALL respect NO_COLOR and terminal bounds without changing provider behavior.

#### Scenario: Resize and selection
- **WHEN** the terminal shrinks or the selected control scrolls below the viewport
- **THEN** the selection remains visible within terminal bounds, with a small-terminal explanation when editing is unavailable.

#### Scenario: Override and plain terminal
- **WHEN** a section is overridden or NO_COLOR is set
- **THEN** the override remains readable and editable, and text markers still identify selection without relying on color.
