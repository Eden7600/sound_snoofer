# Tasks
## Implementation
- [x] 1. `docs(audit)`: specify the audit hardening.
- [x] 2. `fix(gui)`: send priority and output edits from older lists as stale; GUI check (a targeted browser regression test belongs to `gui-test-harness`).
- [x] 3. `fix(nowplaying)`: bound WinRT media calls; build the media helper.
- [x] 4. `fix(build)`: stage all selected companions/models/licenses until esbuild, Tailwind and Go succeed; preserve the sole application output `bin/snoofer.exe`; repair neural-AEC setup; test failure/success/tagged paths and interrupted dependency recovery with isolated fake commands.
- [x] 5. `fix(voicemeeter)`: treat "no callback registered" as a clean stop; callback probe.
- [x] 6. `fix(deck)`: seed the key baseline from the current input report.
- [ ] 7. `fix(deck)`: report every deck path that fails to open; retire temporary path reports on recovery/removal; surface and report-lifecycle tests.
- [x] 8. `fix(hue)`: ignore light-write results from before a room change; test.
- [x] 9. `fix(nowplaying)`: replace bridge connections only from the same origin; test.
- [ ] 10. Validate: gofmt, `go test ./...`, `go vet ./...`, GUI check, OpenSpec strict validation, canonical build.

## Acceptance
- [ ] 11. On hardware: restart Voicemeeter while Snoofer runs and confirm the monitor stays enabled; press a deck key once right after connecting it; connect two decks with one refused and see its error; freeze a media player and confirm Now Playing keeps updating.
