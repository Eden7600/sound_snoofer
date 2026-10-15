# Tasks

- [x] 1. `docs(deck)`: review and commit this focused plan before implementation; wait for
  `gui-typescript-modules` to land.
- [x] 2. `fix(deck)`: add the shared Go compatibility rule, inert status input and focused
  regressions for validation, startup with audio present, editor options and fake hardware
  input. Retain missing-provider, geometry, save and stale-input checks.
- [x] 3. `fix(gui)`: align TypeScript compatibility and model/deck tests; update the UI contract's
  display-only key description. Change the fixture generator to publish audio before startup,
  regenerate the Stream Deck fixture and review the option changes.
- [x] 4. Validate focused Go and fixture-drift tests, frontend gates/model tests, the deck
  browser tests and full GUI suite; inspect native-size status rendering and record results.
  Run the repository checks/canonical build and restore the prior launch configuration if
  a running host was stopped. Do not claim unperformed physical hardware acceptance.

2026-10-15: inspected freshly generated .local/deck-status-acceptance.png from TestRoutingPresentation at native 112px per key. Destination In use/Missing/No output labels, icons and state colors remain legible; real provider-to-binding fixture tests and full package/source gates pass. This is offline renderer acceptance; physical status-key checks remain unclaimed.
