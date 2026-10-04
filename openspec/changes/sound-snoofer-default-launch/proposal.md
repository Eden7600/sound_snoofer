# Proposal

## Why
The project is now Sound Snoofer. Normal use should require only opening the executable, not understanding CLI commands and flags.

## What Changes
- Rename source/module, executable, branding and project directory to Sound Snoofer / sound-snoofer / sound_snoofer.
- Launch live TUI by default with no arguments; accept top-level flags and explicit `tui`.
- Use config.json beside the executable by default for TUI; create it from bundled defaults if missing. Preserve existing config and sidecars.
- Make --dry-run the opt-in preview flag for TUI and watch; retain --apply compatibility and explicit diagnostics commands.

## Capabilities
### New Capabilities
- `desktop-launch`: Default live TUI launch with adjacent persistent configuration and renamed identity.
### Modified Capabilities
None; canonical specs are empty.

## Impact
Project paths, Go module/imports, app identity, CLI, config provisioning, docs/spec references, tests and local executable/config deployment. Git history remains intact.
