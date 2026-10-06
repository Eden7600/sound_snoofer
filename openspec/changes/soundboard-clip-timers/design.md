# Design
## 1. Model
`snoofer.Control.Timers []Timer`, where `Timer{Control string; Ends time.Time}`. `Control` names the control whose label and artwork identify the timer. Timers change only when clips start or stop, so they do not churn revisions. `Ends` is an estimate, not an observation: the voice disappears only when the native player reports completion or a stop.

## 2. Soundboard
Each voice records `ends = start + length`, where length is the normalized WAV's data size over its byte rate (16-bit PCM, header already validated by `validNormalized`). `soundboard.status` carries one timer per voice, in start order. An unreadable length gives no timer rather than a wrong one.

## 3. Deck placement
- A page's providers are the ID prefixes (`soundboard`, `hue`…) of the controls it shows on keys and dials, after region expansion. Stream Deck's own keys don't count.
- Timers from those providers fill the panels of unbound dials, left to right, oldest first.
- With `F` free panels: up to `F` timers get a panel each; beyond that two share a panel; beyond `2F` the newest `2F` show.
- Dial 6 (pages) is never used. Home and Lights show nothing, because no soundboard control is on them.

## 4. Panel
Remaining time is shown as `m:ss`, rounded up, and never below `0:00`.

| Layout | Artwork | Time | Name |
|---|---|---|---|
| One per panel | 64 px on the left | size 4, white | size 1, neutral, beneath the time |
| Two per panel | 40 px per row | size 3 | size 1 beside the time |

- **Artwork:** the clip's current frame. Animated GIFs keep playing. A clip without artwork shows the Play symbol.
- **Refresh:** the strip refreshes at the idle cadence (150 ms), which is enough for a seconds countdown.
