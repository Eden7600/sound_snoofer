# Design

Use existing Control.Group values in first-seen order. Left/right (also [ and ]) cycle sections; up/down select within the active section. Wide terminals show a section rail; narrow terminals show the active section and its position. Keep Controls/Plugins tabs and existing edit semantics. Preserve the selected control by ID across snapshots, and fall back safely if a group disappears. Editors use the full content width. Status appears once in the selected-control detail. Limit form width to 112 cells. Respect explicit NO_COLOR. Development launch instructions remove only automation-injected NO_COLOR when neither user nor machine settings specify it.

Commit checkpoints: specification and launch guidance; renderer/navigation with regression tests; validation and deployment report.

