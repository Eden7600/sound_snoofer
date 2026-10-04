# Proposal

## Why
The UI displays only worker-confirmed choices. During native apply/readback, subsequent clicks appear ineffective and commands sharing an old revision can be rejected.

## What Changes
- Show queued setting choices immediately and continue accepting edits during reconciliation.
- Keep one edit batch in flight and collect subsequent edits locally; dispatch them with the latest acknowledged revision.
- Preserve serialized native calls, save-before-apply and explicit failure feedback.

## Capabilities
### New Capabilities
- `responsive-settings`: Responsive desired-choice editing while audio operations run.
### Modified Capabilities
None; canonical specs are empty.

## Impact
TUI model and worker edit protocol, tests, docs. No concurrent mixer calls or changes to routing invariants.
