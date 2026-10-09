# Inactive deck keys and artwork space
## Why
- **Mic stack off.** Disabling the mic stack blanks the mic-path keys and dials. The user wants them to stay visible, greyed out in an inactive state, so the layout stays readable and it is clear the controls exist but do nothing now.
- **Unused space.** Scene and clip keys no longer show routine status words (`deck-text-refinements`), but their layout did not reclaim the space: the label sits mid-key above an empty row.

## What Changes
- **Inactive controls:**
  - Controls gain an optional `Inactive` flag. The deck draws an inactive key or dial greyed (label and icon in the unknown grey, no value or meter) and ignores its input.
  - Unlike `Hidden`, an inactive binding keeps its position, and covering regions do not reuse it.
- **Mic stack:** while the mic stack is off, mic-path controls are inactive, no longer hidden. The GUI is unchanged.
- **Artwork keys without a value:** the label sits at the bottom of the key, on up to two lines, and the artwork band grows to fill the space above it. Square and pie artwork are cropped to the band's shape.

## Impact
- **Code:**
  - `snoofer/controls.go` (one optional field);
  - `plugins/audio` (mic stack);
  - `plugins/streamdeck` (tiles and input);
  - `internal/streamdeck` (inactive treatment, adaptive band);
  - `docs/ui-contract.md`.
- **Unchanged:**
  - What disabling the mic stack does to routing.
  - Hidden and unavailable semantics.
  - The GUI.
