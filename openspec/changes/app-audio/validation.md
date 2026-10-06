# Validation
2026-10-06, after commits e52ef0c through e0b928c.

## Automated checks
- `go vet ./...` and `go test ./... -timeout 120s`: pass. The appaudio fake-backend tests were also run 10 times in a row without failure. `go test -race` was not run because cgo is disabled.
- `scripts/check-gui.cjs` covers:
  - App audio strips in pick order;
  - a closed app without volume controls;
  - the slider sending `set`, and Mute sending `press`;
  - the Pin, Hide and Unhide edits;
  - dial-range selection and adding a dial region.
- `scripts/check-desktop.cjs` and `openspec validate app-audio --strict`: pass.
- `scripts/build.ps1`: built.

## Native probes
- **Read-only probe** (`SNOOFER_READ_ONLY_PROBE=1`): sessions were listed on every active render device (Voicemeeter inputs, the Arena speakers and the Volt monitor). Each had its name, path, PID, volume, mute, peak and a 64 px icon (Brave, Hue Sync, Steam, Snoofer, Voicemeeter).
- **Write probe** (`SNOOFER_SESSION_WRITE_PROBE=1`): the System sounds session's volume and mute were rewritten with their current values and read back unchanged. This confirms that the float argument passes correctly.
- **Live preview** (the plugin against the real backend with Live off, so no writes):
  - Brave was heard and shown at its Windows volume (54%), and Hue Sync was heard.
  - System sounds merged its 10 device sessions into one app.
  - Voicemeeter was hidden by default; Steam and Orca were listed but silent.

## Personal setup
- `appaudio` enabled.
- Apps page added: a key region r1 c1–c5, a dial region on dials 1–5, and Up/Down at keys 18 and 27.
- Home key 34 (r4c7) set to Go to Apps.
- `snoofer.exe --check` passes. The backup is `snoofer.json.before-apps` in the session scratchpad.

Snoofer was relaunched with automation `NO_COLOR` cleared. Hardware acceptance (task 7) is pending.

## Revision: exclusion list (commits 33b2fcd, 8cd6ccb)
- `go vet ./...` and `go test ./...`: pass. The appaudio tests were run 5 times in a row without failure. They cover:
  - default exclusions, including a heard Hue Sync session;
  - wildcard and System sounds matching;
  - Hide adding file names, and Unhide removing a wildcard default;
  - exclude/include for programs that are not running;
  - pattern validation, and absent versus empty lists.
- `scripts/check-gui.cjs` covers the Excluded card: patterns with "Hiding …", Remove (`include`), Exclude by file name, and Unhide for apps hidden by a rule.
- `scripts/check-desktop.cjs` and OpenSpec validation pass. Built with `scripts/build.ps1`; `snoofer.exe --check` passes. Snoofer was relaunched.
- **Personal config:** `appaudio` settings have no `exclude` key, so the defaults apply and Hue Sync is excluded. The first edit in the GUI saves the explicit list.
