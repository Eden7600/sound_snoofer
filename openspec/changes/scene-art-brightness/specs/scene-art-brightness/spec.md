## ADDED Requirements
### Requirement: Scene brightness badge
Scene keys with artwork SHALL show a sun badge whose rays indicate the scene's overall brightness in five steps. The artwork's wedges SHALL be drawn in full colour.

#### Scenario: Bright scene
- **WHEN** a scene sets its lights to 100%
- **THEN** its key shows a sun with eight rays

#### Scenario: Moody scene
- **WHEN** a scene sets its lights to 10%
- **THEN** its key shows the sun's disc with no rays

#### Scenario: Lights turned off
- **WHEN** a scene turns half its lights off and sets the rest to 100%
- **THEN** its overall brightness is 50%, and the sun shows four rays

#### Scenario: No brightness data
- **WHEN** a scene has neither actions nor palette
- **THEN** no badge is shown

#### Scenario: Full colour
- **WHEN** a scene sets a colour at 10% brightness
- **THEN** its wedge is drawn in that colour at full level
