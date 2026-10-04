# Verification — 2026-10-03

Implemented the source picker, removed the 0 shortcut, reordered recording
settings, separated Actions, shortened stage values, reclaimed title/footer/idle
notice rows and added typed notice lifecycle handling.

Automated checks passed:
- `go test ./... -timeout 30s`
- `go vet ./...`
- Windows build to `bin/sound-snoofer.exe`
- `openspec validate streamline-tui-controls --strict`

Tests cover picker navigation/confirmation/cancellation, unchanged selections,
stale revisions, ordinary refresh, inactive background shortcuts, recording row
order, bounded rendering, removed text, NO_COLOR, sanitization and deterministic
notice expiry/convergence/error priority. Existing save-failure and routing tests
remain passing. Race detection was attempted but could not run: this Go toolchain
has CGO disabled. No toolchain settings were changed to bypass that limitation.

Interactive Windows terminal smoke used `.local/smoke/config.json`, isolated from
personal saved choices. Verified at 80 columns and after resizing the console to
120×30: compact layout, grouped recording settings and Actions, source list with
saved-choice marker, cancel without a change, explicit Off confirmation, Saved ·
Preview feedback and disappearance after the success interval. The isolated
sidecar persisted source=off. Reopening the picker marked Off as current. Both
sessions exited cleanly. Automated tests additionally cover tiny/narrow layouts.

No live mixer or transport writes were requested. Earlier real-device listening,
recording-file and physical hotplug acceptance remain separate and unperformed.

Repository housekeeping requested during implementation:
- Configured repository-local Git identity as Eden7600 <odycay12@proton.me>.
- Verified and archived 20 original work files in
  `.local/archives/work-20261003-161647.zip`, then removed the loose scratch files.
  Historical verification paths under work/ now refer to entries in this archive.
- Kept archives, smoke fixtures, personal configuration, binaries and dependencies
  ignored; added the Conventional Commits requirement to AGENTS.md.
