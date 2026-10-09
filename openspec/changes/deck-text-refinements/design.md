# Design
## Typeface cache
`typeset` keeps one face per (typeface, size) behind a mutex. Faces are not safe for concurrent use, so every measure and draw holds the lock. Measuring on every scroll frame is then cheap.

## Fitting order
1. **Labels:** wrap and shrink as today (16 → 12 px, two lines), without breaking words. If no fit preserves words, the label is one line at 15 px and scrolls.
2. **Artwork key labels:** one line, or two when the key shows no value, at 15 → 12 px. Otherwise one line, scrolling.
3. **Values:**
   1. One line at 30 → 22 px.
   2. Otherwise, for several words, two lines at 22 → 16 px. The two lines start at y 48, and the icon is omitted.
   3. Otherwise one line at 20 px, scrolling.
4. **Strip:** the media dial's target and name scroll when they do not fit at their minimum size. Other strip text keeps its ellipsis.

## Scrolling
- **Motion:** a scrolling line holds for 1.2 s, moves left at 30 px/s until its end shows, holds for 1.2 s, then returns to the start.
- **Clip box:** the line is clipped to its box (x 10–104 on keys), so it never enters the corner inset.
- **Phase:** `renderChangedFrame` takes a phase (the time since the device session started) and reports whether anything scrolls.
- **Caching:** only scrolling keys carry the phase in their presentation, so other keys keep their cached images.
- **Timer:** while anything scrolls, the surface session redraws every 80 ms. The timer stops when nothing scrolls.

## Pie bands
When the artwork's corner pixel is transparent, `drawBand` crops the source to the 2:1 rectangle inscribed in the circle. The circle's radius is half the shorter side, less one pixel. Its height is 2r/√5.

## Routine words
| Icons | Words omitted |
|---|---|
| `hue-scene`, `soundboard-play` | READY, ACTIVE, PLAYING |
