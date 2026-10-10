# Design
`artworkLevel(d)` maps a scene action's brightness b (0–100) to a wedge level:

    level = minArtworkLevel + (1 − minArtworkLevel) × (b / 100)^0.8,  minArtworkLevel = 0.18

| Brightness | Level |
|---|---|
| None given | 1 (full) |
| 100% | 1.00 |
| 50% | about 0.65 |
| 10% | about 0.31 |
| 1% | about 0.21 |

Wedge colours are drawn multiplied by their level. Merging compares hues at full level, so dimming does not shrink colour differences and merge a dim scene's distinct colours. Similar colours merge only when their levels are within 0.15, so one hue at very different brightness stays as separate wedges.
