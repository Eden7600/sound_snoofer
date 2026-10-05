## ADDED Requirements

### Requirement: Snoofer packaging
The application SHALL be named Snoofer while retaining the existing repository directory. Builds SHALL support explicit plugin composition, including core-only without audio native prerequisites and a default build with required audio companion artifacts.

#### Scenario: Core-only build
- **WHEN** a composition excludes audio and VR
- **THEN** building and launching it requires neither the audio native companion nor Voicemeeter.

#### Scenario: Desktop launch
- **WHEN** the renamed default executable launches normally
- **THEN** it starts through the tray without a blank terminal and opens its controls successfully.

### Requirement: External compiled plugin authors
Public contracts SHALL permit an external Go module to register a trusted plugin and rebuild a Snoofer composition without importing private internal packages.

#### Scenario: External module check
- **WHEN** a minimal external plugin is compiled into a documented composition
- **THEN** it can expose a semantic control usable by the existing TUI/deck services without editing those surfaces.

### Requirement: Manual reversible conversion
The changeover SHALL use manual conversion rather than automatic migration. Before conversion it SHALL preserve personal configuration, saved state, safety journals and matching prior executable/native artifacts with rollback instructions.

#### Scenario: Preserved operational state
- **WHEN** personal settings are converted
- **THEN** source choices, recording preferences, deck mappings and recovery/uncertain-command state retain their intended meaning.

#### Scenario: Conversion failure
- **WHEN** the new configuration fails validation
- **THEN** the original files and matched prior build remain available for rollback without partial live application.

### Requirement: Evidence-based completion
Validation SHALL distinguish automated checks from actual device acceptance and leave unperformed acceptance tasks incomplete.

#### Scenario: Tests pass without hardware
- **WHEN** unit tests and builds pass but a Volt, VR or Stream Deck trial has not run
- **THEN** the corresponding hardware acceptance remains explicitly unverified.
