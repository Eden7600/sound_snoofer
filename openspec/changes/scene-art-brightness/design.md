# Design
## Brightness
`sceneBrightness(scene)` in `plugins/hue` returns the scene's overall brightness, 0–1, and whether it is known.

| Source | Contribution |
|---|---|
| Action turning a light off | 0 |
| Action turning a light on, with dimming | brightness / 100 |
| Action turning a light on, without dimming | 1 |
| Palette entries (when there are no actions) | their dimming, or 1 |

- **Average:** the result is the average of the contributions.
- **Unknown:** with neither actions nor palette, the brightness is unknown.

Scene controls (`hue.scene-…` and the room-scene slots) set `Brightness` when it is known.

## Artwork
`sceneSwatches` draws every wedge at full level again, and `artworkLevel` is removed. Merging compares colours as it did before the darkening, without the level term.

## Control and tile
- **Control:** `snoofer.Control.Brightness *float64`, an optional overall brightness that surfaces may indicate.
- **Tile:** `device.Tile` and `keyPresentation` carry it as `Brightness float64` with `BrightnessKnown bool`.
- **Caching:** it is part of the key presentation, so cached key images update when it changes.

## Badge
- **When:** drawn on artwork keys only, after the band.
- **Backing:** a 15 px disc in the key colour, centred at (89, 23) so its edge meets the corner inset, for contrast on any artwork.
- **Sun:** in the light text colour, a 5 px core disc.
- **Rays:** from radius 8 to 12.5, about 3 px thick. The pairs appear in order: vertical, horizontal, then the two diagonals. A step of n shows 2n rays.
