## Why
The deck has unnecessary transport controls and text-only keys. Simplify it for recording, routing and media, reserving vacant positions for future soundboard controls.
## What changes
Use a combined record toggle, remove default Stop Media/tape/Loop/To VST keys, place previous/play-pause/next at zero-based positions 27/28/29, and render recognizable code-drawn icons with concise labels and scoped status.
## Impact
Default Stream Deck layout, recording action dispatch, renderer, config action validation and tests. Existing custom serial profiles remain supported; no tape functions removed from TUI.
