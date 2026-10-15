# Explicit commitment for live dropdowns

## Why
The canonical WebView2 GUI confirmed audit item 2: Down on a focused, closed Processing select
immediately sent a live `audio.normal-mode` set request. Browsing must not write settings.
The shared selection widget and Routing's direct source/channel/activity/device selects all
dispatch on `change` and need the same keyboard commitment rule.

## What Changes
- Preserve native select controls and pointer selection. Closed-select keyboard navigation and
  typeahead stage a local choice; Enter commits once, Escape or leaving the field cancels it.
- Preserve focus-time revisions and stale-list identity checks when a choice is committed.
- Apply this only to selectors that directly change provider settings. Form selections with an
  existing Add/Apply button and Stream Deck draft/navigation selections retain their semantics.
- Verify browser keyboard/pointer journeys and the canonical WebView2 executable with isolated
  input/output fixtures. No test writes to a live audio provider.

## Impact
`app/web/src/widgets.ts`, a small shared select-event helper, live selects in Routing, focused
browser tests, the UI contract and a targeted native test. No new dependency, custom dropdown,
action queue, routing policy or persistence change. Implement after GUI content corrections.
