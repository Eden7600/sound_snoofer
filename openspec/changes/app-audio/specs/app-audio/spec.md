## ADDED Requirements
### Requirement: Per-app volume and mute
Snoofer SHALL control the Windows volume and mute of each app, grouping that app's audio sessions across processes and devices, and SHALL report requested writes as Pending until they are observed.

#### Scenario: Turning an app dial
- **WHEN** the Discord dial turns two detents clockwise from 40%
- **THEN** every Discord session is set to 44%, and the dial shows 44% once all sessions read back that value

#### Scenario: Muting a multi-process app
- **WHEN** Chrome has three sessions, one of them muted, and its key is pressed
- **THEN** all three sessions are muted, and the key shows Muted

#### Scenario: App ignores the change
- **WHEN** a written volume is not observed within 2 seconds
- **THEN** the app shows Ignored by app, and Snoofer does not rewrite it

### Requirement: App visibility
Snoofer SHALL show picked apps first, in the user's order, followed by unpicked apps whose meter exceeded −60 dBFS within the recent window.

#### Scenario: Silent active session
- **WHEN** a game holds an Active session that has produced no sound for 10 minutes and is not picked
- **THEN** it does not appear

#### Scenario: Picked app closed
- **WHEN** a picked app has no sessions
- **THEN** it keeps its position and shows Closed

### Requirement: App rules
Snoofer SHALL apply ordered rules that hide, rename or combine apps by executable, with built-in defaults that hide Snoofer, Voicemeeter and audiodg, and an explicit show rule that overrides a default.

#### Scenario: Combine
- **WHEN** the user combines a game launcher into the game
- **THEN** both executables form one app whose volume and mute apply to both

#### Scenario: Unhide a default
- **WHEN** the user unhides Voicemeeter
- **THEN** a show rule is saved ahead of the defaults, and Voicemeeter appears when heard

### Requirement: Dial regions
A Stream Deck page SHALL support dial regions filled from a collection, which page in step with the page's key regions.

#### Scenario: Seven apps
- **WHEN** seven apps are visible on a page with a five-dial and a five-key Apps region
- **THEN** the first set shows apps 1–5 on both dials and keys, and Down shows apps 6–7 on both
