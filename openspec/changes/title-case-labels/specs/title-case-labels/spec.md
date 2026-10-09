## ADDED Requirements
### Requirement: Title-case built-in labels
Built-in labels SHALL use title case on the deck and in the GUI. Sentences, status words and content names SHALL keep their own case.

#### Scenario: Recorder key
- **WHEN** the Record mic control is shown
- **THEN** its label reads "Record Mic"

#### Scenario: Mic path
- **WHEN** the mic's Direct/Element selection is shown
- **THEN** it is labelled "Mic Path"

#### Scenario: Minor words and acronyms
- **WHEN** a label contains a short preposition or an acronym, such as "Apps on the deck" or "VR playback"
- **THEN** it reads "Apps on the Deck" and "VR Playback"

#### Scenario: Content names
- **WHEN** a media session's title is "lofi hip hop radio"
- **THEN** the title is shown exactly as the player reports it

#### Scenario: Status words
- **WHEN** a control reports the status "VR override"
- **THEN** the status text and every comparison of it are unchanged
