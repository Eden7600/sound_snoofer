# Tasks
- [x] Trace worker wake-ups, buffering and hybrid crossover.
- [x] Implement event wake-up and timing diagnostics.
- [x] Implement smooth neural-driven upper-band gate and verify DSP.
- [x] Validate reduced buffering with real-model probes and timing evidence.
- [x] Verify GUI, Go/spec checks, build, commit and restore host.
- [ ] Listening and live timing observation confirm improvement on user hardware.

Validation: all 18 original model/rate/block cases and 3 hybrid cases passed at a 32 ms buffer, including 40 pre-roll resets, framing recovery, deadline failure/pass-through and exact original-model fixture parity at the reduced offset. Reported latency at 48 kHz/512 is 61 ms neural and 63 ms hybrid. DSP checks preserve crossover reconstruction, gate opening/closing and near-only 10 kHz when neural voice is present; isolated 10 kHz closes in the actual hybrid callback. Gate does not restore high frequencies already suppressed by AEC3, and quiet speech still needs listening acceptance.

Continuous hybrid test (tools/neural-aec-probe/run.ps1 -Soak): 60 seconds audio, peak worker 5.77 ms, queue 0.37 ms, zero gaps and underruns. This does not reproduce or establish the cause of the user's live voice stutter. Event-driven wake-up removes polling; lifetime Timing in the GUI permits live follow-up. Full Go suite, vet, native Go timing reader, GUI dispatch/render checks and 67 specs passed. Canonical build deployed and no-argument host restored, preferences unchanged. Race detector unavailable with current CGO-disabled toolchain.
