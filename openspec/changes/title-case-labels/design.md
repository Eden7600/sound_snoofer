# Design
## Method
1. **Mapping:** a reviewed mapping of exact label literals (old → new) is applied to the files that define or name each label: plugin sources and tests, `app/web/app.js` and `scripts/check-gui.cjs`.
2. **Excluded strings:** a literal that also appears as a status, an error, or a configuration context name is excluded from that file. For example, `"VR playback"` in `internal/config` names a compile context, not a label.
3. **Dynamic labels** follow the rule at their fixed parts: `source + " Playback"`, `"Scenes · " + room`, `"Go to " + page`.
4. **GUI literals** are mapped the same way. Their placeholders and notes are sentences and stay as they are.

## Identifiers
- **Compared names:** the GUI and plugins compare some group names: `Recording`, `Routing`, `Bindings`, and the `Normal` prefix. Single-word names are already title case.
- **Multi-word groups:** "Shared audio", "Normal microphone", "VR playback" and other multi-word groups change in every file that compares them, in the same commit.
- **Short labels:** `plugins/audio/labels.go` derives prefixed short labels by trimming "Mic ". This still works for "Mic Target" and "Mic Path" (the prefixed mode label stays "FX").

## Verification
- Go tests and `scripts/check-gui.cjs` pass with the new labels.
- A grep finds no remaining sentence-case built-in label literals in plugin sources.
- The deck preview shows "Record Mic" and "Mic Path".
