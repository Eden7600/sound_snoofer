# Tasks
Each numbered block is one reviewable commit.

## Implementation
- [x] 1. `docs(apps)`: design the App audio plugin, rules, dial regions and screen; update the UI contract.
- [x] 2. `feat(windowsaudio)`: session adapter. Covers enumeration across render devices, process path, name and icon, volume, mute and peak. Pure helpers are unit-tested; an opt-in native probe test reads live sessions and round-trips a volume.
- [x] 3. `feat(appaudio)`: plugin. Covers rules and defaults, grouping, combined volume and mute, pending/observed/ignored, recency by meter, picked order, edits and preview mode. Fake-backend tests cover each scenario.
- [x] 4. `feat(streamdeck)`: dial regions. Covers the model, validation, expansion with key streams paging in step, the editor view and the default Apps page with its Home go-to key. Includes tests.
- [ ] 5. `feat(gui)`: App audio screen. Covers strips, pin and order, Rename/Combine/Hide, the hidden list, details and dial-region editing. The GUI check covers each action.
- [ ] 6. Validate: Go tests and vet, GUI and desktop checks, OpenSpec, the canonical build; add the Apps page to the personal layout; relaunch.

## Hardware acceptance
- [ ] 7. With real apps:
  - Discord, a browser and a game group correctly;
  - the dials and keys change volume and mute, and the Windows mixer agrees;
  - silent apps drop off after 5 minutes;
  - Hide, Rename and Combine behave;
  - Up/Down pages dials and keys together.
