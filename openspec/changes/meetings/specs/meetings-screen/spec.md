## ADDED Requirements
### Requirement: Meetings screen
The GUI SHALL provide a Meetings screen with Camera and Discord cards.
#### Scenario: Plugin disabled
- **WHEN** the insta360 or discord plugin is disabled
- **THEN** its card shows that it is off, with an Enable button
#### Scenario: Discord setup
- **WHEN** Discord credentials are missing
- **THEN** the Discord card shows "Setup needed"

### Requirement: Meetings deck page
The default Stream Deck layout SHALL include a Meetings page reachable in one press from Home.
#### Scenario: Default layout
- **WHEN** a new installation creates its deck layout
- **THEN** Home has a go-to Meetings key at r4c6, and the Meetings page has the call controls on row 1 with Leave at c9, the camera controls on row 2, and Playback and Mic on dials 1 and 2
