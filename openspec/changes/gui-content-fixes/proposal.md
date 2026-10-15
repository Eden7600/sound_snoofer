# GUI content corrections

## Why
The provider-backed GUI fixtures expose the content defects in `docs/gui-audit.md` items 7–9,
15 and 16. Audio selections all receive the microphone label map; Routing reads Activity keys
with the wrong case; playback route switches leak into Audio's recovery card; and plugin IDs
appear where plugin display names are available. The AEC Retry button has text, but the generic
command widget leaves an empty caption beside it.

One audit subfinding is already resolved: the Stream Deck provider publishes page names in
`streamdeck.page.OptionLabels`, and the generated fixture contains `home: Home`. Retain that
behavior with a browser assertion; do not add another page-name formatter.

## What Changes
- Correct audio option labels at the provider, preserving configured microphone/device names and
  the Automatic, Off and silent-microphone semantics.
- Read the real Activity JSON keys and remove the tracked expected failure and migration shortcut.
- Classify playback source sends as Routing, using the existing source-name vocabulary, and show
  AEC Retry as a compact action without an empty field caption.
- Carry registered plugin labels through the existing host snapshot. Use them in Plugins,
  Third-Party Apps' Not monitored list and deck binding context, with ID fallback.
- Preserve Normal/VR qualifiers in audio binding labels; keep IDs as option values and tooltips.
- Regenerate affected provider fixtures and add focused assertions at each commit checkpoint.

## Impact
- Code: `plugins/audio`, `app/state.go`, `app/tray_windows.go`, typed GUI state and affected
  screens, browser fixtures/tests and `docs/ui-contract.md` where presentation is clarified.
- No action ordering, dropdown commitment, pending-state, confirmation, persistence or routing
  behavior changes. No plugin ID, control ID, option value, saved binding or compact ShortLabel
  changes. No hardware access is needed for fixtures.
- Depends on the completed `gui-typescript-modules` extraction. Deck startup validation (repo
  finding 32), layout polish, action outcomes and repository hardening remain separate changes.
