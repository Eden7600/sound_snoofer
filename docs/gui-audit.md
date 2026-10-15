# GUI audit

Audit of the desktop GUI (`app/web`) against [ui-contract.md](ui-contract.md), dated 2026-10-14.
It covers every screen, navigation, layout and the GUI test harness. It is a planning input, not a
specification: each fix below goes through its own OpenSpec change before implementation.

**Baseline:** `node scripts/check-gui.cjs` passes in about 27 s, and `node --test app/web/model.test.mjs` passes.
Screenshots were reviewed at 1280×820 and 800×600 using the browser fixture.

## Reliability

1. **Clicks are silently dropped.** `send` returns early while any action is pending
   (`app/web/app.js:71`), and every control on the screen is disabled for up to 2.5 s after any
   action. Pressing Mic Mute and then Playback Mute quickly loses the second press without feedback.
2. **Keyboard browsing of a dropdown may write live settings (unverified).** Every selection applies
   on `onchange` (`app.js:112`, plus the mic source, channel mode and region source selects). In
   Chromium on Windows, arrow keys on a closed `<select>` fire `change` per step, so arrowing through
   Processing or Monitor would apply each option. That conflicts with the rule that browsing never
   writes audio settings. The deck binding listbox (`app.js:1031`) changes the draft on every arrow
   press. This still needs to be confirmed in WebView2 with a test.
3. **Rebuilds discard UI state.** Any structural change rebuilds the whole screen (`layoutKey`,
   `app.js:1197`), for example a browser tab starting to play on Media or App Audio. This collapses
   open `<details>` (App Audio "More", Other Rooms, Page Options), loses scroll position and focus,
   and clears unapplied text in fields that are not focused.
4. **Errors follow the user across screens.** In the test run, "Control changed; try again" stayed
   visible from Audio through Soundboard, Lights, App Audio and Diagnostics. The contract asks for
   errors to stay local.
5. **Connection loss looks like an error.** Each failed poll calls `showError`, with no backoff and no
   distinct reconnecting state.
6. **Confirmations are inconsistent.** Delete Page, Forget Deck and turning Own Layout off use the
   native `window.confirm`, which also blocks polling. Host confirmations use the styled `<dialog>`.

## Content and vocabulary

7. **Processing and Monitor show raw values** ("direct", "element", "off", "pre", "post"). The audio
   plugin sets no `OptionLabels` for mode or monitor (`plugins/audio/audio.go:419`). The contract
   uses Direct / Element and Off / Pre / Post.
8. **"virtual:1 Playback" appears on the Audio screen.** This internal label (`audio.go:368`,
   group "Shared Audio") shows in the Routing & Recovery card and duplicates the Computer cell of the
   Routing matrix. The card filter only excludes the "Routing" group (`app.js:210`).
9. **Internal names on other screens.**
   - Stream Deck: the page selector shows page IDs ("home"), and the binding list shows provider
     labels with plugin suffixes ("mic gain · audio").
   - Plugins: shows plugin IDs (`appaudio`, `nowplaying`) instead of names.
   - Echo Cancellation: the Retry row has no label.
10. **Diagnostics → Current Notices lists every status message.**
    - "VR override" appears four times with no profile named.
    - "Music: Speakers (Arena 7)" is not a notice, and Start at Login is repeated.
    - Notices are not grouped by plugin and cannot be copied.
    - The Audio Engine card repeats controls already on the Audio screen.

## Layout

11. **Audio.**
    - Cards stretch to their row: Recording pads to the Echo card's height, leaving a large empty block.
    - In the three-column Live Controls row, Mic Stack, Mic Mute and Playback Mute sit far from
      their switches.
    - The Interface row is stranded below the gain strips.
12. **Routing.**
    - In the half-width Playback card, Up/Down/Remove wrap onto their own line.
    - Each microphone name has its own Apply button.
    - The page is about 2,000 px tall with no in-page navigation.
    - Output slot management (device, rename, remove) is packed into the table headers.
13. **Stream Deck.**
    - The sticky Save bar covers the Page panel because the page has no bottom padding.
    - The Device and Page selectors are misaligned.
    - "Edit Shared Binding: Off" has no context.
14. **Other.**
    - Meter scale labels collide ("-3 0") on narrow strips (App Audio, 800 px).
    - Soundboard search has no "no matches" state.
    - Disabled-plugin screens are handled four different ways (Enable card, "Enable it in Plugins",
      `pluginOff`, "disabled or editor not ready").
    - The sidebar gives no hint of which screen needs attention.

## Code health and tests

- `app/web/app.js` is 1,260 lines: all 11 screens, widgets and global mutable state in one module,
  mostly in dense one-line style. Only `model.mjs` has unit tests (77 lines).
- `scripts/check-gui.cjs` is a single 681-line script:
  - One fixture is shared and mutated by every step, which is why the error in item 4 bled across
    screens.
  - It stops at the first failure and runs only with `scripts/check.ps1 -GUI`.
  - It writes screenshots but never compares them.
  - It has no accessibility checks and no keyboard-journey coverage.
- The fixture is written by hand and has drifted from the real providers. That is how items 7 and 8
  went unnoticed.

## Proposed changes

Each change is its own OpenSpec change, split into commit-sized tasks. Recommended order:

1. **`gui-test-harness`.** Move to Playwright Test:
   - A fresh fixture per test, and one spec file per screen.
   - Keyboard-journey tests, accessibility scans (axe-core), and screenshot comparisons at
     1280×820 and 800×600.
   - Fixtures generated from the real providers in preview, so labels cannot drift.
   - Run by default in `scripts/check.ps1`; the native desktop smoke stays opt-in.
   - Port the existing assertions first, with no behavior change.
2. **`gui-content-fixes`.** Items 7–9 and the Retry label.
3. **`gui-action-reliability`.** Items 1–6:
   - Track pending actions per control, and never drop a click silently.
   - Apply dropdowns only on an explicit choice.
   - Rebuild screens without losing their state.
   - Scope errors to where they happened, and add a reconnecting state.
   - Use the styled dialog for all confirmations.
4. **`gui-modules`.** Split `app.js` into per-screen modules and shared widgets so each part can be
   unit-tested.
5. **`gui-layout-polish`.** Items 10–14, screen by screen, with the contract updated where the UX
   changes.
6. **`gui-attention-journeys`.** Sidebar attention markers and consistent disabled-plugin cards.
   This requires a contract update, because the header is limited to transient badges.

## Open decisions

- Add `@playwright/test` (same version as `playwright`, 1.62.1) and `@axe-core/playwright` as dev
  dependencies?
- Commit screenshot baselines (binary churn on UI changes), or check DOM geometry only?
- Approve the structural split of `app.js` in change 4?
- Write the step 1 proposal first, or all six proposals before any implementation?
