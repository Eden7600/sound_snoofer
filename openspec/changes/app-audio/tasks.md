# Tasks
Each numbered block is one reviewable commit.

## Implementation
- [x] 1. `docs(apps)`: design the App audio plugin, rules, dial regions and screen; update the UI contract.
- [x] 2. `feat(windowsaudio)`: session adapter. Covers enumeration across render devices, process path, name and icon, volume, mute and peak. Pure helpers are unit-tested; an opt-in native probe test reads live sessions and round-trips a volume.
- [x] 3. `feat(appaudio)`: plugin. Covers rules and defaults, grouping, combined volume and mute, pending/observed/ignored, recency by meter, picked order, edits and preview mode. Fake-backend tests cover each scenario.
- [x] 4. `feat(streamdeck)`: dial regions. Covers the model, validation, expansion with key streams paging in step, the editor view and the default Apps page with its Home go-to key. Includes tests.
- [x] 5. `feat(gui)`: App audio screen. Covers strips, pin and order, Rename/Combine/Hide, the hidden list, details and dial-region editing. The GUI check covers each action.
- [x] 6. Validate: Go tests and vet, GUI and desktop checks, OpenSpec, the canonical build; add the Apps page to the personal layout; relaunch.

## Revisions after review
- [x] 8. `docs(apps)`: an editable exclusion list with Hue Sync excluded by default (§6); update the contract.
- [x] 9. `feat(appaudio)`: the exclusion list. Covers the setting, its defaults, wildcard matching, exclude/include/hide/unhide edits and the GUI Excluded card. Fake-backend tests and the GUI check cover it.
- [x] 10. Validate: the canonical build and checks; relaunch.
- [ ] 11. `feat(appaudio)`: configurable recent window (§7). Covers the `recent` edit, its validation and the GUI minutes field. A fake-backend test and the GUI check cover it.

## Hardware acceptance
- [ ] 7. With real apps:
  - Discord, a browser and a game group correctly;
  - the dials and keys change volume and mute, and the Windows mixer agrees;
  - silent apps drop off after 5 minutes;
  - Hide, Rename and Combine behave;
  - Hue Sync is excluded, and excluded programs can be added and removed;
  - Up/Down pages dials and keys together.
