# Design
## Measured cause
A ten-second live paired pass-through probe at 48 kHz/512 samples recorded 942 input and 939 output callbacks: three extra synchronized input callbacks at startup, followed by steady pairs. No malformed buffers or stream changes occurred. AEC's missing-output guard incorrectly assumed pairs from the first input.

## Startup
Configure marks AEC inactive and awaiting reference. Input pre-roll validates mic channels and maintains normal capture framing, but leaves microphone output bit-identical until the first valid matching output arrives. The expected output size follows the latest startup input. The first output primes render history and arms cancellation; the next input reports Active. A one-second input sample budget reports missing output if reference never arrives. Once armed, strict pairing and channel/rate/size validation remain unchanged.

Maintaining capture framing is necessary: discarding the first mic buffer shifted frame origins and regressed the existing double-talk residual measurement from 12.4 to 2.4 dB. Do not suppress that failure or lower the test threshold. Startup output must also remain unchanged when read/write pointers alias.

## Lifecycle
The production monitor forwards commands 1 (start), 2 (end) and 3 (change) to input_stage(context, NULL) on the callback thread. Null is an internal lifecycle notification. AEC invalidates its applied generation, processing, partial frames and pending pairing; clears failure and reported metrics; and rebuilds on the next valid input. Output before rebuilding is ignored. Expensive destruction/allocation remains deferred to Configure. This prevents old reference history from crossing stream boundaries. Existing Go recovery remains a compatibility fallback; persistent same-stream faults still latch.

## Validation
The existing callback probe gains an explicit paired pass-through observation mode. It counts ordering and synchronization without saving audio, acquires only a free callback slot and cleans up normally. The existing acoustic probe drives production monitor dispatch plus the real AEC DLL for stream boundaries and failed-stream recovery, verifies startup pre-roll including aliased buffers and missing-reference timeout, and measures stereo/surround/double-talk with unchanged thresholds. Include measured pre-roll in acoustic scenarios and test post-reset convergence using the baseline 20-second adaptation window.

The initial recovery probe measured 13.1 dB at 5–10 seconds after reset; slower reconvergence is a reported limitation, not a reason to lower thresholds. Preserve user settings and unrelated work. Build via scripts/build.ps1, use graceful shutdown and restore the no-argument host. Hardware listening and final live UI confirmation remain separate from synthetic checks.

Startup acoustic acceptance uses 30 seconds: double-talk with measured pre-roll achieved 9.6 dB residual reduction at 15–20 seconds and 14.5 dB at 25–30 seconds. The standard 20-second paired tests are unchanged. The new 30-second case tests eventual convergence, not a guarantee of full cancellation within 20 seconds.
