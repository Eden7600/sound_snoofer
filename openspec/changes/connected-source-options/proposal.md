# Proposal

## Why
The source picker offers absent microphones, and computer playback cannot be selected directly in the TUI.

## What Changes
- Offer only connected, unambiguous configured microphones, always including Off; installed ASIO alone is insufficient.
- Add a persistent Playback Device selector: Automatic plus connected physical WDM outputs matched by configured playback candidates.
- Prefer the selected device when available; fall back to automatic priority when disconnected and return when it reconnects.

## Capabilities
### New Capabilities
- `device-choice-controls`: Connected microphone choices and explicit playback preference.
### Modified Capabilities
None; canonical specs are empty.

## Impact
Config intent, routing selection, TUI selectors, worker availability, tests and docs. Retain existing regex ownership, output slot placement and migration rules.
