# Scene brightness badge
## Why
The user wants to tell bright scenes from moody, dark ones at a glance. A first attempt darkened each wedge by its brightness, but darker colours were hard to read and didn't help. A separate marker reads better.

## What Changes
- **Badge:** scene keys with artwork show a small sun in the top-right of the artwork. Its rays show the scene's overall brightness in five steps:

  | Average brightness | Rays |
  |---|---|
  | Under 15% | None (the disc alone) |
  | 15–35% | 2 |
  | 35–60% | 4 |
  | 60–85% | 6 |
  | 85% and up | 8 |

- **Overall brightness:** the average over the lights the scene sets. Lights it turns off count as 0, and lights it turns on without a brightness count as 100%. Palette-only scenes average their palette entries. A scene with no brightness data shows no badge.
- **Full colour:** wedges are drawn in full colour again; the per-wedge darkening is removed.
- **New field:** controls gain an optional `Brightness` (0–1) that the Hue plugin sets on scene controls, and the deck draws.

## Impact
- **Code:** `snoofer/controls.go`, `plugins/hue` (scene brightness, full-colour artwork), `plugins/streamdeck` (tile), `internal/streamdeck` (badge), `docs/ui-contract.md`.
- **Unchanged:**
  - The GUI's scene cards.
  - Keys without artwork, where the badge would collide with the label at the top.
