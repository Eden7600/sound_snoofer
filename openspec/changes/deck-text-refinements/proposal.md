# Deck text refinements
## Why
Feedback on the rows layout (`deck-rows-left`) on the device:
- **Cut titles:** song titles and other long names are cut with "…". That wastes space and hides the title.
- **Scene pies:** a scene's artwork band still shows the round edge of its pie.
- **Room names** cannot wrap.
- **Routine status words:** scene and soundboard clip keys repeat a status word that is almost always Ready. Fill colour already shows Playing or Active.

## What Changes
- **Scrolling text:** text that cannot fit, even after wrapping and shrinking, scrolls instead of being cut. This applies to key labels and values, and to the media dial's title and name.
- **Two-line values:** a value of several words that does not fit on one line wraps to two lines and takes the icon's row. Room names are the main case.
- **Pie bands:** artwork with transparent corners, such as scene pies, is cropped to a rectangle inside the circle, so the band shows only wedges.
- **Quieter scenes and clips:**
  - Scene and clip keys omit Ready, Active and Playing.
  - Problem words (Wait, Error, N/A) still show.
  - Without a value, an artwork key's label may use two lines.

## Impact
- **Code:** `internal/streamdeck` (layout, typeface cache, surface loop).
- **UI contract:** scrolling text; scene and clip keys rely on fill for Playing and Active.
- **Unchanged:**
  - The deck is still redrawn only for keys that change. Scrolling redraws only the keys that scroll, at about 12 frames per second, and only while something scrolls.
  - Control semantics, bindings and the GUI.
