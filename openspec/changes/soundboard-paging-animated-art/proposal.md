# Soundboard paging and animated artwork
## Why
Soundboard overflow pages are reachable only by turning the page dial, which also steps through every other page. A soundboard with more clips than keys needs direct paging on the page itself. Clip artwork from GIF files shows only the first frame, so animated artwork looks broken.
## What Changes
- **Scroll keys:** bindable `streamdeck.scroll-up` and `streamdeck.scroll-down` keys move through a page's overflow sets (`Soundboard`, `Soundboard 2`, …) and wrap. They show the set position (`1/2`) and are blank when the page has a single set.
- **Page dial:** a page that binds a scroll key counts as one stop on the page dial, which lands on its first set.
- **Default and personal layout:** Soundboard binds Up below Overlap (r2c9) and Down above Stop (r3c9).
- **Animated artwork:** GIF clip artwork plays on deck keys and on the Soundboard screen. Controls gain optional `Animation` frames; `Artwork` remains the static first frame.
## Impact
- **Code:** Stream Deck layout navigation, the plugin loop and icons; `snoofer.Control`; soundboard artwork loading (now cached by file identity); the desktop bridge and Soundboard screen.
- **Data:** no saved-format change. Animation frames are not part of the polled GUI state; the GUI fetches them once per artwork.
- **Unchanged:** audio routing, clip playback and recorder behavior.
