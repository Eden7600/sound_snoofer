## ADDED Requirements

### Requirement: GUI source is strictly typed and modular
The GUI SHALL use TypeScript modules for its screens, shared widgets, state and host binding,
with strict checking and unchecked indexed access checking enabled. The host and provider types
SHALL reflect the Go wire representation, and browser tests SHALL be included in type-checking.

#### Scenario: Invalid host field or unguarded array entry
- **WHEN** GUI code accesses a nonexistent typed field or uses a possibly missing array entry
- **THEN** the default frontend check fails

### Requirement: Extraction preserves observable behavior
The migration SHALL preserve existing screen behavior, action IDs and revisions, pending/error
semantics, saved settings, confirmation behavior and all existing tests. Tracked content and
layout defects SHALL remain assigned to their owning follow-up changes.

#### Scenario: Existing GUI journey after extraction
- **WHEN** an existing browser journey runs against the bundled GUI
- **THEN** it retains the same assertions, outbound actions and expected outcomes

### Requirement: The packaged GUI includes its generated assets
The canonical build SHALL generate and embed the JavaScript bundle and stylesheet, and reject
missing or empty generated outputs. No development server SHALL be required at runtime.

#### Scenario: Packaged desktop launch
- **WHEN** the canonical executable launches its controls window
- **THEN** all screen modules and styles load from embedded assets

### Requirement: Frontend hygiene gates run by default
The default check SHALL run TypeScript, typescript-eslint, Prettier, pure model tests and the
browser suite. Dependencies SHALL remain development-only and no frontend framework is added.

#### Scenario: Unformatted or lint-invalid frontend code
- **WHEN** a frontend change violates the configured formatting or lint rules
- **THEN** the default check fails before the change is accepted
