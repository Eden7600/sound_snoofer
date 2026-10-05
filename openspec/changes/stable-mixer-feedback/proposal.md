## Why
Immediate readback after a gain or mute setter may still contain the previous value. Gain reports a false failure, while controller errors are incorrectly published as disconnections and flash every deck key unavailable.
## What changes
Verify gain with cancellable, bounded parameter polling. Treat mute readback lag as pending. Separate observation availability from routing/write diagnostics. Keep observed dB visible alongside scoped gain feedback.
## Impact
Shared control actor, mixer controller, Stream Deck renderer, deterministic regression tests. No routing policy or device selection changes.
