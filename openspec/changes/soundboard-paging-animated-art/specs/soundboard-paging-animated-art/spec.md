## ADDED Requirements
### Requirement: Overflow scroll keys
The Stream Deck plugin SHALL offer bindable Scroll up and Scroll down keys that move between a page's overflow sets, wrapping at either end, and SHALL show the shown set's position.

#### Scenario: Paging clips
- **WHEN** the Soundboard page has two sets and the deck shows the first
- **THEN** Down shows the second set with Stop and Overlap in place, and both scroll keys show `2/2`

#### Scenario: Wrapping
- **WHEN** the deck shows the last set and Down is pressed
- **THEN** the first set is shown

#### Scenario: Single set
- **WHEN** every clip fits on the page
- **THEN** the scroll keys are blank and ignore presses

#### Scenario: Page dial skips sets
- **WHEN** a page binds a scroll key and the page dial turns from any of its sets
- **THEN** the deck shows the neighbouring page's first set

### Requirement: Animated clip artwork
The soundboard SHALL play animated GIF artwork on deck keys and on the Soundboard screen, keeping the first frame as static artwork.

#### Scenario: Animated GIF
- **WHEN** a clip's artwork is a GIF with several frames
- **THEN** its deck key and Soundboard card cycle the frames at the GIF's delays

#### Scenario: Oversized animation
- **WHEN** a GIF has more than 120 frames
- **THEN** the first frame is shown as static artwork with an artwork note

#### Scenario: Unchanged files are not decoded again
- **WHEN** the library is rescanned and an image file is unchanged
- **THEN** its artwork is reused without decoding
