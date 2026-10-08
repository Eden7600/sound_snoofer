## ADDED Requirements
### Requirement: Separate combined apps
The App audio screen SHALL list, for each app, the rules that rename or combine programs into it. It SHALL remove a chosen rule on request.

#### Scenario: Uncombine
- **WHEN** Chrome was combined into Discord and the user chooses Separate on that rule
- **THEN** the rule is removed and saved, and Chrome appears as its own app again

#### Scenario: Undo a rename
- **WHEN** the user separates an app's own rename rule
- **THEN** the app returns to the name Windows reports

#### Scenario: Rule already gone
- **WHEN** the rule no longer exists because the settings changed elsewhere
- **THEN** nothing is saved and the error is shown on the App audio screen
