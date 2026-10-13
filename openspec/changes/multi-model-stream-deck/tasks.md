# Tasks
Each numbered block is one reviewable commit.

## Implementation
- [x] 1. `docs(deck)`: specify multi-model and multi-deck support; update the UI contract and application guide.
- [x] 2. `feat(streamdeck)`: a model table, per-model input decoding, image reports (key, + strip, Neo screen) and key rendering at native size and orientation. Tests cover each model's decoding, report headers, key sizes and orientation, and byte-identical + XL output.
- [x] 3. `feat(streamdeck)`: run a session per connected deck. Covers discovery of all supported PIDs, per-deck frames and events, per-deck failure and reconnection, and bounded shutdown of every session. Tests cover two decks, one failing, and shutdown.
- [ ] 4. `feat(deck)`: size layouts to each model and ship per-model defaults. Covers slice-backed pages, column-aware regions, the reserved Pages key, padding and validation, the `models` settings and defaults for the +, Neo, MK.2 and XL. Tests cover legacy + XL decode, validation by geometry and that every default validates.
- [ ] 5. `feat(deck)`: per-deck pages, Pages key, Neo touch paging, Paging setting, remembered decks and per-deck connection reports. Tests cover independent and mirrored paging, Pages key press and hold, routing input to the right deck, first-connection recording, Own Layout, Forget and rename.
- [ ] 6. `feat(gui)`: a device-shaped deck editor. Covers grid and dials from the view geometry, the reserved Pages key/dial, model and deck targets, deck name, Own Layout, Forget (confirmed) and Paging. GUI check covers the + and MK.2 geometries.
- [ ] 7. Validate: Go tests, vet and race; GUI and desktop checks; OpenSpec; the canonical build; native-size renders of each model reviewed; read-only probe of the connected +; relaunch.

## Hardware acceptance
- [ ] 8. Stream Deck +: keys upright and legible, dials 1–3 act, dial 4 pages and returns Home, strip shows four panels in order, brightness and dark mode work.
- [ ] 9. + XL unchanged with an existing layout; + XL and + together, independent and mirrored paging; unplugging one leaves the other working.
- [ ] 10. Remembered deck: rename, Own Layout edit, Forget, and the deck is recalled after replug and relaunch.
- [ ] 11. Unverified models (MK.2/Original v2, XL, Neo): orientation, Pages key or Neo touch buttons and screen. Unchecked until tested on hardware.
