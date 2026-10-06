# Soundboard clip timers
## Why
With overlap on, several clips can play at once, and the deck gives no sign of how long each has left. The Soundboard page leaves dials 2–5 unbound, so their touch-strip panels are empty.
## What Changes
- **Timers:** controls may carry running countdowns (`snoofer.Timer`: the control it belongs to and an estimated end). The soundboard publishes one per playing clip on `soundboard.status`.
- **Deck strip:** on a page showing a provider's controls, the panels above unbound dials show that provider's timers as artwork (or the clip symbol) plus remaining time, oldest first. One timer per panel while they fit, two per panel beyond that.
- **Estimate:** a clip's length comes from its normalized WAV; the end is the start time plus that length. A timer disappears when playback ends or stops.
## Impact
- **Code:** `snoofer.Control`, the soundboard voice pool, the Stream Deck plugin's dial frames and the touch-strip renderer.
- **Unchanged:** playback, bindings and the saved layout. Dials stay unbound; turning or pressing them still does nothing.
