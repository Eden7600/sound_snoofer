# Stable content during GUI polling

## Why

Repository audit minor finding 26 remains in the TypeScript implementation.
App Audio replaces every More detail row on each updater call, destroying selected
program text even when only the relative Heard value changes. Media recreates the
play/pause SVG on every poll although its state usually has not changed. The prior
transport and state-preservation changes did not remove these local replacements.

## What Changes

- Create App Audio's six detail rows once per app card. Update only text that changed.
- Replace the Media play/pause icon only when the displayed playback state changes.
- Keep metadata, relative Heard times, clocks, accessibility and action behavior live.
- Add focused polling regressions and reconcile both existing repository audit reports.

## Impact

Only the two screen builders, their browser tests, the UI contract and audit records.
No dependencies, shared rendering framework, settings, provider or action changes.
Native lost-click behavior is not established by the original report; this change
addresses the confirmed repeated node replacement without claiming that diagnosis.
