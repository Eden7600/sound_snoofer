# Tasks
## Implementation
- [x] 1. `docs(gui)`: specify the GUI test harness; record the TypeScript decisions and new order in `docs/gui-audit.md`.
- [x] 2. `test(gui)`: add `@playwright/test` and `@axe-core/playwright`, config, routed page serving, fake host and the ported hand fixture; Chrome check; Audio specs ported.
- [x] 3. `test(gui)`: port Routing specs.
- [x] 4. `test(gui)`: port Stream Deck specs.
- [x] 5. `test(gui)`: port Soundboard, Lights, App Audio, Meetings, Media, Plugins, Apps and Diagnostics specs; remove `check-gui.cjs`; `check.ps1` runs the suite by default.
- [x] 6. `test(audio)`: golden GUI fixtures from the audio provider (live and preview), with drift test; specs use them.
- [x] 7. `test(deck)`: golden GUI fixture from the Stream Deck editor, with drift test; specs use it.
- [x] 8. `test(gui)`: keyboard journeys and cross-screen journeys.
- [x] 9. `test(gui)`: axe-core scans with recorded known violations.
- [ ] 10. `test(gui)`: layout geometry at both viewports.
- [ ] 11. Validate: full suite within the 60 s budget (record the time), each spec file alone, `go test ./...`, `go vet ./...`, `scripts/check.ps1`, OpenSpec strict validation.
