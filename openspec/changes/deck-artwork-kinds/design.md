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
  - The key uses the plain row layout: label, icon slot, value (volume, Muted, Mixed).
  - The thumbnail is scaled with Catmull-Rom into the 34 px icon slot at (10, 42), or into 40 px at (10, 56) when there is no value.
  - Placeholder tiles render the same way.

## Media dial
| Artwork | Treatment |
|---|---|
| Program icon | 56 px at (12, 22); text starts at x 80 |
| Cover | 100 × 100, as today |
