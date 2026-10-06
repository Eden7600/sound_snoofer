## ADDED Requirements
### Requirement: Vendored Lucide icons
The desktop GUI SHALL render icons from a vendored Lucide set as inline SVG that inherits text color, and SHALL NOT use Unicode glyphs as icons.
#### Scenario: Offline launch
- **WHEN** the GUI starts without network access
- **THEN** every icon renders from bundled assets
#### Scenario: Tone color
- **WHEN** a control's tone is active, attention or critical
- **THEN** its icon takes the same color as its text
#### Scenario: Stream Deck keys
- **WHEN** keys are rendered on hardware
- **THEN** they keep the code-drawn icons

### Requirement: Tailwind-built styling
The GUI SHALL be styled with Tailwind utilities and theme tokens compiled at build time, without hand-written component classes.
#### Scenario: Canonical build
- **WHEN** scripts/build.ps1 runs
- **THEN** the CSS is generated before compiling, and the build fails if it is missing or empty
#### Scenario: Go-only compile
- **WHEN** go build, go vet or go test runs without Node
- **THEN** compilation succeeds
#### Scenario: Visual continuity
- **WHEN** the migrated GUI is compared with the previous reference images
- **THEN** layout, palette, navigation and responsive behavior match, allowing pixel differences
