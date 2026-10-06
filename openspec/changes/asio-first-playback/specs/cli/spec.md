## ADDED Requirements
### Requirement: Redirected maintenance commands need no console
The Windows CLI SHALL preserve valid redirected stdout and stderr and SHALL NOT allocate or attach a console when both streams already exist.
#### Scenario: Read-only validation from automation
- **WHEN** an audio plan command runs with redirected output and no console window
- **THEN** it returns its output through those streams without opening a terminal or an allocation-error dialog
