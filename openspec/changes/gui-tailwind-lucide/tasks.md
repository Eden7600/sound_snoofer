# Tasks
Each numbered block is one reviewable commit.

## Implementation
- [x] 1. `docs(gui)`: specify the Tailwind and Lucide migration.
- [x] 2. `build(gui)`: Tailwind v4 CLI pipeline. Add pinned dev dependencies, `app/web/tailwind.css` with theme tokens (temporarily also carrying the legacy rules, so the UI is unchanged), `npm run css`, gitignored `app/web/dist`, the embed update and build/check script integration.
- [ ] 3. `feat(gui)`: vendored Lucide icons (`scripts/vendor-icons.cjs` → `app/web/icons.mjs`) replacing every glyph icon.
- [ ] 4. `refactor(gui)`: move markup to Tailwind utilities, delete the legacy stylesheet rules, switch test hooks to `data-*`, and update check-gui/check-desktop, the reference images and the UI contract.
- [ ] 5. Validate (GUI checks, model tests, Go checks, OpenSpec, canonical build), inspect the screens, relaunch and record the results.
