## ADDED Requirements
### Requirement: Inactive mic-path keys
While the mic stack is off, mic-path controls SHALL be shown on the deck as inactive: greyed, with no value or meter, ignoring input, and keeping their positions. The GUI SHALL be unchanged.

#### Scenario: Disable the mic stack
- **WHEN** the mic stack is turned off
- **THEN** Mic Mute, Mic Target, Mic Path, Monitor, Mic gain, Record Mic and Mic Stage stay in place on the deck, drawn greyed without values

#### Scenario: Press an inactive key
- **WHEN** an inactive key is pressed or an inactive dial is turned
- **THEN** nothing is dispatched

#### Scenario: Re-enable the mic stack
- **WHEN** the mic stack is turned back on
- **THEN** the keys return to their normal colours and values, and accept input

#### Scenario: Regions keep their cells
- **WHEN** an inactive binding sits inside a region
- **THEN** the region does not place another control in its cell

### Requirement: Artwork fills freed space
An artwork key without a value SHALL bottom-align its label and grow its artwork band to fill the space above it, cropped to the band's shape.

#### Scenario: Clip without a status word
- **WHEN** a soundboard clip key with artwork shows no value
- **THEN** the label sits at the bottom of the key, and the artwork fills the rest

#### Scenario: Scene pie
- **WHEN** a scene key shows pie artwork with no value
- **THEN** the taller band still shows only wedges, with no round edge
