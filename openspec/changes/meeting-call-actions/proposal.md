# Focused meeting actions

## Why
The user's live check confirms Mute and Camera work, but Share and Leave still
target only Discord. Deafen is incorrectly offered while Discord has no call.
The user requests call-focus buttons, defaulting to the most recent call.

## What Changes
- Keep Mute and Camera universal. Add one runtime focus for Share, Leave and
  Deafen, with a button for each joined compatible call and newest-call default.
- Support Meet Leave and its presentation workflow through the existing companion.
  Opening a sharing menu or chooser is not confirmation that sharing started.
- Show Deafen as unavailable/N/A outside a known supported joined call, including
  when Meet is focused. Preserve local Discord controls with corrected gating.
- Reuse existing provider workers, bounded command queues and exact lifetimes;
  do not add transports, dependencies, automatic retries or persistent focus.

## Superseded Scope
This supersedes the Discord-only Share/Leave scope in universal-meeting-state,
the Share/Leave exclusions in google-meet-controls, and the older meetings design
allowing Deafen when merely connected. Other shared-state rules stay unchanged.

## Impact
Meetings coordinator and adapters, browser DTO validation and companion, Meetings
GUI, focused tests/fixtures, UI contract and requested Plus XL Meetings bindings.
Preserve all smaller-deck layouts, shipped defaults and unrelated saved settings.
Browser/site acceptance requires a new desktop-control approval. Never choose a
capture source or start broadcasting on the user's behalf during inspection.
