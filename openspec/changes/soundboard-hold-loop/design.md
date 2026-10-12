# Design
## 1. Requests
Clip controls list `press` and `hold`. `hold` follows the same path as `press` (readiness, unchanged-source check, preparation with Wait, replacement or overlap) with a loop flag carried by the queued command and the pending preparation. Stop, Overlap and Volume are unchanged and do not offer hold.

## 2. Voices
A voice records its clip length (from the normalized WAV; zero when unknown) and whether it loops. Its timer end is start plus length. When polling finds a looping voice complete without error, the pool plays the same normalized file again on the same player and resets the timer end. A restart failure drops the voice and is reported like any native failure; it is never retried.

Restarts happen on the 200 ms poll and rebuild the DirectShow graph, so loops are not gapless; a short gap between iterations is expected. No native ABI change.

## 3. Ending a loop
A looping voice ends exactly as a playing one does: Stop; a new clip with Overlap off; eviction as the oldest of eight with Overlap on; unready or disconnected routing; a native error; preview; plugin stop or quit. The poll checks routing readiness before polling voices, so an unready mixer stops every voice instead of restarting a loop. Loops are not saved and never resume after restart or reconnect.

## 4. Presentation
- The clip's Value is `Loop` while any voice of that clip loops, otherwise `Playing` or `Ready` as before.
- Deck: `LOOP` uses the active accent and is shown on the key (unlike Playing, which the fill already says).
- GUI: Loop is active-toned and the clip button uses the playing state. A pointer held for 500 ms on a clip button sends hold and suppresses the following click; releasing, leaving or cancelling earlier sends nothing extra and the click plays once. The touch context menu is suppressed on clip buttons.
