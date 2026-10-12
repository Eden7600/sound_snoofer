# Validation
2026-10-12, after commits 7d113f2, 67bbd39 and 89d91cb.

- `go vet ./...` and `go test ./... -timeout 120s`: pass. `go test -race` was not run because cgo is disabled.
- `node --test app/web/model.test.mjs` and `scripts/check-gui.cjs`: pass. The GUI check covers a click sending one press, a 500 ms hold sending one hold and no press, and Loop showing the playing state; with click swallowing removed it fails as expected.
- `scripts/check-desktop.cjs`: fails on the Lights screen (`Start sync` expected, `Start Sync` shown), unrelated to this change.
- `openspec validate soundboard-hold-loop --strict`: pass.
- `scripts/build.ps1`: built; `snoofer.exe --check` passes. Snoofer was relaunched with its prior `--config bin/snoofer.json` command line.

Hardware acceptance (task 6) is pending.
