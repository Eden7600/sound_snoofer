# Raw microphone activity
## Why
Echo cancellation runs as a Voicemeeter input insert on the microphone in use. It suppresses everything but local speech, so between sentences the processed signal is near digital silence. Activity metering reads Voicemeeter's pre-fader strip level, which follows the insert. A working microphone can then latch Silent while nobody talks, and Auto moves down to a worse, unprocessed microphone. When the user speaks again it moves back, so selection flaps with conversation.

Silence detection is meant to find microphones that deliver no signal at all (a dead battery or an unplugged channel). That has to be judged from the hardware signal before processing.

## What Changes
- **Pre-insert peaks.** While the input insert is registered, the audio monitor records the peak of every physical input channel before any insert stage runs. The worker takes and clears these peaks each sample.
- **Metering source.** Activity metering uses the pre-insert peaks whenever fresh input-insert audio arrived since the last sample. Otherwise it reads Voicemeeter's pre-fader levels as before, which equal the raw signal when no insert runs.
- **Unknown, not Silent.** A sample with no fresh input-insert audio while the insert is registered (a stream restart or a stopped callback) falls back to the pre-fader level. Missing readings stay Unknown.

## Impact
- **Code:**
  - `internal/voicemeeter/callback` (peak capture, `SnooferReadInputs` export);
  - `internal/voicemeeter` (Go binding, `InputLevels` source);
  - `tools/callback-probe` (self-test).
- **Invariants:** unchanged. Metering stays read-only on the worker, and the audio thread does not lock or allocate. Insert stages still write every channel. Selection, wiring and latch timing are unchanged.
