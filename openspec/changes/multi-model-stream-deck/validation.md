# Validation
2026-10-13, after commits b4eba2b, 499c4e1, 5cdb187, 61dc7ba and 9c086e5.

- `scripts/check.ps1 -Fuzz -GUI`: Go tests, GUI model tests, `go vet ./...`, `openspec validate --all --strict` (88 changes), config and HID decoder fuzzing (3 s each) and `scripts/build.ps1` pass. `go test -race` was skipped because no C toolchain is installed for cgo.
- `scripts/check-gui.cjs`: pass. Covers the + XL grid as before, the + grid (8 keys, 3 dials, reserved page dial, arrow navigation by 4 columns), the MK.2 grid (14 keys, reserved Pages key, no dial row), Paging, and Forget and Own Layout only after confirmation.
- `scripts/check-desktop.cjs`: fails on the Lights screen (`Start sync` expected, `Start Sync` shown), as recorded for soundboard-hold-loop; unrelated to this change.
- Native-size renders of every model reviewed: keys at 112 (+ XL), 120 (+), 96 (XL, Neo) and 72 px (MK.2), the + strip and the Neo screen. + XL key bytes are unchanged (test).
- Read-only probe (`SNOOFER_READ_ONLY_PROBE=1`): the connected Stream Deck + (0FD9:0084) is discovered and opened through the model table; input 512, output 1024, feature 32, serial A00WA6261J4HRK.
- `snoofer.exe --check`: pass. Snoofer was relaunched with its prior `--config bin/snoofer.json` command line; it connected the + and remembered it (`devices: {"A00WA6261J4HRK": {"model": "plus"}}`).

Hardware acceptance (tasks 8–11) is pending.
