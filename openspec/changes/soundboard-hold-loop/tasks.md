# Tasks
Each numbered block is one reviewable commit.

## Implementation
- [x] 1. `docs(soundboard)`: specify hold to loop; update the UI contract and plugin docs.
- [ ] 2. `feat(soundboard)`: loop held clips until Stop. Covers clip `hold`, the loop flag through the queue and preparation, voice length and looping restart, readiness before polling, and the Loop value. Tests cover restart on completion, Stop, replacement, eviction, restart failure and timers.
- [ ] 3. `feat(deck)`: show Loop on clip keys in the active colour. Tests cover the accent and that Loop is shown.
- [ ] 4. `feat(gui)`: hold a clip button to loop it. Covers pointer hold, click suppression and the Loop tone. GUI check covers hold, a short click and the Loop state.
- [ ] 5. Validate: Go tests and vet, GUI and desktop checks, OpenSpec, the canonical build; relaunch.

## Hardware acceptance
- [ ] 6. On the Stream Deck + XL: holding a clip loops it with Loop shown, a tap plays once, Stop ends the loop, and Overlap on/off behave as specified. In the GUI, a held clip loops and a click plays once.
