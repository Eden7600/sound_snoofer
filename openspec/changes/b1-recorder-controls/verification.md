# Verification — 2026-10-03

Implemented recording profile, compatible saved choices, all-strip B1 mix,
separate B1 transitions, recorder observation/preparation, guarded single-shot
Start/Stop and persistent TUI controls.

Automated verification: `go test ./... -timeout 30s`, `go vet ./...`, Windows
amd64 build and `openspec validate b1-recorder-controls --strict` passed.
Regression tests cover the full mic/mode/tap/inclusion matrix, fallback and
missing devices, recording-only transition isolation, disable failure, B1 drift,
optional API failure, allowlists, preparation, repeated commands, dry/pending/
empty/paused/playing/unknown guards, timeout and setter failure without retry,
external transport during preparation, worker stale actions and external status.

Interactive dry-run terminal smoke used `work/recording-smoke.json`: navigated
to microphone inclusion, enabled it, selected Post, attempted Start (refused:
live mode required), reloaded, and exited cleanly. The smoke sidecar contains
the chosen mic/Post values. No mixer or transport writes were made. Quit printed
the notice that native transport remains unchanged.

Installed Potato read-only probe successfully read all 19 required parameters.
Recorder was stopped, B1 solely armed, stereo, bus mode, multitrack Off, B1
unmuted. Tape playback B1/B2/B3 were all On; preparation will turn them Off on
explicit Start. `work/recording-baseline-20261003.json` retains those observations
and original strip B1 values. Successful reads are not transport acceptance.

Local config and sidecar were backed up under `work/config.local.before-recording-20261003*`.
Added recording profile with both inclusions Off by default; retained existing
saved desk/Direct/monitor-Off choices. Compared before/after desired operations
excluding B1: identical. Preview is `work/recording-local-preview.json`.

Build: `bin/voice-snooter-recording.exe`.

Hardware acceptance remains pending: no native Start/Stop or file-content test,
no audible Element effect comparison, no physical hotplug or live-recording tap
test. Configure/confirm the native recorder output folder and format, then run
the tests in tasks 5.3–5.5. No recording was started during this build session.
