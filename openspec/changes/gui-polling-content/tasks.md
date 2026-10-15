# Tasks

- [x] 1. Commit the reviewed, strictly validated plan before changing source or tests.
- [x] 2. Add focused browser regressions and verify they fail on the existing repeated replacements.
- [x] 3. Preserve App Audio detail nodes and guard Media icon replacement; update the UI contract without changing layout or action behavior.
- [x] 4. Run focused browser checks, strict TypeScript/ESLint/Prettier/model gates and the full browser suite; inspect both affected screens at 1280x820 and 800x600. Review and commit the source checkpoint.
- [ ] 5. Reconcile the existing audit files, record validation, and perform the authorized canonical build with the prior host configuration restored. Keep previously unperformed hardware checks explicitly deferred.

## Validation (2026-10-15)

- Both new regressions failed against unchanged source: App Audio lost selected
  Programs text while Heard advanced; Media detached its SVG during unchanged polls.
- The corrected regression uses Artist/Album updates for a stable card and models a
  real Title plus control Label change separately. Independent review found no
  remaining issue after this evidence correction.
- The focused screen run passed 13 tests; the final corrected pair passed again.
  Strict TypeScript, full ESLint/Prettier and all 15 model tests pass.
- The full installed-Chrome headless suite passes 297 tests in 44.2 seconds with
  four workers, within the 60-second budget. Strict change validation passes.
- Headless renderer inspection of App Audio More and Media passes at 1280x820 and
  800x600. No layout classes, settings or action dispatch paths changed.

Native lost-click behavior was not reproduced or claimed. The fix establishes
stable nodes during polling; it does not change the existing structural rebuild
policy. No desktop control was used for these checks.
