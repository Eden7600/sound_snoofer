## ADDED Requirements
### Requirement: Page navigation context
The reserved page dial SHALL show previous, current and next page names in that vertical order, highlighting the current page.
#### Scenario: Cyclic navigation
- **WHEN** the displayed page is first or last
- **THEN** neighboring names wrap through the same order used by dial rotation
#### Scenario: Small layout
- **WHEN** only one or two pages exist
- **THEN** the names reflect the real cyclic neighbors even when repeated
#### Scenario: Generated pages
- **WHEN** automatic pages are present
- **THEN** they participate in the displayed navigation order
#### Scenario: Preserved interaction
- **WHEN** the dial turns or is pressed
- **THEN** it still changes pages or returns Home respectively
