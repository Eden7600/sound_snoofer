# Tasks
Each numbered block is one reviewable commit.

## Implementation
- [x] 1. `docs(deck)`: specify page regions, go-to keys, the Lights page and the motion toggle. Record the UX review and update the UI contract's deck and Hue sections.
- [ ] 2. `feat(hue)`: motion toggle for the selected room. Covers:
  - the `motion` resource in the model and event merges;
  - the association to the room;
  - On/Off/Mixed values, Pending until observed, and Failed;
  - the `hue-motion` icon;
  - the Lights screen Motion row.

  Tests use a fake bridge, plus a native-size icon preview.
- [ ] 3. `feat(snoofer)`: control collections. Covers:
  - the `Collection`/`CollectionLabel` fields;
  - the soundboard clips collection;
  - the Hue room-scene and per-room scene collections;
  - room-scene slots sized to the room, with unused slots Hidden.

  Tests cover the published metadata.
- [ ] 4. `feat(streamdeck)`: page regions. Covers:
  - the model;
  - validation;
  - expansion with sequential same-source fill and parallel overflow;
  - legacy `auto_controls` compatibility;
  - the editor view's region data.

  Tests cover a fixed frame, overflow, manual cells, Hidden candidates, overlap rejection and legacy equivalence.
- [ ] 5. `feat(streamdeck)`: go-to page keys and the `deck-page` icon. Covers publishing, press handling, Here state and binding eligibility. Includes tests and a native-size preview.
- [ ] 6. `feat(gui)`: region editor. Covers:
  - rectangle selection (Shift+click, Shift+arrows);
  - the Regions list with a source picker and Remove;
  - region tint and number;
  - one-based key numbering;
  - removal of the prefix field.

  The GUI check covers creation, source change, removal and selection never dispatching.
- [ ] 7. `feat(streamdeck)`: default layout with Soundboard and Lights pages. Includes tests.
- [ ] 8. Validate:
  - Go tests and vet; GUI and desktop checks; OpenSpec; the canonical build.
  - Migrate the personal layout (§4) with the editor or an equivalent stopped-host edit.
  - Relaunch and record the results.

## Hardware acceptance
- [ ] 9. On the Stream Deck + XL and the bridge:
  - Home shows the slimmer Hue block with Motion, and the go-to keys reach Lights and Soundboard;
  - Motion pauses and restores the room's sensors (as confirmed in the Hue app);
  - the Room key changes scenes and Motion together;
  - soundboard overflow keeps Stop, Overlap and Home fixed.
