# Reset AEC at native stream boundaries
## Why
AEC pairs input and output inserts across callbacks, but the monitor currently consumes STARTING, ENDING and CHANGE without notifying AEC. An interrupted pair survives the boundary and can latch "missing output callback". Polling recovery occurs later and cannot prevent processing old reference history.
Live paired observation also found three extra synchronized input callbacks before the first output. Strict startup pairing is the reproduced cause of the persistent live error.

## What Changes
- Tolerate bounded input pre-roll with exact mic pass-through until reference startup, preserving capture framing and strict pairing afterward.
- Forward lifecycle notifications synchronously to the input stage with a null buffer.
- Invalidate AEC framing on the audio thread, clear failure and metrics, and rebuild on the next valid input. Skip output until rebuilt.
- Retain same-stream timing faults and pass-through; do not blindly retry faults or weaken reference validation.
- Exercise the production monitor dispatch together with the actual AEC DLL and rerun surround/double-talk quality checks.
## Impact
Native monitor and AEC, internal hook contract and existing probes. No routing, configuration or DSP tuning changes.
