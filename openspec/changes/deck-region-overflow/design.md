# Design
## Edit
A new editor control, `streamdeck.region-clip` (text, set), takes `index,on` or `index,off`. It goes through `Layout.editRegion` as op `clip`, with the same indexes as `region-source` and `region-remove`: key regions first, then dial regions. On a legacy prefix page, the edit first converts the prefix into an explicit region, as the other region edits already do. Like every other draft edit, the result is validated and only saved by Save layout.

## Expansion
In `Layout.expanded`, a stacked region (one with `sources`) contributes only its first set when it is clipped. Unclipped stacks are unchanged. Single-source key and dial regions already honour Clip.

## Editor view
- `EditorRegion` gains:
  - `Clip`.
  - `Stacked`: the labels of the sources after the first, joined with " + ". It is empty when no other source shares the region's rows.
- `EditorView` gains:
  - `Sets`: how many sets the edited page expands to with the current controls.
  - `Scrolls`: whether the page binds an Up/Down key.

## GUI
- **Region row:** R*n*, the source picker, `+ <Stacked>` when set, the keys or dials, an Overflow checkbox (checked when not clipped), and Remove. The checkbox is not labelled Clip, because Soundboard clips are a region source.
- **Sets note:** in the Regions panel, shown when there is more than one set. It reads `3 sets · Up/Down` on a page with a scroll key, and otherwise `3 sets · page dial visits each`, in attention colour.
- **Make Home:** disabled while the edited page is Home. The plugin also stops marking the draft dirty when Home does not change.

## Personal layout
`bin/snoofer.json` is user configuration and is not committed. The two Home regions get `"clip": true` while Snoofer is stopped. Positions and sources are unchanged.
