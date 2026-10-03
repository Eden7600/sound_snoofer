# Selectable voice routing validation — 2026-10-03

## Automated verification

- `go test ./... -timeout 30s`: passed, including legacy fixed-device/studio tests.
- `go vet ./...`: passed.
- Windows build: `bin/voice-snooter-next.exe`.
- `openspec validate selectable-voice-routing --strict`: passed.

Requirement coverage:

| Requirement | Evidence |
| --- | --- |
| Profile ownership, edition and config defaults | config/voice_test.go; routing/voice_test.go |
| Preferred/effective mic, ambiguity and reconnect | routing/voice_test.go |
| Exclusive voice, protected buses and monitoring | routing/voice_test.go matrix and migration tests |
| Optional playback and no playback destination | routing/voice_test.go |
| Ordered transitions, partial failure, restart, debounce and drift | controller/voice_test.go; existing topology/controller/ownership tests |
| API bounds and invalid snapshots | voicemeeter/voice_test.go; existing client tests |
| Shared saved intent and no-write commands | cli/voice_test.go; config/voice_test.go |
| TUI edits, stale commands, invalid reload/reset and save failures | tui/rules_test.go; existing TUI tests |
| Host health limitation | TUI messaging and README; end-to-end hardware tests still outstanding |

An interactive Windows terminal smoke test ran the new binary in dry-run using work/voice-smoke.json. Enter changed desk to lav, Space disabled voice delivery, the sidecar recorded those choices, Tab opened Routing, reload retained selections, and q exited successfully and restored the terminal. No mixer writes were requested. The first run revealed clipped footer hints at the terminal width; the footer was shortened and secondary controls moved into the Rules view. Keyboard tests also caught and fixed Bubble Tea's named Space key.

## Local migration

The previous config was backed up to work/config.local.before-voice-rules-20261003.json. The read-only mixer baseline is work/voice-baseline-20261003.json. The local config now explicitly enables the voice profile with desk/Element/monitor Off defaults. Its read-only plan is work/voice-local-preview-20261003.json and resolves desk -> B2 -> AUX -> B3 with playback A2 and Volt A1. At inspection the only differing desired assignment was webcam on input 3; this was not applied.

The earlier bin/voice-snooter.exe remains running. The updated binary is deliberately side-by-side as bin/voice-snooter-next.exe. Close the old instance before starting the new one in live mode. Do not reload the new config in the old binary: it does not understand studio.voice.

## Outstanding hardware acceptance

OpenSpec tasks 6.2–6.6 remain unchecked. The earlier user-confirmed Discord test establishes the manually configured pass-through baseline only. New-code Direct/Element audio, a real plugin effect, the lav with a charged battery, pre/post headphone monitoring, physical device disconnect/reconnect, and Element stop/recovery still require real-device verification. No claim is made that API tests prove those audio behaviors. Default execution stays dry-run; live operation is explicit.
