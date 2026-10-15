# Tasks

- [x] 1. Review this plan against the landed transport/recovery frontend and enumerate explicit
  element keys and edit-context boundaries. Strictly validate the change before source edits.
- [x] 2. Commit checkpoint: preserve focus, scroll, details and searches around same-screen
  structural builds. Add failing regressions for App Audio More, Lights Other Rooms, Deck Page
  Options/binding search and Soundboard search, stable control focus and removed-target fallback.
  Verify restored filtering, both viewport sizes and no action during restoration.
- [x] 3. Commit checkpoint: retain dirty explicit-commit fields and form choices with their edit
  identity/revision. Cover unfocused drafts, unchanged-target rebuilds, removed options, renamed
  apps, deck page/profile/serial/slot boundaries and reordered Routing lists. Keep focused select
  deferral, dropdown cancellation and stale revision-zero behavior; verify deferred rebuilds after
  blur when transport returns null. Prove explicit cancel/observed completion clear draft state.
  Verify Routing Add choices retain their payload meaning, clear when raw option numbers are
  reused for different devices, and create an output without device fields for explicit No device.
- [x] 4. Commit checkpoint: replace the three Deck native confirmations with the shared styled
  dialog. Cover Cancel/Escape, exact one-shot accept, continued polling/rebuilds, revision/target
  invalidation, missing/disabled controls, reconnection, host-prompt priority and focus return.
  Preserve each existing warning and provider draft/save semantics; enabling Own Layout stays direct.
- [ ] 5. Update UI contract and audit status for these intentional changes. Run strict type/lint/
  format, model and complete isolated GUI checks within the harness budget; inspect dialog layout
  at 1280×820 and 800×600 and run strict OpenSpec validation.
- [ ] 6. Coordinate the canonical build and
  packaged WebView2 acceptance of state restoration, dialog cancellation, continued polling and
  captured-target rejection. Restore the prior launch; hardware integration acceptance is separate.

- [ ] 5a. Add a failing shared-notice Dismiss focus regression, retain the rendered button across unchanged polls, and verify dismissal and full GUI checks without relaxing timeouts.
