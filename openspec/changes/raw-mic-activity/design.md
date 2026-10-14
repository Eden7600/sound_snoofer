# Design
## Capture (audio thread)
`pass_through.h` keeps `input_peaks[16]` (physical channels: Potato's 5 strips × 2, rounded up) and an `input_inserts` counter, all `volatile LONG`.

In the input insert (command 10), after the buffer passes `valid`, and **before** the insert stage or pass-through runs:
- For each channel below `min(inputs, 16)`, compute the peak `|x|` over the buffer. NaN samples are skipped.
- Raise the stored peak with a compare-exchange loop on the float's bits. Non-negative IEEE floats order like their bit patterns, so the integer comparison is the float comparison.
- Increment `input_inserts`.

No locks, no allocation; the work is bounded by `samples × 16`.

## Read (worker)
`SnooferReadInputs(float *peaks, long count)` exchanges each of the first `count` (≤16) peaks with zero, writes them as floats, and returns the exchanged `input_inserts`. The worker is the only reader. A buffer that straddles a read contributes to either window, which is harmless for a 100 ms activity sample.

`winAPI.InputPeaks(channels int) ([]float32, bool)` returns `false` when the monitor is not active, no insert hook is registered, or no input insert ran since the last read.

## Metering source
`Client.InputLevels(strips)` keeps its contract: linear peaks by physical strip, with missing entries unknown.
- If the API provides `InputPeaks` and it reports fresh audio, each strip's level is `max(peaks[2s], peaks[2s+1])`.
- Otherwise the existing `GetLevel(0, …)` reads apply.

The input insert carries every input channel regardless of which strip the echo canceller processes, so checked microphones that are not in use read raw too.

## Considered
- **Exempt the processed microphone from Silent.** Simpler, but a lav whose battery dies while in use would never be detected, which is the case activity metering exists for.
- **Engine-reported peaks.** The AEC3 and neural engines would each need an export, and metering would depend on which engine runs. The monitor already sees every input-insert buffer first.
