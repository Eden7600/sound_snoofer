## ADDED Requirements
### Requirement: Scene artwork shows brightness
Each wedge of a scene's artwork SHALL be drawn at a level that follows the brightness the scene sets for that colour, across a range wide enough to tell bright scenes from dark ones, while the darkest wedges stay visible.

#### Scenario: Moody scene
- **WHEN** a scene sets its lights to 10% brightness
- **THEN** its wedges are drawn at about a third of full level, clearly darker than a 100% scene's

#### Scenario: Mixed brightness
- **WHEN** a scene sets one light bright and another of a similar colour dim
- **THEN** the artwork shows them at different levels

#### Scenario: No brightness given
- **WHEN** a scene action sets a colour without a brightness
- **THEN** that wedge is drawn at full level
