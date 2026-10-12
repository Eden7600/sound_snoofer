# Soundboard hold to loop
## Why
A clip can only play once. Holding a clip key should keep it going until Stop.
## What Changes
- **Hold:** clip controls offer `hold`. On the Stream Deck a key held for 500 ms sends hold (the existing deck hold); a quicker tap plays on release. In the GUI, a clip button held for 500 ms with the mouse or touch sends hold; a click or Enter still plays once.
- **Loop:** a held clip plays as a looping voice. When it reaches the end it starts again from the beginning, until Stop.
- **State:** a looping clip shows Loop (active colour) instead of Playing, on deck keys and in the GUI.
## Impact
- **Code:** soundboard voice pool and worker, deck key accent, GUI clip buttons and tone.
- **Unchanged:** Overlap, replacement and eviction rules, normalization, routing and readiness, timers, Stop, saved settings and bindings. Loops are not persisted or resumed.
- **Changed:** deck clip taps act on key release instead of key down, because the deck only knows a tap from a hold when the key comes up.
