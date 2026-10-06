# Validation
2026-10-06, after commits fa372e4, e943335 and 352a794.

- `go vet ./...` and `go test ./... -timeout 120s`: pass. `go test -race` was not run because cgo is disabled.
- `scripts/check-gui.cjs`, `scripts/check-desktop.cjs` and `openspec validate soundboard-clip-timers --strict`: pass.
- `scripts/build.ps1`: built; `snoofer.exe --check` passes.
- Strip preview, rendered at native size and reviewed:
  - one timer per panel with artwork, a large `m:ss` and the name;
  - two per panel in rows;
  - the Play symbol when there is no artwork;
  - unused panels left blank.
- The personal layout needs no change: Soundboard dials 2–5 are unbound.
- Snoofer was relaunched with automation `NO_COLOR` cleared.

Hardware acceptance (task 5) is pending.
