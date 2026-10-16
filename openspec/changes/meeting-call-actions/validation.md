# Validation — 2026-10-15

## Implemented
Call focus defaults to the newest first-observed joined lifetime. Explicit focus
persists until that exact call ends. Focused Deafen/Share/Leave capture identity,
revision and lifetime; Mute/Camera remain universal. Unsupported Deafen is N/A.
Discord actions require a joined call. Meet Leave has a guarded, one-click DOM
adapter and requires observed departure. Optional Share transport and provider
support do not establish real-site presentation support.

Manual focus cancels a claimed but unsent Leave while its call is still current;
actual departure/lifetime replacement preserves the exact bounded result waiter.
Pending keeps capability availability for native Wait. Actionable unknown Share
reads Ready. A typed Submitted instruction clears only after fresh matching
readback from the captured call, including reversed result/readback order.

## Automated evidence
- Full Go tests and vet pass.
- Full Go race suite passes with the previously approved portable MinGW compiler;
  no system toolchain or PATH change. Focused core/provider race tests also pass.
- Command composition passes for core, no_audio, no_discord, no_nowplaying and
  no_googlemeet build tags.
- Strict TypeScript, ESLint, Prettier and 16 model/attention tests pass.
- All 308 GUI tests pass in 40.8 seconds, including focus, N/A, submission and
  layout/accessibility checks. Meetings was visually inspected at 1280x820 and
  800x600 using isolated fixtures.
- All 58 companion tests pass. Isolated headless Brave loads the actual extension
  and verifies media, ratings, and guarded Meet mic/camera/Leave round trips;
  Leave clicks exactly once and observes departure. These are fixtures, not proof
  of current live-site behavior.
- Native 112/120 px focus, Deafen, Share and Leave previews pass and were inspected.
- All 110 OpenSpec changes validate strictly.
- Companion 1.4.0 Chrome and Firefox packages generated. Firefox real-browser
  acceptance was not performed.

## Build and personal configuration
The canonical scripts/build.ps1 stopped PID 48500 normally and built only
bin/snoofer.exe. Snoofer was restarted as PID 71696 with the exact prior arguments:
--config C:/Users/Eden7600/Documents/sound_snoofer/bin/snoofer.json

The staged configuration passed the built executable's --check before application.
Only the Plus XL Meetings page changed: r1c1–c4 are meetings.mute, meetings.deafen,
meetings.camera and meetings.share; r1c9 is meetings.leave. The previously empty
r2c1–c9 holds a meetings.focus collection with normal overflow. Smaller layouts,
other pages, dials and audio settings were preserved. Post-restart comparison
found only these layout changes and a normal Discord refresh-token update.
Private before/candidate snapshots were deleted after verification.

Unrelated deleted config examples, CLAUDE.md and scripts/check-desktop.cjs remain
outside these commits; check-desktop's SHA-256 remains
6FDF36287D59726824552AC9D0196ABC09B26D108C8212C8D62AE3CC1DDAA7B9.

## Pending
- User reports live Mute/Camera work; that does not establish every multi-call
  propagation case.
- The requested new browser-control approval has not arrived. No user desktop or
  browser control was taken during this change.
- Meet Share's real DOM workflow still requires inspection and implementation.
  It is intentionally unavailable. Do not mark this feature complete.
- The user's loaded companion has not been reloaded to 1.4.0 in this session;
  already-open Meet tabs also require refresh before exposing Leave.
- Real Meet Leave and Share, Discord/Meet focus switching, focused deck actions,
  no-call Deafen, and departure fallback remain live acceptance work.
