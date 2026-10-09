# Deck rows layout
## Why
- **Grey and blue only.** Every key is grey or cyan unless it is pending or failing, so nothing says which part of the studio a key belongs to.
- **Hard to read.** Labels use a 5×7 bitmap font at 1× (7 px tall), and `fit` cuts long labels and values with "...".
- **Shared lines truncate.** Any layout that puts the icon beside text cuts off one or the other.
- **Chosen design:** the user picked "Rows · left" from the Stream Deck aesthetics canvas (round 3), with the round 2 colour system.

## What Changes
- **Rows:** each key has three full-width rows, flush left: the label (up to two lines), a 30 px icon, and the value. Nothing shares a line.
- **Fitting:** labels and values wrap and then shrink to fit. They are only cut, with an ellipsis, when they cannot fit at the smallest size; tests keep built-in labels clear of that.
- **Area colours:** the icon names the area of the studio:

  | Area | Colour |
  |---|---|
  | Mic | cyan |
  | Playback | blue |
  | Recorder | violet |
  | Soundboard | green |
  | Media and apps | magenta |
  | Lights | ivory |
  | Pages and other controls | grey |

  An off key is dark, with its label and icon in the area colour. An on key fills with the area colour.
- **Problem states:** red and amber stay reserved for problems.

  | State | Drawn as |
  |---|---|
  | Muted, recording | Red fill |
  | Error | Red rounded frame and a `!` |
  | Pending | Amber dashes and an amber value |
  | Unknown | Grey N/A |
- **Corners:** the keycap rounds the visible corners. Text and icons stay inside an 8 px inset, and full-bleed fills and artwork run to the edges.
- **Artwork keys:** artwork fills the top of the key, with the label and value rows beneath it.
- **Touch strip in the same style:**
  - **Control dials:** the label in the area colour, a large condensed value, a position track, and a full-width meter in flat green, amber and red zones.
  - **Media dial:** artwork beside the title and time.
  - **Page dial:** `← Previous`, the current page in large type, and `Next →`.
- **Typeface:** Archivo Bold for labels and Archivo Condensed ExtraBold for values. Both are embedded, under the SIL Open Font License.

## Impact
- **Code:** `internal/streamdeck` (renderer, fonts, palette, tests), `docs/ui-contract.md`, `third_party/README.md`.
- **Dependencies:** `golang.org/x/image` is already a dependency; its `font/opentype` package rasterizes the fonts. No new module.
- **Unchanged:**
  - Control semantics, bindings, layouts, state precedence, meter readings, changed-key caching, rotation and JPEG output.
  - The GUI deck editor's own previews.
