# Attention markers and plugin recovery screens

## Why

GUI audit item 14 leaves disabled or failed plugins with inconsistent screens and no reliable
route to recovery. The sidebar cannot show where a problem lives. Reusing the existing control
tone would incorrectly flag ordinary mute, and parsing status text would confuse failures with
descriptive detail or remembered errors.

## What Changes

- Publish typed plugin lifecycle state alongside the existing descriptive plugin status text.
- Use one small plugin screen gate for missing, disabled, starting and failed providers. Keep
  existing enabled-screen layouts and empty-content states.
- Add one severity marker to a navigation button only when its destination contains a current,
  explicitly classified problem. Reuse typed connection reports and provider-owned failure
  fields; add optional attention metadata where no structured signal currently exists.
- Use the existing Enable confirmation and Plugins retry journey. Navigation itself sends no
  action. The shared header remains transient-only.

## Impact

Host lifecycle snapshot and ViewState; optional Control attention metadata; narrow publication
sites for Audio, AEC, App Audio, Soundboard, Now Playing and Stream Deck; shared frontend state,
navigation and plugin gates; provider-backed fixtures, focused tests and UI contract.

No new dependency, automatic retry, persistent notification history, lifecycle policy change,
new screen, settings change or audio write is introduced. Action submission/outcome ownership
stays in gui-action-outcomes; restoration and confirmations stay in gui-state-preservation.

This checkpoint prepares the plan only. Implementation follows review of the complete proposal,
design, requirements and tasks, with overlapping provider files coordinated with the action
outcomes owner.
