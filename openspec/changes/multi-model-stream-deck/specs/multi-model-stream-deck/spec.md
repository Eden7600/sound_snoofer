## ADDED Requirements
### Requirement: Supported Stream Deck models
The Stream Deck plugin SHALL drive the Stream Deck + XL, +, XL, MK.2 (and Original v2) and Neo over USB HID, each with its own key count, key resolution, image orientation, dials and display strip or screen.

#### Scenario: Stream Deck + connected
- **WHEN** a Stream Deck + (0FD9:0084) is connected
- **THEN** its 8 keys show 120 px images upright, its first three dials are bindable, its fourth dial pages, and the strip shows four panels

#### Scenario: + XL unchanged
- **WHEN** a Stream Deck + XL is connected with an existing layout
- **THEN** it shows the same images, layout and dial roles as before this change

#### Scenario: Unsupported model
- **WHEN** a Mini, Original v1, Pedal, Module or Studio is connected
- **THEN** it is ignored and no other deck is affected

### Requirement: Several decks at once
The plugin SHALL run every connected supported deck at the same time, each with its own session, frames and input.

#### Scenario: Two decks
- **WHEN** a + XL and a + are both connected
- **THEN** both show their layouts and each deck's keys and dials act on that deck's displayed bindings

#### Scenario: One deck unplugged
- **WHEN** one of two decks is unplugged or fails
- **THEN** the other keeps working and the unplugged deck reconnects when plugged back in

### Requirement: Per-model layouts
Each model SHALL have its own layout sized to its geometry, with a shipped default, shared by every deck of that model unless a deck has its own layout.

#### Scenario: First use of a model
- **WHEN** a model with no saved layout connects
- **THEN** it shows the shipped default for that model

#### Scenario: Layout outside the geometry
- **WHEN** a saved layout binds a key or region outside its model's keys, or the reserved Pages key
- **THEN** validation rejects it with the page and position

### Requirement: Remembered decks
A deck with a serial SHALL be remembered with its model on first connection and SHALL be renamable, given its own layout, and forgotten from the editor, whether connected or not.

#### Scenario: First connection
- **WHEN** a deck with an unknown serial connects
- **THEN** its serial and model are saved once and it appears in the editor's deck list

#### Scenario: Own layout
- **WHEN** Own Layout is turned on for a remembered deck
- **THEN** the deck uses a copy of its model layout that can be edited without changing other decks of that model

#### Scenario: Forget
- **WHEN** Forget is confirmed for a deck
- **THEN** its record and own layout are removed and a connected deck falls back to its model layout

### Requirement: Paging across decks
Page navigation SHALL be per deck when Paging is Independent and SHALL follow the last navigation on every deck that has the page when Paging is Mirrored.

#### Scenario: Independent
- **WHEN** Paging is Independent and one deck changes page
- **THEN** other decks keep their pages

#### Scenario: Mirrored
- **WHEN** Paging is Mirrored and one deck goes to Lights
- **THEN** every deck whose layout has Lights shows it, and the others stay where they are

### Requirement: Paging on decks without dials
Decks without dials SHALL page without a bindable control: the XL and MK.2 reserve their bottom-right key as the Pages key, and the Neo uses its touch buttons.

#### Scenario: Pages key
- **WHEN** the Pages key is pressed, or held for 500 ms
- **THEN** a press shows the next page, wrapping around, and a hold shows Home; the key shows the current page name

#### Scenario: Neo touch buttons
- **WHEN** a Neo touch button is pressed, or either is held for 500 ms
- **THEN** left shows the previous page and right the next, a hold shows Home, and the info screen shows the current page name
