# Sound Snoofer implementation and verification

The software work in `harden-application-consistency` is implemented in `C:\Users\Eden7600\Documents\sound_snoofer`. Four physical acceptance items remain unperformed. Automated checks and preview observations do not establish audible correctness.

Work is based on commit `22d71ae` and the updated AGENTS.md. The pre-existing AGENTS.md edit and personal configurations were preserved. Changes are uncommitted; no running live build was replaced. No runtime dependencies were added.

## Implemented work

| Area | Result | Main regression evidence |
| --- | --- | --- |
| Persistence and selection | CLI mute journals use the config path. VR-only playback choices validate, save, reload, and route without duplicate matchers. Misplaced-journal recovery is documented. | CLI/config consistency tests; TUI choice-to-readback integration |
| Mic Off and mute | Configured VR inputs join the managed microphone set. Off clears their B1 sends despite recorder conflicts. TUI, deck, and controller share mute targets and relevant route dependencies. | Inputs 4/5 across stopped, recording, paused, playing, unknown, and conflicting states; existing baseline-mute and output-migration regressions |
| Worker lifetime | Profile removal publishes disabled default policy. Revocation supersedes queued policy and waits for acknowledgment before preview or writer release; timeout keeps ownership and reports uncertainty. | Default lifecycle tests; fake worker profile-removal and delayed-revocation tests |
| Disruptive commands | Stale restart confirmations are rejected. Queued transport retains its original command/revision. Disconnect drops undispatched actions; the reader and renderer finish before the FIFO disconnect marker. Focus survives state coalescing. | Restart, queue, concurrent disconnect-marker, and desktop focus regressions |
| State and usability | Successful observation time is separate from worker progress. Repeated errors remain visible. Current health is independent of the previous restart result. Compact details, contextual reasons, configured headset labels, reset/discard confirmation, and minimum-size edit guards are implemented. | Observation/error tests; layout, Unicode, sanitization, NO_COLOR, keyboard, resize, and discard-ack tests |
| Simplicity | One action numbering, shared atomic replacement, one fresh process enumeration, bounded absent-deck discovery, checked CLI output, and removal of newly orphaned/unreachable code. | Existing persistence/ownership checks plus new process/discovery/output regressions |
| Documentation | Short README, application guide, journal troubleshooting, reproducible check script, explicit historical-spec supersession, and checked task evidence. | Strict validation of all 30 OpenSpec changes |

Automatic stall recovery remains visibly unavailable. The separate proposed recording-source/playback-target redesign was not folded into this change.

## Executed validation

On 2026-10-05:

- Required gofmt/import grouping completed on touched Go sources; `git diff --check` passed.
- `scripts/check.ps1` passed the full 13-package suite, `go vet ./...`, the Windows GUI-subsystem build, and all 30 strict OpenSpec validations.
- Bounded stdlib fuzzing passed: the latest config run executed 205,668 cases and HID decoding executed 252,713 cases. No failures were found. The final disconnect-ordering fix was subsequently tested and the full release checks rerun.
- Stale restart/preview revocation checks passed five consecutive runs; transport/disconnect-marker regressions passed ten.
- The repository now contains 190 Test/Fuzz declarations, versus 165 at audit time. This is a source count, not a claim that opt-in hardware cases ran.
- Race testing was skipped because CGO is disabled. No toolchain was installed or changed.

Reproduce from the repository root:

```powershell
.\scripts\check.ps1 -Fuzz -Output bin/sound-snoofer-candidate.exe
```

The checked executable is delivered alongside this report as `sound-snoofer-checked.exe`. It uses the Windows GUI subsystem. Specify the intended config explicitly when trying a build from a different directory: default config paths are executable-relative. Do not infer that the delivered executable automatically uses your existing personal profile.

## Measurements

These are software measurements with fake audio observations and real worker/persistence code, not live audio latency guarantees.

| Measurement | Observed result |
| --- | --- |
| Monitor preference input to acknowledgment | 1.5754 ms |
| Monitor acknowledgment to verified fake readback | 20.7658 ms |
| VR playback selection input to acknowledgment | 1.6054 ms |
| Playback acknowledgment to verified fake readback | 1.000084 s |
| 1,002 fake host observations | 14.1464 ms; 1,002 process enumerations |
| Previous host-enumeration structure | 2,004 enumerations for that many observations, inferred from the two independent probe calls |
| Absent-deck discovery with 100 rapid state updates | Two calls, 2.0002104 seconds apart |

The latency test saves choices to a temporary profile, checks acknowledgment separately from routing settlement, then reloads the VR playback preference. Existing debounce, delayed-readback, call-count, transaction-freshness, and no-retry checks remain intact.

## Native preview evidence

Read-only Windows endpoint discovery passed and resolved the default Voicemeeter playback/capture targets. Read-only HID discovery passed and returned zero Stream Deck + XL interfaces. Detection does not verify actual controls or audio.

The isolated dry-run tray test passed on the interactive desktop and exited through context cancellation after three seconds. Its first sandboxed attempt could not initialize the Windows tray; rerunning outside that desktop restriction passed.

A console-subsystem smoke build of the same TUI was run with explicit `--dry-run` and a copied `config.voice.json` under the task's scratch directory. It connected to the installed Voicemeeter, showed recorder Stopped, opened scrollable help/details with the isolated config path and observation age, opened reset confirmation, cancelled it, switched to the observed Graph, and quit successfully with terminal restoration. No live toggle, recorder command, or mixer setter was requested. Native desktop screenshot/accessibility inspection of the terminal view was not completed; the terminal output and keyboard flows were inspected directly.

## Remaining physical acceptance

Keep tasks 8.1 through 8.4 unchecked:

1. Listen through Volt/webcam/headset switching, Off, native mute, Element tails and fallback. Confirm computer playback and capture continue.
2. Create and listen to native recording files for intended Pre/Post/computer content. Check active-transport guards and restart/reconnect without replay.
3. Attach an actual supported deck and test keys/knobs, backlog/disconnect, controls closed, Windows-default contention/disable, and SteamVR transitions.
4. Check actual tray Quit, Explorer restart, sleep/resume, and hidden hotplug in a planned live session. Screen-reader narration also remains unverified.

These require the user's hardware and listening participation. The read-only probe found no supported deck, and there was no live listening session during this implementation. The existing hardware checklist and historical evidence remain intact.

Executable SHA256: `fab4c9b9005c299fb0ca382804b368b6d8485af03a2a3471fd4be62011d24002`.
