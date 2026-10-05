## Why
Deck labels repeat information conveyed by icons, key borders add clutter, and processing/stage icons do not distinguish their meaning. Builds have also drifted between commands and output names.

## What Changes
- Codify expert-oriented terse UI language and borderless Studio icons.
- Give controls optional compact labels without losing context in the TUI.
- Distinguish Direct/Element and Pre/Post with signal-path icons.
- Use scripts/build.ps1 for the sole app output bin/snoofer.exe; check.ps1 delegates to it.

## Capabilities
### New Capabilities
- `terse-deck-ui`: Compact, borderless Stream Deck presentation and a canonical build workflow.

## Impact
Control presentation metadata, provider labels, deck renderer, development scripts/guidance. Keep bindings, personal configuration, gain meters, commands and audio routing unchanged.
