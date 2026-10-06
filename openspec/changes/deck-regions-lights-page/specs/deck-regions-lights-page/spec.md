## ADDED Requirements
### Requirement: Room motion sensor toggle
The Hue plugin SHALL offer a toggle for the motion sensors of the selected room. It SHALL report the observed state, and SHALL NOT persist or reapply that state.

#### Scenario: Room with sensors
- **WHEN** the selected room contains motion sensors that are all enabled
- **THEN** the Motion key shows On, and pressing it disables every sensor in the room

#### Scenario: Pending until observed
- **WHEN** the disable requests succeed but the bridge has not yet reported the change
- **THEN** the control shows Pending, and shows Off only after every sensor is observed disabled

#### Scenario: Mixed state
- **WHEN** some, but not all, sensors in the room are enabled
- **THEN** the key shows Mixed, and pressing it enables every sensor

#### Scenario: Room without sensors
- **WHEN** the selected room or zone has no motion sensors
- **THEN** the Motion key is blank on the deck and its input is ignored

#### Scenario: Write failure
- **WHEN** a sensor update fails
- **THEN** the key shows Error and the GUI shows the reason, and the value reflects the observed sensors

### Requirement: Page regions
A Stream Deck page SHALL support rectangular regions filled from a named control collection. Manual and shared bindings SHALL take precedence, and overflow SHALL add pages that repeat the page's fixed bindings.

#### Scenario: Region with fixed frame
- **WHEN** a page has a clips region and a manually bound Stop key outside it
- **THEN** clips fill the region in label order, and Stop stays at its position

#### Scenario: Overflow
- **WHEN** a region has more candidates than cells
- **THEN** additional pages show the remaining candidates, and every manual binding repeats at its position

#### Scenario: Manual binding inside a region
- **WHEN** a key inside a region has a manual binding
- **THEN** that key keeps its binding, and the region skips the cell

#### Scenario: Legacy automatic page
- **WHEN** a saved page has `auto_controls` and no regions
- **THEN** it fills exactly as before, as a whole-page region with prefix matching

#### Scenario: Hidden candidates
- **WHEN** a collection member is Hidden, such as an unused room-scene slot
- **THEN** the region does not place it

#### Scenario: Overlapping regions rejected
- **WHEN** a layout defines overlapping regions on one page
- **THEN** validation rejects the layout with a message naming the page

### Requirement: Go-to page keys
The Stream Deck SHALL offer a bindable key for each saved page. Pressing it SHALL switch the device to that page without changing the saved layout.

#### Scenario: Jump to a page
- **WHEN** the Lights go-to key on Home is pressed
- **THEN** the deck shows the Lights page, and the dial's press still returns Home

#### Scenario: Deleted page
- **WHEN** a go-to key refers to a page that no longer exists
- **THEN** the key is unavailable and pressing it does nothing

### Requirement: Region editor
The GUI deck editor SHALL let the user select a rectangle of keys, create a region from it with a named source, change its source and remove it as a draft edit. It SHALL number keys from one throughout.

#### Scenario: Create a region
- **WHEN** the user Shift-selects keys 10–36 and adds a region with source Soundboard clips
- **THEN** the draft preview shows those cells filled with clips marked Auto, and Save persists the region

#### Scenario: Selection never binds
- **WHEN** the user selects or extends a key range
- **THEN** no binding or control action is dispatched

### Requirement: Default content pages
New layouts SHALL include Soundboard and Lights pages built from regions. Existing saved layouts SHALL NOT be rewritten.

#### Scenario: First launch
- **WHEN** no Stream Deck layout is saved
- **THEN** the default layout has Home, Soundboard and Lights pages
