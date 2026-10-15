# Validation

## Shared Meetings presentation, 2026-10-15

- One Call Controls card publishes shared Mute/Camera actions. Discord and Google
  Meet cards display actual per-app state without duplicate competing toggles.
- Thirty focused Meetings/provider-gate GUI tests pass. The full 305-test isolated
  GUI suite passes in 43.4 seconds after correcting an existing notice-copy test
  synchronization race under gui-layout-polish. TypeScript, ESLint and Prettier pass.
- Rendered Meetings pages inspected at 1280×820 and 800×600: labels, state readouts,
  controls and provider cards remain readable with no horizontal overflow.
- TestMeetControlPreview passes for native 112/120-pixel Plus XL/Plus keys; the
  rendered shared mic/camera On, Off, Wait, N/A and Error states were inspected.
  The existing icons, palette and renderer are reused unchanged.

Real-site/account, physical-deck and personal placement checks remain deferred by
the user. Browser fixtures and rendered images do not establish live acceptance.

## Backend and packaging, 2026-10-15

- Shared coordinator and audio wiring: 23 coordinator/Host tests and two audio
  boundary tests cover saved authority, ordered external changes, save failure,
  audio absence, camera periods, retained echoes, bounded source history, exact
  control identity, confirmed-result/snapshot ordering and shutdown.
- Discord and Meet adapter tests cover shared aliases, actual readouts, independent
  mic/camera writes, deafen, call/control replacement, fanout, stale rejection,
  uncertain reconnects and late readback. Discord's focused race cases also pass
  25 repeated runs.
- Full go test ./... -timeout 30s -p 4, go vet ./... and
  go test -race ./... -timeout 90s -p 4 pass. The approved portable
  MinGW compiler was used only in the test process; system PATH is unchanged.
- Optional no_audio, no_discord, no_nowplaying and core command compositions compile.
- Companion: 54 Node tests and isolated headless Chrome/Brave checks pass, including
  Media, YouTube Music ratings and actual DOM-node replacement in Meet. Chrome and
  Firefox archives are rebuilt and manifest-verified at 1.3.1. No install or reload.
- All 109 OpenSpec changes validate strictly. Final frontend type/lint/format checks
  and the shared binding-label regression pass; the 305-test GUI run remains the
  complete presentation gate. Native 112/120-pixel control images were inspected.
- Canonical scripts/build.ps1 passed, including native companion self-tests. Previous
  host PID 41348 exited normally; PID 10360 runs the same bin/snoofer.exe with the
  same absolute bin/snoofer.json argument. Audio and deck settings were preserved;
  the only envelope change was normal Discord refresh-token rotation. The temporary
  private comparison snapshot was removed.

## Deferred live acceptance

Task 7 remains unchecked at the user's explicit Later instruction. Install/reload
companion 1.3.1, real-site ratings with restoration, both-app and multi-call changes,
physical buttons and personal Plus XL placements have not been accepted here.
Like remains planned at Plus XL Home r1c9 and Dislike at r2c9; smaller decks and
shipped layouts remain unchanged. Existing Meetings aliases already target shared
intent, while explicit personal rebinding remains part of that live session.

Discord camera still depends on an actual camera event. Its initial discovery and
cross-channel event attribution require live acceptance; unknown state is not
treated as Off or blindly toggled. Neither browser fixtures nor race tests prove
real-site/hardware compatibility.
