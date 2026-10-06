# Validation

## Automated (2026-10-05)
- `npm run css` (Tailwind CLI 4.3.3) generates `app/web/dist/app.css` (about 25 KB minified). `scripts/build.ps1` ran it before compiling and passed its missing/empty check.
- `node --test app/web/model.test.mjs` passes. `check-gui.cjs` passes on the migrated markup, with all selectors now using `data-part` hooks or accessible names.
- `go vet ./...` and `go test ./...` pass without the generated CSS (the embed of `all:web/dist` keeps the tracked `.gitkeep`).
- `check-desktop.cjs` passes against the new binary in the real WebView2 window (Lights / Start sync, revisioned actions, close/reopen, graceful tray shutdown).
- Strict OpenSpec validation passes.

## Visual inspection
The Audio, Soundboard, Lights, Stream Deck, Third-party apps and narrow (800 px) captures were compared with the previous references. Layout, palette, navigation, cards, deck grid and breakpoints match. Intentional differences: Lucide icons in the nav, buttons, placeholders and deck previews; minus and plus icons; slightly lighter heading weight. The deck screen's sticky save bar overlaps the Page panel in full-page captures, as before the migration. The references in docs/design were regenerated.

## Notes
- `npm install` left the optional `@parcel/watcher` install script unapproved. It is only used by `tailwindcss --watch`; one-shot builds do not need it.
- `pnpm-lock.yaml` predates this change and is stale; npm's `package-lock.json` is the lockfile in use.
