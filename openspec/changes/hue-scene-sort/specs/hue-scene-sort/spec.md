## ADDED Requirements
### Requirement: Sortable scenes
The Hue plugin SHALL offer a persisted scene sort of Name, Hue or Brightness. The deck's room-scene keys and the GUI's scene cards SHALL follow it, with ties sorted by name.

#### Scenario: Sort by brightness
- **WHEN** the sort is Brightness
- **THEN** the brightest scenes come first, scenes of equal brightness are in name order, and scenes without brightness data come last

#### Scenario: Sort by hue
- **WHEN** the sort is Hue
- **THEN** colourful scenes are ordered by their main colour around the colour wheel, mostly-white scenes follow them, and scenes without colour data come last

#### Scenario: Cycle on the deck
- **WHEN** the Sort key is pressed on the deck
- **THEN** the sort moves to the next option, is saved, and the scene keys reorder

#### Scenario: Saved across restarts
- **WHEN** Snoofer restarts
- **THEN** the chosen sort is still applied

#### Scenario: Existing settings
- **WHEN** the Hue settings have no scene sort
- **THEN** scenes sort by name
