# Tasks

## 1. Explicit source selection

- [x] 1.1 Add draft source-list state and Enter/Space opening, arrow/j/k navigation, Enter confirmation and Escape/Tab cancellation; verify opening, browsing, cancellation and same-choice confirmation submit no edits, and changed confirmation submits exactly one revisioned edit.
- [x] 1.2 Remove the 0 shortcut and update source-control documentation; test 0 is inert in every view and picker state, stale drafts cancel safely, device disconnect/reconnect does not change the draft, and save failure preserves active intent.

## 2. Recording layout and compact chrome

- [x] 2.1 Reorder and relabel recording settings, render Pre/Post, and separate Start/Stop with a nonselectable spacer and Actions heading; test row-key focus, skipped separators, Enter-only transport, profile absence and unchanged saved mic_tap behavior; update recording-control docs.
- [x] 2.2 Remove title, idle notice filler, second footer and dashboard exit reminder; test dynamic content height, selected-row visibility, source-picker bounds and absence of removed text across narrow/wide/tiny layouts with notices; preserve NO_COLOR and text sanitization.

## 3. Concise feedback

- [x] 3.1 Implement short queued/saved/pending/applied/reloaded notices with three-second success expiry and persistent error priority; test expiry with deterministic time, preview versus live convergence, unresolved routes, full Events diagnostics and failure after success; document visible status meanings.

## 4. Integration and terminal acceptance

- [x] 4.1 Run Go tests, vet, Windows build and strict OpenSpec validation; record automated results and verify routing/recorder regression behavior remains unchanged.
- [x] 4.2 Run an isolated dry-run terminal smoke covering picker confirm/cancel, preview save, resize, recording-row order and removed text against the annotated screenshot; record visual results separately from prior real-device/audio acceptance, without claiming unperformed hardware checks.
