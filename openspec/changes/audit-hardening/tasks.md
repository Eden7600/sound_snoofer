# Tasks
## Implementation
- [x] 1. `docs(audit)`: specify the audit hardening.
- [ ] 2. `fix(gui)`: send drawn revisions for index-based edits; GUI check.
- [ ] 3. `fix(nowplaying)`: bound WinRT media calls; build the media helper.
- [ ] 4. `fix(build)`: stage companions until the Go build succeeds; repair neural-AEC setup.
- [ ] 5. `fix(voicemeeter)`: treat "no callback registered" as a clean stop; callback probe.
- [ ] 6. `fix(deck)`: seed the key baseline from the current input report.
- [ ] 7. `fix(deck)`: report every deck path that fails to open; surface test.
- [ ] 8. `fix(hue)`: ignore light-write results from before a room change; test.
- [ ] 9. `fix(nowplaying)`: replace bridge connections only from the same origin; test.
- [ ] 10. Validate: gofmt, `go test ./...`, `go vet ./...`, GUI check, OpenSpec strict validation, canonical build.

## Acceptance
- [ ] 11. On hardware: restart Voicemeeter while Snoofer runs and confirm the monitor stays enabled; press a deck key once right after connecting it; connect two decks with one refused and see its error; freeze a media player and confirm Now Playing keeps updating.
