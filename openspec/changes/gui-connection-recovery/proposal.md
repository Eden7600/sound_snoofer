# Host connection recovery

## Why
GUI audit item 5 reports every failed state poll as an action error and retries at the normal telemetry cadence. Users cannot distinguish a lost host connection from an action failure, and repeated connection errors overwrite useful feedback.

## What Changes
- Show one global Reconnecting banner between the shared header and action notice while state polling fails.
- Back off failed polls to a maximum of five seconds, returning to the screen's normal cadence after a successful response, including an unchanged response.
- Preserve the last screen and existing action errors. Refuse new actions explicitly while disconnected; never queue or replay them after recovery.
- Keep render failures separate from transport failures and always schedule the next poll.

## Impact
Shared frontend polling, state, send boundary, document markup, UI contract and isolated browser regressions. No host/provider API changes, dependencies or deployed-app changes. Action-specific pending/outcome attribution remains owned by gui-action-outcomes.
