# GUI test harness
## Why
The GUI is about to be rewritten: `app/web/app.js` is split into TypeScript modules (`gui-typescript-modules`). The only protection today is `scripts/check-gui.cjs`:
- one 681-line script with a single fixture that every step mutates;
- it stops at the first failure and runs only with `scripts/check.ps1 -GUI`;
- it writes screenshots but never compares them, and runs no accessibility scans.

Its fixture is hand-written and has drifted from the real providers. That is how raw Processing/Monitor values and the "virtual:1 Playback" label went unnoticed ([docs/gui-audit.md](../../../docs/gui-audit.md)). A rewrite needs a net that fails on behavior changes, reproduces real provider output, and runs on every check.

## What Changes
- **Playwright Test** (`@playwright/test` 1.62.1, the installed `playwright` version) replaces `check-gui.cjs`:
  - specs grouped by screen, plus cross-screen journeys;
  - a fresh fixture and page per test;
  - parallel workers;
  - installed Chrome only, never downloading a browser.
- **Port first:** every existing assertion is ported with no application change, and `check-gui.cjs` is deleted only once the new suite covers it.
- **Provider fixtures:** golden Go tests in the audio and Stream Deck packages publish controls through the real provider code from fixed inputs, with live and preview variants. They write the browser fixtures on request and fail when the committed fixtures drift. Plugins without such inputs yet stay hand-written, listed with the reason.
- **Keyboard and accessibility:** keyboard journeys and axe-core scans of every screen. Existing violations are recorded with an owning change, and new ones fail.
- **Layout:** geometry assertions at 1280×820 and 800×600 for horizontal overflow, clipped controls and overlapping interactive elements. Screenshots are kept as review artifacts, not baselines.
- **Default:** `scripts/check.ps1` runs the suite by default; the native desktop smoke stays behind `-GUI`.

## Impact
- **Code:** `package.json`/`package-lock.json` (`@playwright/test`, `@axe-core/playwright`), `app/web/test/`, `plugins/audio`, `plugins/streamdeck` (golden tests only), `scripts/check.ps1`. `scripts/check-gui.cjs` is removed.
- **Application behavior:** unchanged. This change adds tests and test tooling only.
- **Invariants:** unchanged. Fixture generation uses fixed inputs and fake backends, with no installed devices, network services or personal settings.
