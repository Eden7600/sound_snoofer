## ADDED Requirements
### Requirement: Title-case clip labels
A soundboard clip's label SHALL be its file stem in title case, with underscores and hyphens shown as spaces. The clip's ID and artwork matching SHALL still use the file name.

#### Scenario: Snake-case file
- **WHEN** the folder holds `air_horn.mp3`
- **THEN** the clip is labelled "Air Horn" on the deck and in the GUI

#### Scenario: Minor words and acronyms
- **WHEN** the folder holds `rick_and_morty.mp3` and `OMG-wow.mp3`
- **THEN** they are labelled "Rick and Morty" and "OMG Wow"

#### Scenario: Existing bindings and artwork
- **WHEN** `air_horn.mp3` has `air_horn.png` beside it and is bound to a deck key
- **THEN** the key keeps its binding and shows the artwork
