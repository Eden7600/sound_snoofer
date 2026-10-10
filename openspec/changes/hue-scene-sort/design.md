# Design
## Setting and control
- **Setting:** `Settings.SceneSort` (`scene_sort`) is `name`, `hue` or `brightness`. Empty or unknown means name, and validation rejects other values.
- **Control:** `hue.scene-sort` is a selection with options `name`, `hue` and `brightness` (labels Name, Hue, Brightness), short label Sort and icon `hue-sort`. Setting it saves before it takes effect, like the room choice. The deck cycles the options on a press because the control has no press operation.

## Sort keys
For every scene, from its resources:
- **Hue key:** the hue angle (0–360°) of the scene's largest merged wedge (`mergeSwatches(sceneSwatches(scene))[0]`).
  - A wedge whose HSV saturation is below 0.25 counts as white and sorts after every colour.
  - A scene with no wedges sorts last.
- **Brightness key:** `sceneBrightness`, descending. An unknown brightness sorts last.
- **Name:** the scene's short label, case-insensitive, then its ID. It breaks every tie.

`model.sortScenes(scenes, mode)` orders a list. `roomScenes` applies the setting, so the room-scene slots follow it.

## Order on controls
- **Scene controls:** each `hue.scene-…` control's `Order` is its position within its room in the chosen sort, and each room-scene slot's `Order` is its slot number. Region candidates already sort by `Order`, then label.
- **GUI:** the scene cards, both the room's own and each room under Other Rooms, sort by `Order`, then label. The Lights screen rebuilds when the sort changes.

## Layout
- **Default Lights keys:** `hue.group`, `hue.brightness`, `hue.motion`, `hue.scene-sort`, `hue.sync`, `hue.sync-mode`, `hue.sync-intensity` at r1c1–c7.
- **Personal layout:** the user's Lights page gets the same row while Snoofer is stopped, with a backup.

## Icon
`hue-sort`: three horizontal bars, long, medium and short, top to bottom.
