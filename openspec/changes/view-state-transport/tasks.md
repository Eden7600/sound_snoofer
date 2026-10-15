# Tasks

## Planning checkpoint
- [x] 1. Review and commit the proposal, design and requirements; pass strict OpenSpec validation before implementation.

## Implementation checkpoint
- [x] 2. Add failing transport/desktop tests for complete change detection, telemetry with stable revisions, asset removal and explicit empty-map encoding, queue replacement/reversion/focus, encoding failure recovery, provider immutability, new sessions, poll cursors and legacy frames.
- [x] 3. Implement stripped session snapshots, complete asset-map comparison, versioning/coalescing and the desktop cursor API; preserve configuration/action paths and existing bounded shutdown.
- [x] 4. Integrate central frontend hydration and animation generations; adapt the fake host and add focused browser regressions for null-poll local updates, artwork lifecycle/navigation, complete animation changes and stale lookups. Preserve existing fixture assertions and polling error recovery.
- [x] 5. Add/run the representative payload benchmark; record pipe/IPC bytes and allocations for initial, unchanged, telemetry-only and asset-changing snapshots against full-state encoding.

## Automated validation and commit
- [ ] 6. Run gofmt, focused and full Go tests, `go vet ./...`, model tests, TypeScript/lint/format checks, GUI tests and strict OpenSpec validation; commit the complete validated transport change.
- [ ] 7. Coordinate `scripts/build.ps1` and the existing native desktop smoke without modifying its script; verify legacy fixtures, revisioned actions, focus, close/reopen and EOF cleanup. Report race-detector results or the existing toolchain limitation.

## Live acceptance
- [ ] 8. With live media or soundboard artwork, verify telemetry and clocks continue, animation/artwork changes and removal render correctly, and closing/reopening controls restores assets. Leave this unchecked if hardware acceptance is not performed.
