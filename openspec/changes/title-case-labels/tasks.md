# Tasks
## Implementation
- [x] 1. `docs(ui)`: specify title-case built-in labels and the Mic Path rename; update the UI contract.
- [x] 2. `feat(plugins)`: Mic Path, and title-case control, option, collection, group and report labels in every plugin; tests.
- [x] 3. `feat(gui)`: title-case screen titles, headings, buttons and field labels; GUI check.
- [x] 4. Validate: gofmt, `go test ./...`, `go vet ./...`, GUI checks, OpenSpec strict validation, canonical build; restart Snoofer. Race detector skipped: CGO is disabled on this toolchain.

## Acceptance
- [ ] 5. On the deck and in the GUI, built-in labels read in title case ("Record Mic", "Mic Path"); song, app and room names are unchanged.
