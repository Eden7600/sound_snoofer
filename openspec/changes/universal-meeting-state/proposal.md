# Universal meeting state

The 2026-10-15 [focused call actions](../meeting-call-actions/design.md) follow-up
supersedes the Discord-only Share/Leave scope and adds explicit call focus.

## Why
The user wants one mute target and one camera target reflected in all compatible
call apps. Changes made inside Discord, Google Meet or a future compatible app
must flow back into the shared target and out to the other apps. Separate buttons
and competing mute reconcilers do not meet that requirement.

## What Changes
- Introduce one concrete shared-state owner, reusing the existing audio mute
  preference and provider workers. No new transport, dependency or cloud service.
- Mute follows the saved Snoofer Mic Mute preference when audio is available;
  camera exists only while compatible calls remain. An external app's observed
  change updates the shared target and synchronizes the other eligible calls.
- Reuse the existing Meetings Mute/Camera controls, with per-app observed state
  and uncertainty visible. Keep Discord-only Deafen, Share and Leave separate.
- Apply Like/Dislike only to the user's Plus XL Home r1c9/r2c9 during the next
  approved desktop session. Smaller-deck redesign and shipped defaults are out of
  scope. Use existing Meetings positions for the shared Mute/Camera controls.

## Superseded behavior
This user-directed change supersedes google-meet-controls' separate local mute,
selected-only universal actions and observation-only recovery policy. Current
shared targets may be applied to known eligible calls, including calls joining an
existing call period. A camera-on target never survives a period with no eligible
calls. Uncertain attempts are not blindly replayed. Existing exact identity,
revision, readback, preview and bounded-lifecycle safeguards remain.

## Impact
Shared state/services, audio preference registration, Discord and Meet adapters,
Meetings GUI and fixtures, native control previews, specifications and tests.
Meet observations gain an additive control identity counter; browser transport,
protocol version and command format remain unchanged. Live acceptance is deferred
by the user; background implementation and automated checks may proceed.
