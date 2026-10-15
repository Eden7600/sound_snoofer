# TypeScript GUI modules

## Why
The GUI harness now protects the existing screens, but `app/web/app.js` still combines every
screen, rendering helper, host action and mutable UI value. The user chose strict TypeScript,
an esbuild bundle, typescript-eslint and Prettier, with the harness completed first and the
split before content fixes. This change implements that decision without changing behavior.

## What Changes
- Move application source into `app/web/src`: one module per screen, shared DOM/widgets,
  state, host bindings, pure model helpers and typed host/provider data.
- Enable strict TypeScript including `noUncheckedIndexedAccess`, lint and formatting checks;
  include the browser tests in type-checking.
- Bundle the entry point with esbuild to `app/web/dist/app.js`; build the CSS from source modules.
- Update the canonical build, browser fixture setup and embedded asset list together.
- Preserve existing tests, provider fixtures and explicitly tracked failures. GUI fixes and the
  remaining repository hardening follow this extraction.

## Impact
- Code: `app/web`, `package.json`, `package-lock.json`, frontend tooling configuration,
  `scripts/build.ps1`, `scripts/check.ps1`, `scripts/vendor-icons.cjs` and `app/desktop_windows.go`.
- No framework, new state-management mechanism, control IDs, saved-setting changes or new actions.
- Native audio behavior, pending/stale-action semantics and the UI contract remain unchanged.
