# Validation

## Automated (2026-10-06)
- `go vet ./...` and `go test ./...` pass. New coverage:
  - **Motion:** a fake-bridge test covers On, Pending until observed, Mixed, Off, the enable-only-differing writes, the failure reason and a zone without sensors. The room association is tested separately.
  - **Collections:** published metadata for room-scene slots and per-room scenes, and slots growing past twelve.
  - **Regions:**
    - row-major fill around manual cells;
    - a fixed frame across overflow pages;
    - sequential same-source fill;
    - parallel overflow;
    - skipping Hidden members;
    - validation (overlap, range, source, mixed with a prefix, no free key);
    - legacy prefix equivalence;
    - editor region data;
    - edit operations with draft isolation.
  - **Go-to keys:** rendering, jump and return, and a deleted page that is inert. The shown page is marked Here.
  - **Default layout:** pages, expansion and a JSON round trip.
- Running `go test ./...` found that the strict layout decoder rejected `regions`. This is fixed in `fix(streamdeck): decode saved page regions`.
- `node --test app/web/model.test.mjs` passes.
- `check-gui.cjs` passes. It covers:
  - the Motion row (press, then hidden without sensors);
  - one-based key numbers;
  - range selection by Shift+click and Shift+arrows without dispatch;
  - Add region with the 18-key range, Region source change and Remove;
  - the overlapping-selection guard.
- `check-desktop.cjs` passes against the new binary.
- Strict OpenSpec validation passes.
- `scripts/build.ps1` built `bin/snoofer.exe` after a graceful stop.
- Soundboard clip collection metadata has no unit test. The clip controls are only published by the native-backed worker, and its live test needs hardware.

## Personal layout
While Snoofer was stopped, `bin/snoofer.json` was backed up and migrated as design §4 describes:
- **Home:** the twelve room-scene slots were removed. Motion is at r2c9, and the go-to Soundboard and Lights keys are at r4c8 and r4c9.
- **Soundboard:** converted to a clips region in r1–r4 c1–c8, with Overlap, Home and Stop in column 9.
- **Lights:** a new page after Soundboard.

The migrated file passes `snoofer.ValidateEnabled` with every compiled plugin. Snoofer was then relaunched with it.

## Visual inspection
- **Native key renders:** Motion (On, slashed Off, Mixed) in `docs/design/hue.png`, and folder go-to keys (with Here active) in a scratch preview.
- **GUI captures:** the Lights screen with the Motion row, and the deck editor with a tinted, numbered region and a dashed selection. The references in `docs/design` were refreshed.

## Revisions after review (2026-10-06)
- **Option labels:** keys show a selection's option label (the Room key showed a room ID); covered by a binding-tile test.
- **Go-to keys:** they stay on Home only. No go-to control is published for the Home page, and the default content pages no longer bind one. The go-to test now returns Home with the page dial's press and asserts no Home key exists.
- **Room scope:** fake-bridge tests cover:
  - choosing a zone moves and saves the selection;
  - unknown IDs are dropped;
  - only chosen groups are offered, and the Room key hides with one;
  - unchosen rooms' scenes are not published;
  - an empty list is rejected, and a selection outside the list is rejected by the registry;
  - a saved selection outside the list moves on connect.
- **Text controls:** layout validation, the editor and the GUI picker reject them as bindings.
- **GUI check:** the Rooms card sends the new list, and the last chosen room is disabled.
- **Build and checks:** `go test ./...`, `go vet`, the GUI and desktop checks, strict OpenSpec and `scripts/build.ps1` pass.
- **Personal config** (Snoofer stopped, backup in the scratchpad): the two Home go-to keys on Lights and Soundboard were removed, and Hue rooms set to Cody Office (already selected). The file passes `snoofer.ValidateEnabled`, and Snoofer was relaunched.

## Not yet verified
Task 9 hardware acceptance is still open.
