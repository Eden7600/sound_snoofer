# Design

## Source and dependencies
Use `app/web/src/app.ts` as the entry point and `screens/` for Audio, Routing, Soundboard,
Lights, App Audio, Meetings, Media, Stream Deck, Plugins, Third-Party Apps and Diagnostics.
Keep each screen's existing private helpers with that screen.

Shared modules keep the existing responsibilities:
- `types.ts`: Go wire data and host binding types; provider view types live with their screen
  or in focused adjacent type files where several consumers need them.
- `model.ts`: existing pure helpers, with their tests preserved.
- `dom.ts`: utility strings, element creation, icons and unbound DOM helpers.
- `state.ts`: the current snapshot, controls map, pending/error state and widget/updater lists,
  plus the existing update/notice rendering. Keep a simple mutable object, not a store or event bus.
- `host.ts`: the typed Desktop binding and existing send/request/press behavior.
- `widgets.ts`: bound controls, meters, mixers, artwork and shared clip presentation.

Dependency direction is types/model/DOM → state → host → widgets → screens. The entry point
imports the screen builders and owns polling, navigation, layout signatures and rebuilds.
Screens never import the entry point. Keep the existing callback arrays and pending behavior.

## Types and behavior
Mirror `app/state.go`, `snoofer/controls.go` and the provider ViewData structs, including nil
slices/maps as nullable values and Go times as strings. `Control.ViewData` is unknown unless
narrowed at the provider access boundary; it is not an unrestricted production `any`.
Use typed DOM factories and references, guard optional controls/array entries, and avoid
blanket non-null assertions or TypeScript suppression directives.

Preserve behavior even when accurate types reveal an already tracked defect. In particular,
Routing's legacy Activity key reads remain explicitly isolated and documented until
`gui-content-fixes`; do not invent false uppercase fields in the wire type or silently repair
the casing during extraction. Keep the expected-failure test and exact known layout exceptions.

## Tooling and packaging
- Pin TypeScript, esbuild, ESLint/typescript-eslint, Prettier and required type packages as dev
  dependencies. Use strict no-emit checking with `noUncheckedIndexedAccess` for source and tests.
- Use the recommended typescript-eslint configuration and Prettier on frontend source/tests and
  their tooling files. Do not reformat unrelated Go, scripts, generated fixtures or vendored data.
- Build the browser entry as one bundled `dist/app.js`. The installed Node supports stripping
  TypeScript for the pure model tests; no additional test runner is needed.
- Keep icon generation reproducible. If icons move to TypeScript, update the generator rather
  than editing its generated output by hand.
- `index.html` loads the bundle. Go embeds the HTML and generated dist directory; source modules
  and tests are not runtime assets. Tailwind scans HTML and source modules, not tests or the bundle.
- Both the canonical build and browser setup generate JS and CSS. The build rejects either
  missing/empty output. `check.ps1` runs type-check, lint, format, model and browser checks by default.

## Verification
Run the existing model and browser suites, including fixture drift checks and both viewport
layouts. Confirm lint/type/format gates fail on a representative violation and pass after removal.
Build only through `scripts/build.ps1`, preserve the running host's launch arguments, and run
the native desktop smoke against the canonical executable. Inspect representative packaged
screens for missing assets/styles; source-served tests alone cannot verify embedding.
Leave hardware acceptance unclaimed. The GUI contract changes only in later behavior changes.
