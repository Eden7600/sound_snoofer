# Validation

## Automated (2026-10-06)
- `go vet ./...` and `go test ./...` pass. New tests cover dial ballistics (instant attack, 24 dB/s release, 1.5 s hold then fall, reset on unknown readings and binding changes), position parsing (dB, percent, `Sync 62%`, Off, N/A), and the renderer's gradient, peak tick and position track.
- `node --test app/web/model.test.mjs` passes, including the GUI ballistics and gain position helpers.
- `check-gui.cjs` passes: Engine row on Audio only, no header health badge or host footer, a live meter with peak marker, a 75% position track for -6 dB, and the -60/-30/-12/-3/0 scale.
- `check-desktop.cjs` passes against the new binary (native WebView2 window, revisioned actions, close/reopen, graceful tray shutdown).
- Strict OpenSpec validation passes. `scripts/build.ps1` built `bin/snoofer.exe` after stopping Snoofer gracefully; Snoofer was relaunched with the personal configuration.
- `go test -race` was not run: the toolchain has cgo disabled, and the toolchain was left unchanged.

## Visual inspection
- `docs/design/dial-meters.png` is a native-size render of the dial strip (upright): metered gain, a near-clip mic with red peak tick, an unknown reading (`LEVEL N/A`) and Hue brightness with a percentage track.
- `docs/design/gui-audio.png` shows the redesigned mixer strips.

## Hardware acceptance (2026-10-06)
Confirmed by the user on the Stream Deck + XL: meter motion and peak hold look right at the new refresh rate, mic keys blank and return with the stack toggle, and overlapping clips play together.
Pending: Stop silencing every overlapping clip has not been confirmed.
