# TypeScript migration validation

- Strict TypeScript (`strict`, `noUncheckedIndexedAccess`), recommended typescript-eslint and
  Prettier pass for the production source and browser tests. Deliberate missing wire-field,
  unchecked test-array, explicit-any and formatting probes each failed their gate; probes removed.
- All six original pure-model tests pass from TypeScript using Node's built-in type stripping.
- The bundled GUI retains all 95 browser cases, including the expected Activity casing failure
  and exact narrow-deck overlap exceptions. Full repository run: 13.6 s for Playwright.
- `scripts/check.ps1` passes Go tests (including provider fixture drift), frontend gates, model
  and browser suites, Go vet and strict OpenSpec validation. The canonical build generated and
  embedded both JS and CSS and restored the prior Snoofer launch arguments.
- Native WebView2 smoke passes accessible Lights/Start Sync, revisioned actions, window close,
  reopen, host EOF and scoped graceful tray shutdown. Its old `Start sync` expectation was
  corrected to the pre-existing `Start Sync` label; application wording did not change.
- Inspected the canonical executable's Audio, Routing and Stream Deck screens through Windows
  computer use with an isolated `__controls` fixture. Bundle, stylesheet, icons, card layouts and
  deck grid render correctly. Browsing these screens produced no semantic actions. The fixture
  exited normally after its input pipe closed. Screenshot evidence is in the task's tool history.
- The native dropdown probe confirmed audit item 2: focusing Processing, cancelling its popup,
  then pressing Down on the closed select immediately emitted `audio.normal-mode`, revision 35,
  operation `set`, value `element`. The isolated fixture never dispatched to a live provider.
  This is retained behavior and belongs to `gui-dropdown-commit`.

The race detector remains skipped because CGO is disabled; no physical hardware acceptance was
performed. `npm audit` reports six high-severity entries on the existing OpenSpec/Tailwind path
through braces/micromatch/fast-glob/Parcel; those versions predate this migration. No new tooling
dependency appears on the reported paths, and no forced downgrade was applied. Node emits a
harmless module-type detection warning for the TypeScript model test.
