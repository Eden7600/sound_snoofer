## ADDED Requirements
### Requirement: GUI behavior is covered by an isolated browser suite
The GUI SHALL be verified by a browser test suite in which each test starts from its own fixture and page, and every scenario of the previous GUI check is retained.

#### Scenario: Tests run independently
- **WHEN** any single GUI test runs alone or in a different order
- **THEN** it passes or fails exactly as in the full suite

#### Scenario: Existing coverage retained
- **WHEN** the suite replaces the previous GUI check
- **THEN** every assertion of the previous check has a corresponding test

### Requirement: GUI fixtures reproduce provider output
Fixtures for providers with fixed-input publication SHALL be generated from the real provider code, and a test SHALL fail when the committed fixture no longer matches it.

#### Scenario: Provider label changes
- **WHEN** a provider changes a published label, option or option label
- **THEN** the fixture drift test fails until the fixture is regenerated

#### Scenario: No personal state
- **WHEN** fixtures are generated
- **THEN** no installed device, network service, saved choices or personal setting is read

### Requirement: GUI accessibility regressions fail
Every screen SHALL be scanned for accessibility violations, and a violation not recorded with an owning change SHALL fail the suite.

#### Scenario: New violation
- **WHEN** a change introduces an unlabeled control
- **THEN** the accessibility scan fails

### Requirement: GUI layout holds at supported sizes
Every screen SHALL show no horizontal scroll, clipped controls or overlapping interactive elements at 1280×820 and 800×600.

#### Scenario: Narrow window
- **WHEN** a screen renders at 800×600
- **THEN** all controls are reachable without horizontal scrolling and none overlap

### Requirement: GUI suite runs by default without downloads
The GUI suite SHALL run in the default repository check, use the installed Chrome, and never download a browser.

#### Scenario: Chrome missing
- **WHEN** Chrome is not installed
- **THEN** the suite fails with a message naming the missing browser, and nothing is downloaded
