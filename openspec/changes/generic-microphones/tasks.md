# Tasks
## Implementation
- [x] 1. `feat(config)`: microphones schema, interface input map (mono/stereo, legacy array), legacy synthesis, ID validation everywhere, edition limit; tests.
- [x] 2. `feat(routing)`: strips by position, patches, device inputs, options, fallback, managed strips, patch allowlist; planner tests including the legacy matrix unchanged.
- [ ] 3. `feat(audio)`: labels from names, activity strips from config, microphone edits; tests.
- [ ] 4. `feat(gui)`: Microphones card and interface channel fields; deck labels; GUI check; UI contract; AGENTS.md invariants.
- [ ] 5. Validate: gofmt, `go test ./...`, `go vet ./...`, GUI check, OpenSpec strict validation, canonical build.

## Acceptance
- [ ] 6. The live setup routes exactly as before; rename the lav, add a second device microphone and see both on the Routing screen and the deck.
