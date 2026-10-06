# Tasks
Each numbered block is one reviewable commit.

## Implementation
- [x] 1. `docs(deck)`: specify scroll keys and animated artwork; update the UI contract.
- [x] 2. `feat(streamdeck)`: scroll keys. Covers the published controls, local press handling, binding eligibility (plugin and GUI picker), page-dial stops, `deck-up`/`deck-down` icons and the default Soundboard page. Tests cover paging, wrapping, single set, dial order and the defaults.
- [x] 3. `feat(soundboard)`: animated GIF artwork. Covers the `Animation` field, compositing and limits, the artwork cache, deck frame selection and refresh cadence, and GUI playback through `Animation(id)`. Tests cover disposal, delays, the frame limit, the cache and frame selection.
- [x] 4. Validate: Go tests and vet, GUI and desktop checks, OpenSpec, the canonical build; bind Up/Down on the personal Soundboard page; relaunch.

## Hardware acceptance
- [ ] 5. On the Stream Deck + XL: Up/Down page the clips with Overlap and Stop fixed, the page dial skips the second set, and GIF clips animate on keys and in the GUI.
