## ADDED Requirements
### Requirement: Meetings screen
The GUI SHALL provide a Meetings screen with Camera and Discord cards.
#### Scenario: Plugin disabled
- **WHEN** the insta360 or discord plugin is disabled
- **THEN** its card shows that it is off, with an Enable button
#### Scenario: Discord setup
- **WHEN** Discord credentials are missing
- **THEN** the Discord card shows "Setup needed"
