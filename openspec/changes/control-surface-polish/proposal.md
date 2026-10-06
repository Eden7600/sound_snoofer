# Control surface polish
## Why
The knobs and meters look flat and update slowly. Mic controls clutter the deck when the mic stack is off. The soundboard cannot layer sounds. The GUI chrome shows a pointless "Connected / Plugin host" badge, and a "No stall detected" badge on every page, even though Snoofer now covers more than audio.
## What Changes
- More engaging, more responsive meters and knobs:
  - **Stream Deck dials:** smooth gradient meter with VU ballistics and peak hold, plus a knob-position track.
  - **GUI mixer strips:** a matching meter and position track.
  - **Timing:** faster level sampling and deck refresh while meters are shown.
- When the mic stack is off, the Stream Deck blanks mic-related keys and dials; the GUI keeps showing them. `Control.Hidden` becomes a surface-only signal that no longer implies unavailable.
- A soundboard Overlap toggle (GUI and a bindable deck key): up to 8 simultaneous voices, oldest cut, Stop silences all.
- The sidebar "Connected / Plugin host" footer is removed. Audio health leaves the shared header and appears only on the Audio and Diagnostics screens.
## Impact
Stream Deck rendering and plugin, audio controls, soundboard plugin settings (`overlap`, optional), GUI and UI contract. The native companions are unchanged: overlap uses several player instances. No routing or recorder changes.
