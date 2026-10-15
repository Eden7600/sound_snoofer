# Tasks

## Plan
- [x] 1. `docs(audit)`: review and commit this proposal before implementation; confirm
  `audit-hardening` native/media/build checkpoints have landed and retain the acknowledgement/batch,
  thumbnail and `onOff` deferrals.

## Implementation checkpoints
- [x] 2. `refactor(control)`: remove write-only publication fields and unused presentation helpers;
  preserve acknowledgement/batch plumbing and meaningful worker tests, including
  single-edit persistence failure, stale selection/restart rejection, feedback and latency.
- [x] 3. `test(storage)`: add the strict `Decode` table and valid/default/null cases; consolidate
  config/saved-choice decoding and direct journal/storage callers while retaining normalization,
  validation, state tokens, contextual errors and failed-load behavior.
- [x] 4. `refactor(go)`: share bus-number parsing with existing caller strictness, replace
  `firstNonEmpty` with `cmp.Or`, and share only cancel-and-wait; test bus boundaries and shutdown
  deadlines and retain plugin join tests.
- [ ] 5. `refactor(dead-code)`: recheck and remove finding 30's confirmed unused members/helpers;
  preserve the renderer, media and child-process behavior covered by their remaining tests.
- [x] 6. `refactor(build)`: consolidate the six native toolchain setup sites after rebasing on
  hardening; retain staging, output paths, compiler flags, media self-tests and failure context;
  run/update isolated fake-command build tests, including paths with spaces and setup failure.
- [ ] 7. `refactor(native)`: share buffer shape/channel/copy primitives in the existing header;
  retain strict callback pointer validation; run C/C++ buffer self-tests and compile affected
  companions through the canonical build workflow.

## Validation
- [ ] 8. Run gofmt, focused and full Go tests/vet, frontend gates/GUI tests, OpenSpec strict
  validation and `scripts/build.ps1`. Review deletion searches and final diffs for unrelated
  settings, generated runtime files and unintended public API changes.
- [ ] 9. If a running host was stopped for builds, restore its prior launch configuration and
  record actual native smoke results separately. Leave physical device acceptance unclaimed
  unless performed.

Acknowledgement/batch redesign or removal, thumbnail merging and cross-plugin `onOff` sharing are
deferred, not additional unchecked implementation tasks.
