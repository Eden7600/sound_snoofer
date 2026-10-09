## ADDED Requirements
### Requirement: Editable region overflow
The Stream Deck editor SHALL show and set, for each key and dial region, whether it adds overflow sets. A region with overflow off SHALL show only what fits and SHALL NOT add sets, including a region whose rows are shared by several sources. The change SHALL apply only when the layout is saved.

#### Scenario: Turn overflow off
- **WHEN** the user turns off Overflow on a Home region and saves
- **THEN** the region is saved with `clip: true`, and Home has no extra sets from it

#### Scenario: New region
- **WHEN** the user adds a region
- **THEN** its Overflow is on, as before

#### Scenario: Clipped stacked region
- **WHEN** a clipped region's rows are shared by two sources with more members than fit
- **THEN** the page has one set showing what fits

#### Scenario: Unsaved draft
- **WHEN** the user turns Overflow off and then chooses Discard
- **THEN** the saved layout and the deck are unchanged

#### Scenario: Stale editor
- **WHEN** an Overflow edit arrives for an editor revision that has changed
- **THEN** it is rejected with "Editor changed; try again", and the draft is unchanged

#### Scenario: Unknown region
- **WHEN** an Overflow edit names a region index that does not exist
- **THEN** the edit is refused, and the draft is unchanged

### Requirement: Visible page sets
The editor SHALL show how many sets the edited page has, and whether they are reached with Up/Down keys or by the page dial. It SHALL name every source that shares a region's rows.

#### Scenario: Page without scroll keys
- **WHEN** the edited page has no Up/Down key and its regions expand to three sets
- **THEN** the Regions panel shows `3 sets · page dial visits each`

#### Scenario: Page with scroll keys
- **WHEN** the edited page binds Up or Down and has two sets
- **THEN** the Regions panel shows `2 sets · Up/Down`

#### Scenario: Source disappears
- **WHEN** a provider stops and its collection has no members
- **THEN** the set count is recomputed from the remaining controls, and the region keeps its saved source

#### Scenario: Shared rows
- **WHEN** a region's rows are shared by Now playing and App audio
- **THEN** its row shows the Now playing source picker and `+ App audio`

### Requirement: Make Home is a real change
Make Home SHALL be unavailable on the page that is already Home and SHALL NOT mark the draft unsaved there.

#### Scenario: Already Home
- **WHEN** the edited page is Home
- **THEN** Make Home is disabled, and a stale press leaves the draft clean
