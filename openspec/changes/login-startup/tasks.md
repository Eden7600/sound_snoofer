# Tasks
## Implementation
- [x] 1. `docs(startup)`: specify login startup; update the UI contract, README and application guide.
- [x] 2. `feat(startup)`: Run entry read/write with StartupApproved handling, `core.startup` control, tray action with read-back and preview gating; unit tests and scratch-key registry tests.
- [x] 3. `feat(gui)`: Startup card on Diagnostics; GUI check for the toggle row and its press request.
- [ ] 4. Validate: gofmt, `go test ./...`, `go vet ./...`, GUI check, OpenSpec strict validation, canonical build.

## Acceptance
- [ ] 5. Turning Start at Login on in the GUI creates the Run entry and shows On; turning it off removes it and shows Off.
- [ ] 6. With Start at Login On, signing out and in starts Snoofer in the tray with the personal configuration and no controls window.
- [ ] 7. Disabling Snoofer in Task Manager's Startup apps shows Off with the Windows note; turning it on in the GUI re-enables it.
