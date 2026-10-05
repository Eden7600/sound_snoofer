## Why
Capture settings and tape rehearsal are currently mixed together. Recording sources, clip destinations and transport actions need distinct roles so the TUI and Stream Deck behave predictably.
## What changes
Separate Recording Sources from Playback Targets. Add configured named soundboard clips with a one-shot load-and-play action. Allow speakers/headphones and Discord/apps destinations separately or together. Element remains exclusively the live voice processor. Preserve native recording and existing source preferences, and migrate legacy rehearsal choices deliberately.
## Impact
Configuration and preference migration, routing planner, native recorder adapter, shared actor, TUI groups, Stream Deck clip bindings and integration tests. This change is a design proposal; implementation tasks remain unchecked.
