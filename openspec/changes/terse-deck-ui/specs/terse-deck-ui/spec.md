## ADDED Requirements
### Requirement: Compact expert-facing controls
The Stream Deck SHALL use concise labels and state text, retaining enough distinction to identify each action and preserve uncertainty/errors.

#### Scenario: Built-in audio labels
- **WHEN** built-in audio controls are bound
- **THEN** mic mute reads Mute, monitoring reads Monitor, processing reads Mic processing, and enablement reads Mic stack without explanatory suffixes.

#### Scenario: Status and unavailable controls
- **WHEN** a control is pending, failed or unavailable
- **THEN** its deck status is concise and its TUI retains detailed diagnostics.

### Requirement: Borderless meaningful icons
Keys SHALL omit perimeter borders and use distinct signal-path icons for processing and mic-stage states.

#### Scenario: Processing and recording stage
- **WHEN** Direct/Element or Pre/Post changes
- **THEN** the icon changes to the corresponding straight/processed path or before/after tap while the text state remains visible.

### Requirement: Single application build output
The canonical build script SHALL produce only bin/snoofer.exe as the application executable, use fixed release flags and native self-checks, and reject in-use outputs before writing.

#### Scenario: Running application
- **WHEN** the application executable or companion is in use
- **THEN** the script requests exit and leaves the outputs intact without choosing another name or directory.

#### Scenario: Validation build
- **WHEN** check.ps1 reaches compilation
- **THEN** it calls build.ps1 instead of maintaining a separate build command.

### Requirement: Stable UI contract
UI changes SHALL follow the documented goals, content boundaries, terminology, semantic palette and interaction invariants; unrelated builds SHALL preserve them.

#### Scenario: Competing status colors
- **WHEN** unavailable, error, pending or fallback state coexists with an active/muted control
- **THEN** the documented precedence determines the local accent and ordinary Off remains neutral.

#### Scenario: Routine build
- **WHEN** a change does not request a UI redesign
- **THEN** established wording, placement, color meanings and interaction behavior remain unchanged.
