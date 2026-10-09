# Tasks
## Implementation
- [x] 1. `docs(meetings)`: specify removing the Insta360 integration; update plugin docs and the UI contract.
- [ ] 2. `refactor!: remove the Insta360 integration`: plugin, camera package and DLL source, build script and tag, default layout keys, deck icons, GUI card and checks; tests.
- [ ] 3. Validate: gofmt, `go test ./...`, `go vet ./...`, GUI check, OpenSpec strict validation, canonical build.
- [ ] 4. Personal settings: back up and clean `bin/snoofer.json` while Snoofer is stopped; restart Snoofer.

## Acceptance
- [ ] 5. Snoofer starts with no Insta360 plugin listed, the Meetings screen and deck page show no camera, and the webcam microphone still works.
