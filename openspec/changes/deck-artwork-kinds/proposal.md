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
| Program icon | `app-audio` | Icon-sized above the title; the title stays one line (scrolling when long) above the value, as it read before; never cropped |

- **Media dial:** a program icon on the dial (the focused app) is drawn icon-sized beside the text, not as a 100 px cover. Clip timers keep their fitted thumbnails.
- **Detection:** pie cropping follows the content type, no longer a transparent corner pixel.

- **Resolution:** the artwork contract rises from 64 px to 128 px, and encoded size from 32 KB to 128 KB, so scenes and covers stay sharp when cropped and scaled to the key's width.
  - Scenes and covers (Windows media, and the browser extension) are produced at 128 px, and covers are scaled with Catmull-Rom.
  - Soundboard clips and program icons stay at 64 px: they are shown at or below that size, and clip animations would quadruple in memory.

## Impact
- **Code:** `internal/streamdeck` (key layout, artwork band, strip, artwork limit), `snoofer/controls.go` (contract comment), `plugins/hue` and `plugins/nowplaying` (128 px art), `extension` (128 px art), `docs/ui-contract.md`.
- **Unchanged:** animation, the GUI, soundboard and program artwork sizes.
