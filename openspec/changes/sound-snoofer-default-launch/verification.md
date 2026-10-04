# Verification

- Go tests, vet, Windows build and strict OpenSpec validation passed in the renamed project directory.
- Launch tests confirm no-argument and explicit TUI startup are live, --dry-run selects preview, default config resolves beside the executable and explicit config overrides it. Watch defaults live and releases writer ownership.
- Default provisioning tests confirm valid bundled config, preservation of existing files and errors on unwritable paths.
- Preview terminal smoke launched bin/sound-snoofer.exe --dry-run from Documents/Codex with no config flag. It opened the TUI and loaded the preserved Lav/Direct choices. Quit cleanly without mixer writes.
- Personal config and sidecar copied to bin/config.json and bin/config.json.state.json, with matching hashes. Originals remain available.
- User renamed the folder after Windows released its lock. Git history remains intact; renamed source builds successfully. Repaired 169 local dependency junctions whose absolute targets still referenced the prior directory.
- Maintained tracked references use Sound Snoofer naming; historical Git objects and zipped scratch archives remain historical. Removed the closed, superseded executable.
