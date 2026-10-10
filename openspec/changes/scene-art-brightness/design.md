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

Wedge colours are multiplied by the level before merging, as today. Similar colours at very different brightness therefore stay separate wedges, because the merge distance is measured in RGB.
