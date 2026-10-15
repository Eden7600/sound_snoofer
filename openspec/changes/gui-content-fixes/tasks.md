# Tasks

Implementation starts after `gui-typescript-modules` lands. Each item is one reviewable commit
with its focused checks; retain unrelated GUI defects and user changes.

- [x] 1. `docs(gui)`: specify content corrections and their existing provider/fixture paths; correct
  the audit's current-status notes (completed harness/TypeScript, page labels already correct,
  current full binding labels, exact tracked narrow-deck overlap) without erasing the dated baseline.
- [x] 2. `fix(audio)`: assign control-specific option labels; extend provider tests for active,
  Normal/VR, tap, Automatic/Off and silent microphones; regenerate both audio fixtures and then
  Stream Deck; assert rendered Audio option labels and Routing's Pre label. Run drift tests.
- [x] 3. `fix(gui)`: read Activity's lowercase fields; remove legacy cast and expected failure;
  assert initial 2/-70/10 values, absent Activity, no render/focus dispatch and all edit payloads.
- [x] 4. `fix(gui)`: classify/name playback sends with existing Routing/source metadata and render
  Retry as a compact command action; regenerate affected fixtures, verify no duplicate recovery
  switches, unchanged Routing dispatch and Retry availability/press; inspect Audio at both sizes.
- [x] 5. `feat(gui)`: carry registered plugin labels through ViewState and use them in plugin
  headings, Not monitored and binding context; qualify full Normal/VR provider labels; retain
  raw IDs, custom names, search and missing-label fallback. Update wire types/fixtures and focused
  host/provider/browser tests; regenerate affected audio and Stream Deck fixtures.
- [x] 6. `test(gui)`: protect already-correct built-in/custom page OptionLabels with an unchanged-ID
  selection assertion; reconcile the UI contract for the intentional presentation changes and
  inspect native-size deck renderer output for affected option labels.
- [ ] 7. Validate: frontend type/lint/format/model/browser checks, provider fixture drift,
  `go test ./...`, `go vet ./...`, OpenSpec strict validation, canonical build and packaged desktop
  smoke; inspect affected screens at both viewport sizes and restore the prior host launch.

Hardware behavior is unchanged and hardware acceptance is not claimed by these checks.
