# Validation
2026-10-06, after commits 884d1db, ecbcce5 and 8675069.

- `go vet ./...` and `go test ./... -timeout 120s`: pass. `go test -race` is not run because cgo is disabled on this toolchain.
- `node --test app/web/model.test.mjs`, `scripts/check-gui.cjs` (now checks GIF frames cycling on a Soundboard card) and `scripts/check-desktop.cjs`: pass.
- `openspec validate soundboard-paging-animated-art --strict`: valid.
- `scripts/build.ps1`: built `bin/snoofer.exe`.
- Library probe: the 7 GIF-backed clips in the personal library decode to 10–35 frames (23–88 KB of thumbnails each); a full scan with an empty cache took 84 ms, and rescans reuse the cache.
- Personal layout: Soundboard keys 18 and 27 (r2c9, r3c9) bound to Up and Down (both were free); `snoofer.exe --check` passes. Backup at the session scratchpad `snoofer.json.before-scroll`.
- Relaunched Snoofer with automation `NO_COLOR` cleared.
- Icon review: the Up and Down arrows were rendered at native key size and checked.

Hardware acceptance (task 5) is pending.
