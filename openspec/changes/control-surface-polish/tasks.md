# Tasks
Each numbered block is one reviewable commit.

## Implementation
- [x] 1. `docs(ui)`: specify control surface polish.
- [ ] 2. `feat(ui)`: remove the sidebar host footer and move audio health from the header to the Audio and Diagnostics screens. Update the GUI check and contract.
- [ ] 3. `feat(audio)`: mic controls hidden on the deck when the stack is off. Hidden becomes surface-only, and the deck ignores input on hidden bindings. Tests cover the audio controls and the deck dispatch.
- [ ] 4. `feat(soundboard)`: overlap setting, toggle control and deck icon; voice pool with an 8-voice cap and oldest-cut. Tests cover voice management with a fake player; contract and docs.
- [ ] 5. `feat(streamdeck)`: dial meter redesign (gradient bar, VU ballistics, peak hold, position track) and faster level sampling and refresh while meters show. Presentation tests and a native-size preview.
- [ ] 6. `feat(gui)`: mixer meter peak hold, smooth motion, scale and position track; faster polling on audio screens. Update the GUI check, screenshots and contract.
- [ ] 7. Validate (Go tests/vet, GUI and desktop checks, OpenSpec, canonical build), relaunch and record the results.

## Hardware acceptance
- [ ] 8. On the Stream Deck + XL: meter motion and peak hold look right at the new refresh rate; mic keys blank and return with the stack toggle; overlapping clips play together and Stop silences all.
