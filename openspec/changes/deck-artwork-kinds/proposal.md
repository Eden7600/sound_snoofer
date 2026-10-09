# Artwork by content type
## Why
Every key with artwork is drawn like cover art: the image fills a band at the top, cropped to the band's shape. That suits album covers and scene pies, which the user is happy with. It fails for the other two kinds:
- **Soundboard clips.** Clip art is the user's own image, fitted into a square with transparent padding, and the crop cuts it apart.
- **Program icons.** Icons and the name-coloured placeholder tiles have transparent corners. That makes them look like pies, so the renderer zooms into their centre and blows a small logo up into a blurry band.

## What Changes
The key's icon identifier names its artwork's content type, and each type gets its own treatment:

| Kind | Icon | Treatment |
|---|---|---|
| Cover | `media-*` and any other | Band crop, as today |
| Pie | `hue-scene` | Rectangle inscribed in the circle, as today |
| Clip | `soundboard-*` | The whole image, fitted inside the artwork area without cropping and centred; the key's fill shows around it. Animation is unchanged |
| Program icon | `app-audio` | Drawn as the key's icon in the normal row layout (label, icon, value), smoothly scaled, never cropped |

- **Media dial:** a program icon on the dial (the focused app) is drawn icon-sized beside the text, not as a 100 px cover. Clip timers keep their fitted thumbnails.
- **Detection:** pie cropping follows the content type, no longer a transparent corner pixel.

## Impact
- **Code:** `internal/streamdeck` (key layout, artwork band, strip), `docs/ui-contract.md`.
- **Unchanged:** providers, the artwork contract (64 px PNG), animation, the GUI.
