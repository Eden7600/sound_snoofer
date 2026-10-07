# Recover transient neural callback gaps

## Why
The user reports LocalVQE stopped midstream with missing output callback. The host is still running, with localvqe-voice selected. Source inspection identifies an unconditional permanent failure on the first repeated input after reference startup. The exact upstream cause of the missing callback has not been captured. A single discontinuity must not permanently disable otherwise working cancellation.

## What changes
Allow the neural bridge to discard mismatched audio, invalidate queued results and re-prime after an isolated missing or extra output. Require one second of continuous valid pairs to finish recovery; latch if instability persists for two seconds of input audio. Retain the existing one-second no-reference timeout and all compute/deadline/invalid-audio failures. No changes to model, quality, rate, routing or GUI controls.

## Impact
Neural native bridge and real-model regression probe. Replaces the strict single-gap latch in neural-aec for recoverable callback discontinuities only.
