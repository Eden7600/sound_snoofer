# Design
## Inactive
- **Field:** `snoofer.Control.Inactive`: shown greyed and inert on control surfaces, keeping its binding's position. It differs from:
  - `Hidden`, which is blank, inert, and yields its key to covering regions;
  - `Available = false`, which shows N/A.
- **Mic stack:** `plugins/audio` sets `Inactive` for `micPathControls` while the mic stack is off, in place of `Hidden`.
- **Tiles:** `plugins/streamdeck` maps an inactive control to `device.Tile{Label, Icon, Inactive: true}`, with no value, meter or artwork.
- **Input:** presses, holds and dial turns on inactive bindings are ignored, as for hidden ones.
- **Regions:** candidate filtering is unchanged. Inactive members still occupy their cells.

## Rendering
| Surface | Inactive treatment |
|---|---|
| Key | Dark key; label and icon in `unknownColor` (#5E656C); no value, so the icon takes the value row's space (40 px) as other value-less keys do |
| Dial | Target and value in `unknownColor`; no position track or meter |

## Artwork band
When an artwork key has no value, the label is fitted first: up to two lines at 15 → 12 px, or one scrolling line. It is bottom-aligned so its last line ends at y 104. The band fills from y 0 to 4 px above the label.

Cropping to the band's aspect a = 112 / height:

| Artwork | Crop |
|---|---|
| Square | The centred rows of height `side / a` |
| Pie (transparent corners) | The rectangle inscribed in the circle with that aspect: height `2r / √(1 + a²)`, width `a × height` |

Keys with a value keep today's 56 px band.
