# Design
Reference: the "R3 · Rows left" and "R2 · Colour system" artboards on the Stream Deck aesthetics canvas. Coordinates are key pixels (112 × 112, before rotation for the + XL).

## Typeface
- **Files:** `internal/streamdeck/fonts/` embeds `Archivo-Bold.ttf` and `ArchivoCondensed-ExtraBold.ttf` from Omnibus-Type/Archivo (`fonts/ttf`), with `OFL.txt`. `third_party/README.md` records the source and licence.
- **Loading:** faces are parsed once. If parsing fails, text falls back to the existing bitmap glyphs, so a bad font never blanks the deck.
- **Antialiasing:** text is drawn at the renderer's 4× supersample and averaged down with the geometry.

## Fitting
- **Label:** 16 px, at most 94 px wide (x 10–104).
  1. One line at 16 px.
  2. Otherwise two lines, word-wrapped, at 16 px.
  3. Otherwise repeat steps 1 and 2 at each size down to 12 px.
  4. Otherwise break inside words at 12 px.
  5. An ellipsis is the last resort.
- **Value:** 30 px condensed, one line, shrinking to 18 px, then an ellipsis.
- **Tests:** the longest built-in short labels fit without an ellipsis.

## Key layout
| Row | Position |
|---|---|
| Label | Top 8, left 10, line height = size + 1 |
| Icon | 30 × 30 at (10, 44). The icon's 80 px drawing area is scaled down |
| Value | Top 76, left 10 |

- **No value:** for go-to keys and blank values, the icon grows to 40 px at (10, 56).
- **No icon:** the value row stays where it is.
- **Artwork:** the thumbnail is scaled to the key's width and centre-cropped into a 112 × 56 band at the top. The label (15 px, one line, fitted) sits at top 60 and the value (24 px) at top 80.
- **Corners:** the keycap masks a rounded corner, assumed 10 px; to confirm on hardware. No text or icon pixel is placed within 8 px of an edge. The error frame is a 4 px stroke with a 10 px outer radius.

## Areas
`keyArea(icon)` maps icon identifiers to an area. The icon is identity, not state, so state precedence (`keyAccent`) stays independent of icons.

| Area | Colour | Icons |
|---|---|---|
| Mic | #38C6E0 | `mic-mute`, `vr-mic`, `mic-stack`, `mode-direct`, `mode-element`, `tap-pre`, `tap-post`, `echo`, call and camera icons |
| Playback | #6E93FF | `speaker-mute`, `vr-playback`, `monitor`, `discord-deafen` |
| Recorder | #A98BFF | `record-*`, `tape-*` |
| Soundboard | #5FD38A | `soundboard-*` |
| Media | #E57CF0 | `media-*`, `app-audio` |
| Lights | #EFE7D3 | `hue-*`, `huesync-*` |
| Other | #C5CBD0 | everything else, including `deck-page`, `deck-up`, `deck-down`, `focus-reset`, `open-controls` |

## Treatments
In `keyAccent` precedence:

| State | Background | Label and icon | Value |
|---|---|---|---|
| Neutral (off) | #17191B | Area colour | #8A9096 for Off, Ready, Paused and Stopped; #F2F2EF for other values, such as selections |
| Active | Area colour | #0B0C0D | #0B0C0D |
| Critical (muted, recording) | #FF5A64 | #0B0C0D | #0B0C0D |
| Error (Error, Failed) | #17191B, with a red rounded frame | Area colour; a red 30 px square with `!` replaces the icon | Red |
| Pending (Wait) | #17191B, with 5 px amber dashes along the top edge | Icon amber | Amber |
| Other attention, fallback | Amber fill, ink #0B0C0D | | |
| Unknown (N/A) | #17191B | Area colour; icon grey | #5E656C |

Value words keep today's mapping (Recording → REC, Pending → WAIT, Unavail → N/A).

## Touch strip
| Panel | Layout |
|---|---|
| Control | Target at (12, 10), 13 px bold, in the area colour or light. Status note right-aligned in amber. Value at (11, 28), condensed, 44 px shrinking to fit 176 px. Position track 4 px at y 80 with the zero mark. Meter: a 10 px full-width bar at the bottom in flat zones (green below −12 dB, amber to −3 dB, red above) over a dimmed track, with the held peak tick. An unknown level shows `N/A` in the meter row. |
| Artwork | 100 × 100 artwork on the left. Target 14 px and name 12 px grey at x 110; value condensed 20 px; a full-width 6 px progress bar at the bottom. |
| Page dial | `← PREVIOUS` at (12, 10), 12 px grey. Current page at (11, 30), condensed 40 px, light. `NEXT →` right-aligned at y 78. |
| Timers | Today's layout, set in the new faces. |
