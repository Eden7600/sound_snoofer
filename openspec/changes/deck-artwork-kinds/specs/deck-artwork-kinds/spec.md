## ADDED Requirements
### Requirement: Artwork treatment by content type
Deck artwork SHALL be drawn by content type, named by the control's icon: covers fill a cropped band, scene pies show a rectangle inside the circle, clip art is shown whole without cropping, and program icons are drawn as the key's icon.

#### Scenario: Soundboard clip
- **WHEN** a clip key shows its artwork
- **THEN** the whole image is visible, centred in the artwork area, with the key's fill around it

#### Scenario: Program key
- **WHEN** an app key shows its program icon
- **THEN** the icon sits above the app name at icon size, uncropped, and the name is one line above the volume, scrolling when too long

#### Scenario: Program placeholder
- **WHEN** an app has no logo
- **THEN** its name-coloured placeholder tile is drawn above the name the same way

### Requirement: Sharp scene and cover artwork
Scene and cover artwork SHALL be produced and accepted at up to 128 px, so it is not upscaled from 64 px on the deck.

#### Scenario: Scene key
- **WHEN** a scene key shows its wedges
- **THEN** the artwork comes from a 128 px image

#### Scenario: Oversized artwork
- **WHEN** artwork is larger than 128 px or 128 KiB encoded
- **THEN** it is rejected, and the key falls back to its icon

#### Scenario: Album cover and scene unchanged
- **WHEN** a media session or scene key shows artwork
- **THEN** it is drawn as before: a cropped cover band, or wedges inside the pie

#### Scenario: Focused app dial
- **WHEN** the focused-app dial shows a program icon
- **THEN** the icon is drawn icon-sized beside the dial's text, not stretched across the panel
