# Design
## 1. Scroll keys
Overflow already expands a page into sets `ID`, `ID~auto~2`, … that repeat the fixed bindings. The Stream Deck plugin publishes two local controls beside the go-to keys:

| ID | Label | ShortLabel | Icon | Value |
|---|---|---|---|---|
| `streamdeck.scroll-up` | Scroll up | Up | `deck-up` | `n/N` |
| `streamdeck.scroll-down` | Scroll down | Down | `deck-down` | `n/N` |

- They describe the page being drawn (like go-to keys), so `n/N` is the shown set's position among its page's sets.
- With one set they are Hidden: a blank, inert key that keeps its binding.
- Presses wrap (Down on the last set shows the first). Deck presses are handled locally; GUI presses switch the device the same way.
- They are the only Stream Deck controls, with go-to keys, offered as bindings. Validation treats them as ordinary press controls.
- Wrapping avoids a dead key at either end; with the usual two or three sets both keys stay useful.

## 2. Page dial
`next` visits each page once when that page's effective keys bind a scroll key: its overflow sets are skipped and a turn from any set moves to the neighbouring page's first set. Pages without scroll keys keep stepping through every set, so overflow is never unreachable. Page names on the dial follow the same order.

## 3. Defaults
The default Soundboard page binds Overlap (key 9), Up (18), Down (27) and Stop (36) down column 9; clips fill the rest of r1–r4. The personal layout gets the same two keys.

## 4. Animated artwork
- **Model:** `snoofer.Control.Animation []ArtworkFrame{Artwork, Delay}`. Each frame follows the `Artwork` thumbnail contract. `Artwork` stays the first frame, so surfaces without animation, and the editor preview, are unchanged. `Animation` is excluded from control JSON, because the GUI polls the whole state every 100 ms.
- **Decoding:** GIF frames are composited onto the logical canvas honouring disposal (none, background, previous), then scaled into 64px thumbnails like static artwork. Delays of 10 ms or less become 100 ms, as browsers do. A GIF with more than 120 frames stays static with an artwork note. Single-frame GIFs have no animation.
- **Cache:** the catalogue reuses artwork while the image file's size and modification time are unchanged, so the 5 s rescan no longer re-decodes every image.
- **Deck:** a key shows the frame for the current time (one shared clock, loop = sum of delays). While a shown key is animated the plugin refreshes at the meter cadence (60 ms); the surface rewrites only keys whose frame changed and drops stale frames.
- **GUI:** the desktop bridge gains `Animation(id)`. The Soundboard screen requests it when a clip's artwork changes and cycles the frames in its image; other screens keep static artwork.
