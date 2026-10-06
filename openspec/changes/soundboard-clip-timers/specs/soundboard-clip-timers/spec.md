## ADDED Requirements
### Requirement: Clip countdowns on the touch strip
The Stream Deck SHALL show the remaining time of each playing soundboard clip in the touch-strip panels above unbound dials on pages that show soundboard controls.

#### Scenario: One clip
- **WHEN** a clip plays and the Soundboard page shows dials 2–5 unbound
- **THEN** the dial 2 panel shows the clip's artwork and its remaining time, counting down each second

#### Scenario: Overlapping clips
- **WHEN** six clips play at once with four free panels
- **THEN** the panels show two clips each, oldest first, and a clip's timer disappears when it ends or Stop is pressed

#### Scenario: Pages without soundboard controls
- **WHEN** clips play while Home or Lights is shown
- **THEN** their panels are unchanged

#### Scenario: Unknown length
- **WHEN** a clip's normalized file length cannot be read
- **THEN** the clip plays without a timer
