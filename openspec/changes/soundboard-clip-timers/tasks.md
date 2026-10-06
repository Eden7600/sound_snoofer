# Tasks
Each numbered block is one reviewable commit.

## Implementation
- [x] 1. `docs(deck)`: specify clip timers on the touch strip; update the UI contract.
- [ ] 2. `feat(soundboard)`: publish clip timers. Covers `snoofer.Timer`, WAV length, voice end times and `soundboard.status` timers. Tests cover the length calculation and timers following voices.
- [ ] 3. `feat(streamdeck)`: show timers above unbound dials. Covers provider matching, panel distribution, the `:` glyph and panel rendering. Tests cover placement and capacity, plus a native-size preview.
- [ ] 4. Validate: Go tests and vet, GUI and desktop checks, OpenSpec, the canonical build; relaunch.

## Hardware acceptance
- [ ] 5. On the Stream Deck + XL: single and overlapping clips count down above dials 2–5 on the Soundboard page, timers clear on end and Stop, and Home and Lights are unchanged.
