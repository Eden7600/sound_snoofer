# Keep tape routing while it plays
## Why
Pressing Play routes the tape to the Playback destination, and the next reconciliation (about 20 ms later) removes that send. The tape plays silently.

`routing.ProfileConfig` restores the saved base configuration before resolving profiles, and that drops the runtime-only `TapeListening` flag the controller has just set. The final convergence check in `applyTopology` and the gain-target check also build plans from `Controller.Config` without the flag, and the worker's UI plan omits it too. Existing tests miss the bug because their controller has no `Profiles`, so the base swap never happens.

## What Changes
- `ProfileConfig` keeps `TapeListening` from its input when it swaps in the saved base.
- Every plan the controller and the worker build carries the controller's current `TapeListening`.
- A regression test runs tape listen-back with profiles configured.

## Impact
- **Code:** `internal/routing/profiles.go`, `internal/controller` (plan configuration), `internal/control/worker.go` (UI plan).
- **Invariants:** unchanged. The tape send still follows the Playback destination only while Snoofer's Play is in effect and is cleared after Stop.
