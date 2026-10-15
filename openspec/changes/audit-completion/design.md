# Design

## Worker protocol (19)
All callers use individual Action rows and the existing Complete callback. Remove
Action.ID, Action.Edits, State.Acks, EditAck/EditError, seen/dedup capacity state,
SettingEdit, maxDeferredEdits, EditsRow and EditBatch. Keep Action.Origin because
production audio/Discord/Soundboard callers use it to clear prior Notice. Keep
Action.Complete, ClaimCompletion, GUI IDs/reservations and windowsaudio.Request.Ack.
Call EditIntent directly and preserve bare/Normal/VR device availability checks.
Retain individual feedback, mute generations/lifetimes, contextual revisions,
save-before-apply, restart confirmation, recording invariants and cancellation.
Convert acknowledgement-based tests to actual outcomes/state; discard only tests
whose sole contract was the unused batch/dedup protocol. Preserve failed-save,
stale-device/restart, feedback clearing, latency and settled-readback assertions.
Update conflicting older spec clauses to refer to the surviving callback outcomes.

## Small shared operations (20, 29)
Add internal/artwork.Thumbnail(image.Image, size int, scaler draw.Scaler) returning
the same base64 PNG and an error. Share only proportional fit, transparent NRGBA
canvas, scaling and PNG/base64 encoding. Now Playing stays 128px CatmullRom;
Soundboard stays 64px ApproxBiLinear. Keep caller decoding, dimensional/file limits,
GIF logical canvas/disposal/timing, animation limits, cache and fallbacks local.
Characterize current pixels before extracting, retain source bounds/transparency
and source immutability, and guard invalid bounds/size without changing valid output.
Add snoofer.OnOff(bool) string with exact On/Off words; replace both local helpers
and all eight callers. Do not add a generalized rendering framework.

## Browser connection admission (25)
Use the original audit's reject-second-live-connection option: an incumbent is
retained regardless of a new claimant's extension origin. A declined connection
must publish no sessions, receive no commands, cancel no pending incumbent command
and consume no incumbent generation. After incumbent cleanup, a fresh connection
may register and receives a fresh generation. Keep duplicate checks synchronized
and preserve close/cancellation/reconnect behavior. Test same/different origins,
concurrent claims, incumbent command routing and genuine disconnect/reconnect.
This does not authenticate the first claimant. Chrome has no published pinned ID;
Firefox's manifest ID is not its profile-specific moz-extension origin. Do not
invent IDs, pairing, persistent trust state or an extension publishing prerequisite.

## Acceptance and boundaries
Run focused tests and meaningful negative/characterization checks per checkpoint,
then full Go tests/vet/race, frontend gates/browser checks, strict specs and the
canonical build. Record executable command/configuration and restore the tray host.
Portable compiler installation is user-approved; verify the official distribution,
use a local tool directory and process-scoped compiler settings only.

Remaining live checks must have a committed bounded preparation before execution.
For a refused second deck, revisit only a verified smaller-Plus path with a read-only
holder that performs no report I/O: access-zero/share-read may deny Snoofer's write
open while coexisting with unrelated read handles. Require a proven sharing denial,
explicit release, a hard at-most-90-second deadline and cleanup before host restore.
Never change device ACLs/drivers or close unrelated applications to make setup work.

A frozen-player/uncertain-action test must use an attributable, uniquely owned
player and a bounded reversible stall, while another session stays responsive.
First establish feasibility without touching a user's running player. Do not count
fake source tests, an absent session or fixture-only Unknown as live acceptance.
Do not mark hardware checks passed merely because setup was attempted. Record the
no-dial model waiver explicitly. The prior anonymous shutdown timeout stays historical
unless fresh evidence identifies a cause; preserve named shutdown diagnostics and
do not invent an unverified repair. Ask before desktop input/visible launches.

## Validation fixture ownership
The full checks exposed two test setup defects. Stream Deck outcome tests can
submit against remembered-device controls before the fake Connected event changes
the editor epoch; the correct stale rejection then leaves an unbounded test wait.
Wait for the connected device report before dispatch, bound save-entry waits and
report premature outcomes. Remove the device-report test's unnecessary frame drain,
which waits forever on a channel the fake surface never closes. Demonstrate the
ordering with a controlled fake schedule and repeat affected tests across CPUs.

The browser bridge test starts its worker before populating fake Windows sessions,
racing Snapshot. Initialize that fixture before starting the worker rather than
adding synchronization to the synchronous unit-test rig. Preserve the duplicate
Windows/tab session assertion and run the bridge tests with the race detector.
No production timing, stale-action guard or queue semantics changes are intended.
Further full-suite failures require confirmed diagnosis before their repair.
