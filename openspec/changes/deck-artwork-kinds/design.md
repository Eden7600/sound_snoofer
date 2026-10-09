# Design
## Kinds
`artworkKind(icon)` (with the `-muted` and `-off` suffixes trimmed) returns one of:

| Icon | Kind |
|---|---|
| `hue-scene` | pie |
| `soundboard-*` | clip |
| `app-audio` | program icon |
| anything else | cover |

## Keys
- **Cover and pie:** today's band (56 px with a value; grown to the label without one), cropped as today.
- **Clip:** the same band area. The square thumbnail, padding included, is scaled to the band's height, centred horizontally and drawn over the fill, so letterboxing shows the key's colour and nothing is cropped.
- **Program icon:**
  - The thumbnail is scaled with Catmull-Rom to 40 px at (10, 12), above the text.
  - The title is one line at y 60 (15 → 12 px, scrolling when longer), and the value at y 80 (24 → 16 px), as artwork keys showed them before.
  - Placeholder tiles render the same way.

## Resolution
- **Limit:** artwork is a square PNG at most 128 px, base64 at most 128 KiB (`decodeArtwork`, `snoofer.Control.Artwork`).
- **Scenes:** `plugins/hue` renders wedges at 128 px.
- **Covers:** `plugins/nowplaying` thumbnails Windows covers at 128 px with Catmull-Rom, and accepts extension art up to 128 px (128 KiB). The extension draws its canvas at 128 px.

## Media dial
| Artwork | Treatment |
|---|---|
| Program icon | 56 px at (12, 22); text starts at x 80 |
| Cover | 100 × 100, as today |
