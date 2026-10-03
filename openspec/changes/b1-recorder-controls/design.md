# Design

## Context

See proposal.md. The current API reads B1 but the voice planner owns only B2/B3. Voice transitions currently gate every managed voice cell when any voice operation changes. Saved intent is a strict version-1 sidecar; transport is not modeled. The worker already owns all DLL calls and writer locking, with revisioned typed actions suitable for recording controls.

The user explicitly chose TUI Start/Stop plus source controls and said Post requires Element voice mode. Treat mic recording as a consumer of enabled voice: Pre can use either voice mode, Post only Element. Computer recording remains independent of playback audibility. Defaults for both inclusion toggles are Off, with Pre retained as the initial mic tap.

## Goals / Non-Goals

Goals: a dedicated stereo B1 mix, durable source selections and explicit verified native-recorder commands.

Non-goals: encoding audio in Go, multitrack stems, changing recording directory/format, automatic record on launch/reconnect, pause/resume UI, or processing solely for recording when voice mode is Direct.

## Decisions

### Configuration and compatibility

Add optional studio.recording alongside studio.voice; recording requires voice and Potato. Include computer_sources defaulting to [virtual:1], permitting virtual:1 and virtual:3 only, with duplicate/unknown/AUX rejection. Add optional recording choices to Intent (mic_enabled, mic_tap pre/post, computer_enabled). Existing version-1 sidecars missing these choices normalize to disabled defaults; keep strict rejection of malformed present fields. Save the normalized form on the next explicit edit, using the existing lock/CAS/atomic replacement. Missing profile never claims B1. A sidecar containing recording settings without its profile is incompatible until reset or profile restoration.

Computer-source membership is a configured capture set, independent of studio.playback_sources and its runtime enabled flags. Default primary VAIO captures only programs routed there, not every Windows endpoint.

### Desired B1 matrix and narrow transitions

All Strip[0..7].B1 values default to zero under this profile. Add the effective mic for enabled Pre with voice enabled; add Strip[6] for enabled Post only when voice is enabled, Element is selected and an effective mic exists. Add each configured computer source when computer inclusion is on. Both off produces a silent B1 mix but does not stop an existing recording.

B1 inclusion is maintained while recorder is stopped so its mix can be inspected before Start. Changing choices during recording changes the ongoing mix. Reuse stable source resolution and fallback; never mix dry and processed mic. Missing playback must not block capture. Missing mic may leave eligible computer capture operational.

Refactor transition classification into recording cells and voice/monitor cells. A recording-only edit gates B1 cells only, so a mic recording switch does not interrupt Discord. Source/device/ASIO patch changes gate affected consumers, including B1, before repatching. Retain per-phase expected readbacks and abort-on-drift. Simply extending voiceCell to include B1 would unnecessarily mute Discord for every recording edit and is rejected.

### Recorder API and preparation

Official Remote API documentation page 16 lists recorder.record, recorder.stop, recorder.pause, recorder.play, Recorder.mode.recbus, Recorder.ArmBus(i), Recorder.Channel and Recorder.mode.MultiTrack. Its recorder.B1/B2/B3 values are tape PLAYBACK sends, not record input selection.

Read-only probe against the installed DLL succeeded for Recorder.ArmBus[5] (and its parenthesis spelling), mode.recbus, record/pause/stop/play, Channel, MultiTrack and B1. Observed values: B1 armed; recbus=1; record=0; pause=0; stop=1; play=0; Channel=2; MultiTrack=0; recorder.B1=1. Use bracket spelling consistently. Numeric read support does not prove transport write semantics or successful disk output; controlled hardware acceptance remains mandatory.

Add a separate RecorderSnapshot/transport API so unsupported recorder reads cannot invalidate legacy device routing. Probe the full relevant set on the installed Potato: ArmBus[0..7], transport flags, mode/format-shape parameters and tape playback B1/B2/B3. Preserve parameter errors explicitly; never map a failed read to zero or stopped.

Preparation is requested by Start, not performed blindly by every polling pass. While stopped, verify and set recbus=1, only ArmBus[5]=1 (Potato B1), Channel=2, MultiTrack=0 and tape playback sends B1/B2/B3=0. Preserve A playback sends, file type, bit resolution, sample rate and recording directory. After preparation, verify the entire setup and desired B1 matrix again before issuing record=1. Once prepared, protect tape playback B sends during live recording-profile operation. If another application changes recorder source/mode while active, report conflict and freeze recorder-specific preparation/B1 reconciliation until stopped; unrelated voice/playback rules can continue. Stop remains available from a valid fresh transport snapshot.

B1 master gain/mute/effects stay unchanged. Show a warning for a muted B1 if supported by readback; document that this is a post-fader bus mix, so pre-Element does not mean raw pre-fader capture. Validate a resulting file audibly before calling hardware acceptance complete.

### One-shot transport state machine

Use explicit Start and Stop rows/actions, not a persisted recording boolean and not the generic desired-state retry loop. Commands require live ownership, fresh transport observations and a single in-flight command. Reject commands from stale UI revisions. Start refuses playback, pause, unknown transport, an incompatible active recording, zero eligible sources, or unverified pending routing. An already recording compatible state is an idempotent no-op. Stop is idempotent when stopped; while playing/paused/recording it submits stop=1 and verifies stopped.

After each submitted command, poll within the existing bounded verification timeout. Never automatically resend Record after an API failure or timeout: native REC may have toggle/pause semantics. Show Unknown until refreshed if acknowledgement is uncertain. Clear pending commands on disconnect; never auto-resume after reconnect. External transport changes are observed, not fought.

Quitting or moving to dry-run leaves transport untouched, consistent with existing shutdown semantics. Show that recording continues in Voicemeeter. Format/path setup stays in Voicemeeter; if recording fails because of output configuration, surface observed failure/timeout rather than inventing a filename or claiming a file was saved.

### TUI

Add recording source rows and Start/Stop rows to the Rules view with explicit state: Stopped, Recording, Paused, Playing, Pending, Unknown/Error. Space toggles inclusion, Enter cycles Pre/Post or invokes the selected transport action. Keep inclusion controls visually separate from transport. Show inactive Post reason, effective mic and the configured computer sources. Dry-run can save choices but cannot prepare or operate the recorder. The Start action uses the same latest intent as plan/apply/watch; ordinary apply/watch never starts recording.

## Risks / Trade-offs

- Native REC may toggle pause -> guard by fresh transport state, do not repeat Start while recording, and never retry uncertain commands.
- Directory/format settings are external -> preserve them and require a recorded-file acceptance test, not just parameter readback.
- Starting with tape playback B1 on can capture replay audio -> disable and verify tape playback B1/B2/B3 before recording readiness.
- Live changes can create a short gap in the recording -> disable-before-enable for mic tap/source changes; preserve unrelated voice routes.
- All B1 sends become owned -> opt-in configuration plus visible bus reservation; backup before migration.

## Migration Plan

1. Implement the optional profile and compatible sidecar decoding; retain all legacy no-profile tests.
2. Back up config, sidecar and B1/recorder settings. Add the profile with both source toggles Off; do not auto-start or modify an active native recorder.
3. Confirm Voicemeeter's chosen recording directory/format; run an explicit short Start/Stop test with microphone consent already represented by the user's recording request, and verify a new file exists and plays correctly.
4. Verify mic-only Pre/Post, computer-only and combined capture, mode changes, missing devices and unknown transport outcomes. Leave physical tests unchecked until performed.
5. Rollback: Stop explicitly, stop enforcement, restore settings/config backups. Removing the profile alone does not restore prior bus sends.

Sources: https://download.vb-audio.com/Download_CABLE/VoicemeeterRemoteAPI.pdf (recorder API); https://voicemeeter.com/user-guide-recording-with-the-integrated-recorder/ (bus recording and REC/pause behavior).
