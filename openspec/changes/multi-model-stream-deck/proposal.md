# Multi-model Stream Deck
## Why
Only the Stream Deck + XL (0FD9:00C6) is recognized; a Stream Deck + plugged in is ignored. Key count, key resolution, dials and the touch strip are fixed to the + XL in the driver, the layout model and the GUI editor, and only one deck can run at a time.

## What Changes
- **Models:** direct HID support for every current JPEG-protocol Stream Deck: Original (v2) and MK.2 (15 keys, 72 px), XL (32 keys, 96 px), Neo (8 keys, 96 px, two touch buttons, info screen), + (8 keys, 120 px, 4 dials, 800 × 100 strip) and + XL (36 keys, 112 px, 6 dials) unchanged. Keys render at each model's native resolution and orientation.
- **Several decks at once:** every connected supported deck gets its own session, frames and input; one failing or unplugged deck does not affect the others.
- **Per-model layouts:** each model has its own layout sized to its grid with a shipped default; decks of one model share it. The existing layout stays the + XL's.
- **Remembered decks:** a deck is remembered by serial the first time it connects (model and an optional name). A remembered deck can have its own layout, be renamed, or be forgotten, and is editable while unplugged.
- **Paging:** each deck keeps its own page (Independent, default) or all decks follow one page (Mirrored), set by a user setting. Decks without dials reserve their bottom-right key as the Pages key (press: next page, hold: Home); the Neo pages with its touch buttons and shows the page on its info screen. The + uses its fourth dial as the page dial, as the + XL uses its sixth.
- **GUI:** the deck editor draws the selected model's grid and dials and offers model layouts and remembered decks, deck name, own layout, Forget and Paging.

## Impact
- **Code:** `internal/streamdeck` (model table, decoding, image reports, rendering, multi-session surface), `plugins/streamdeck` (geometry-sized layouts, defaults per model, per-deck state, settings, editor, connection reports), `app/web` deck editor.
- **Settings:** additive. `layout` remains the + XL layout; new `models`, `devices` and `paging`; `serials` overrides keep working and gain a model through their device record (legacy overrides without one are + XL).
- **Unchanged:** + XL rendering bytes, layout, dial roles and behavior; touch gestures stay unbound; dark/inert presence handling; one brightness for all decks; control dispatch semantics.
- **Out of scope:** legacy BMP-protocol models (Original v1, Mini), the Pedal, Stream Deck Modules and Studio.
- **Hardware:** only the + and + XL are available for acceptance; Original/MK.2, XL and Neo support follows published protocol details and stays unverified until tested on hardware.
