## ADDED Requirements
### Requirement: Row layout without truncation
Each deck key SHALL show its label, icon and value in separate full-width rows. Labels and values SHALL wrap and then shrink to fit before any truncation, and built-in short labels SHALL fit without truncation.

#### Scenario: Long label
- **WHEN** a key's label is "Echo strength"
- **THEN** it wraps to two lines at full size, and the icon and value keep their own rows

#### Scenario: Label too long for two lines
- **WHEN** a label does not fit on two lines at 16 px
- **THEN** it shrinks, down to 12 px, until it fits, and only beyond that is it cut with an ellipsis

#### Scenario: Key without a value
- **WHEN** a go-to key has no value
- **THEN** its icon is drawn larger in the space the value would take

#### Scenario: Missing font
- **WHEN** an embedded font cannot be parsed
- **THEN** keys still render, using the bitmap glyphs

### Requirement: Area colours
A key's area colour SHALL come from its icon. An off key SHALL be dark, with its label and icon in the area colour. An on key SHALL be filled with the area colour. Red and amber SHALL be used only for muted, recording, error, pending and fallback states.

#### Scenario: Playback on
- **WHEN** the Playback key is on
- **THEN** the key is filled blue, with dark text and icon

#### Scenario: Recorder off
- **WHEN** Record PC is off
- **THEN** the key is dark, with a violet label and icon and a grey value

#### Scenario: Muted mic
- **WHEN** the mic is muted
- **THEN** the key is filled red, and the icon is slashed

#### Scenario: Error
- **WHEN** a key's value is Error
- **THEN** the key is dark with a rounded red frame, a `!` in place of the icon and a red value

#### Scenario: Unknown area
- **WHEN** a key's icon belongs to no area
- **THEN** it uses the neutral grey area colour

### Requirement: Rounded keycaps
Key images SHALL keep text and icons at least 8 px from every edge, so the rounded keycap never hides them. Frames SHALL be drawn with rounded corners.

#### Scenario: Corner pixels
- **WHEN** any key is rendered
- **THEN** no label, icon or value pixel lies within 8 px of the image edge

### Requirement: Touch strip in the row style
Dial panels SHALL use the same typefaces and area colours, with a large value, a flat-zone meter and the page dial's previous, current and next page names.

#### Scenario: Page dial
- **WHEN** Home is shown between Media and Lights
- **THEN** the page dial shows `← MEDIA`, a large `HOME` and `LIGHTS →`

#### Scenario: Pending dial
- **WHEN** a dial's control is pending and shows Wait
- **THEN** the value is amber and dashes run along the top of its panel, as on a pending key

#### Scenario: Unknown level
- **WHEN** a meter reading is unknown or expired
- **THEN** the meter row shows N/A instead of silence
