# Hue scene sort
## Why
Scenes are always listed by name. The user wants to sort them by colour or by brightness, so similar or similarly bright scenes sit together on the deck and in the GUI.

## What Changes
- **Sort control:** a new Sort selection (`hue.scene-sort`) with Name, Hue and Brightness. It is a persisted Hue setting (`scene_sort`; Name by default). On the deck a press cycles the options.
- **Orders:**
  - **Name:** alphabetical, as today.
  - **Hue:** by each scene's main colour around the colour wheel, red through violet. Mostly-white scenes come after the colours, and scenes without colour data last. Ties sort by name.
  - **Brightness:** brightest first, by the overall brightness the scene badge shows. Scenes without brightness data come last. Ties sort by name.
- **Where it applies:** the deck's room-scene keys and the GUI's Lights scene cards, including Other Rooms.
- **Placement:**
  - The default Lights page puts Sort at r1c4 beside Motion; Sync, Mode and Intensity move to r1c5–c7.
  - The user's layout is changed the same way, as requested.
  - The GUI shows Sort as a segmented control under Motion.
- **Icon:** three stacked bars of decreasing length.

## Impact
- **Code:**
  - `plugins/hue` (setting, control, sort keys, scene order);
  - `internal/streamdeck` (icon);
  - `plugins/streamdeck` (default Lights keys);
  - `app/web/app.js` (Sort control, card order);
  - `scripts/check-gui.cjs`;
  - `docs/ui-contract.md` and `docs/plugins.md`.
- **Data:** an optional `scene_sort` in the Hue settings. Existing settings load as Name.
