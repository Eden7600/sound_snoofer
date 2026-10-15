# Tasks

User-authorized scope: shared mute and camera targets; changes in any compatible
call app flow back outwards. Camera clears after the last compatible call ends.
Like/Dislike placement applies only to Plus XL. Desktop acceptance remains deferred.

- [x] 1. Commit the resolved state/lifecycle/persistence/echo contract and update
  conflicting earlier design and UI language before implementation.
- [x] 2. Implement and test one shared owner, audio preference registration,
  target publication, exact intent revisions and bounded cleanup. Prove saved mute
  authority, runtime-only audio-absent operation, camera reset and no feedback/replay.
- [x] 3. Adapt Discord's mute and camera with confirmed explicit writes, preserved
  deafen behavior, exact lifetime/revision checks, and compatibility bindings.
- [x] 4. Adapt all eligible Meet calls to shared intent, preserving browser command
  identity, readback, cancellation, freshness and per-control pending limits.
  Add and validate per-control observation identity so node replacement baselines
  without propagating a false external edit; test older missing-identity reports.
- [x] 5. Reuse Meetings Mute/Camera controls in the GUI with actual per-app state,
  truthful pending/unknown/error, independent provider gates and native key checks.
- [x] 6. Run full Go/vet/race, extension regression where affected, frontend/GUI
  gates and strict specs; canonical build and restore prior launch configuration.
- [ ] 7. During a newly approved live session, apply only requested Plus XL Home
  and Meetings placements, verify both apps and external changes, two Meet calls,
  new/last call camera behavior, reconnect uncertainty and cleanup. Verify and
  restore tested song ratings. Leave unavailable checks explicitly unchecked.

Commit each validated implementation block; keep unrelated user changes intact.
