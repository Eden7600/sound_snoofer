# Deck regions and Lights page
## Why
The Hue block takes 16 of the Home page's 36 keys, mostly room-scene slots that are rarely used during a session. Automatic layout is all-or-nothing: a page can name one ID prefix that fills every free key, so a page cannot mix fixed controls with a bounded block of clips or scenes, and the editor asks for a raw prefix such as `soundboard.clip-`. Motion sensors in the selected room cannot be paused from the deck, so lights flip on during films or streams. A UX review of the deck surface (design.md §5) found further friction: deep page cycling, inconsistent key numbering in the editor and an empty default layout for new users.
## What Changes
- **Hue motion toggle:** a `hue.motion` key for the selected room's motion sensors (On / Off / Mixed). It is blank when the room has none.
- **Lights page:**
  - Hue controls move to their own page.
  - Home keeps Brightness (press toggles the room), the Brightness dial, Sync, Mode, Intensity and the new Motion key.
  - The Lights page adds the Room key and a region of the selected room's scenes.
- **Room scope:** Snoofer controls only the rooms chosen on the Lights screen, in the GUI and on the deck. The Room key shows room names.
- **Page regions:**
  - **Placement:** a page holds any number of rectangular regions, each filled from a named control collection (Soundboard clips, Room scenes, Scenes in a given room).
  - **Precedence:** manual and shared bindings win.
  - **Overflow:** it adds pages that repeat the fixed controls.
  - **Migration:** the legacy `auto_controls` prefix keeps working as a whole-page region.
- **Control collections:** controls carry an optional `Collection` and `CollectionLabel`, so the editor can offer named sources instead of prefixes.
- **Go-to page keys:** bindable `streamdeck.goto-<page>` keys on Home jump straight to a content page. There is no go-to key for Home; the page dial's press returns there.
- **Editor:**
  - **Selection:** Shift+click or Shift+arrows select a rectangle of keys.
  - **Regions list:** the page panel lists regions, each with a source picker and Remove.
  - **Grid:** region cells are tinted, and all key numbers are one-based.
- **Defaults:** new layouts include Soundboard and Lights pages built from regions.
## Impact
- **Code:** the Hue plugin (adds the `motion` resource reads and writes), the soundboard (collection metadata), the Stream Deck layout model and editor, the GUI deck screen and the Lights screen.
- **Data:** `snoofer.Control` gains two optional presentation fields. Saved layouts gain an optional `regions` list, and existing layouts load unchanged.
- **Personal layout:** the user's own layout is migrated as requested (§4).
- **Unchanged:** audio routing and recorder behavior.
