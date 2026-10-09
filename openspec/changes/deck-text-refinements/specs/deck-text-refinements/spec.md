## ADDED Requirements
### Requirement: Scrolling text instead of truncation
Deck text that cannot fit after wrapping and shrinking SHALL scroll within its box instead of being cut with an ellipsis. Scrolling SHALL redraw only the scrolling keys or strip, and only while something scrolls.

#### Scenario: Long song title
- **WHEN** a media session key's title does not fit on one line at 12 px
- **THEN** the title holds, scrolls left until its end shows, holds, and returns to the start

#### Scenario: Nothing scrolls
- **WHEN** every shown key's text fits
- **THEN** no scroll timer runs, and no key is redrawn until a frame changes

#### Scenario: Corner inset
- **WHEN** a key label scrolls
- **THEN** its pixels stay inside x 10–104

#### Scenario: Static keys while another scrolls
- **WHEN** one key scrolls
- **THEN** the other keys keep their cached images

### Requirement: Two-line values
A value of several words that does not fit on one line at 22 px SHALL wrap to two lines in place of the icon.

#### Scenario: Room name
- **WHEN** the Room key shows a room name too long for one line
- **THEN** the name wraps to two lines, and the icon is omitted

### Requirement: Pie artwork fills its band
Artwork with transparent corners SHALL be cropped to a rectangle inside its circle, so the band shows no round edge.

#### Scenario: Scene key
- **WHEN** a scene key shows its wedge artwork
- **THEN** the band is covered by wedges, with no background showing at its edges

### Requirement: Routine status hidden on scenes and clips
Scene and soundboard clip keys SHALL omit Ready, Active and Playing, relying on fill colour, and SHALL still show problem words.

#### Scenario: Playing clip
- **WHEN** a clip plays
- **THEN** its key is filled green with no status word

#### Scenario: Scene waiting for the bridge
- **WHEN** a scene is pending
- **THEN** its key shows Wait
