# Living in the system tray

Build with `./scripts/build.ps1`, then double-click `bin/sound-snoofer.exe`.
It starts live using `config.json` beside the executable (created from defaults
if missing). The orange mascot icon may be inside Windows' hidden tray icons.

Right-click the tray icon and select **Open controls**. The familiar TUI opens
in its own window. **Q**, **Ctrl+C**, or the window close button closes controls;
automatic device selection and routing keep running. Open controls again whenever
you need them. Launching the same executable/profile/mode again also opens controls.
Only one controls window is opened per tray instance.

Select **Quit Sound Snoofer** in the tray menu to stop automatic management.
Quitting leaves current mixer routes and Voicemeeter's recorder transport unchanged.
The tray status shows connection or configuration problems; open controls for details.

For an isolated preview, use `sound-snoofer.exe --dry-run --config PATH`.
Explicit `tui`, `watch`, `devices`, `plan`, and `apply` commands remain available.
`tui` is a standalone terminal session: Q ends its worker. Avoid running a separate
live TUI/watch alongside the tray; the existing single-writer lock protects routing.
No Windows login startup is installed by this change.

The release build uses the Windows GUI subsystem, so no terminal flashes at launch.
Plain `go build` produces a console-subsystem development build instead; use the
script for distribution. To build while an existing executable is in use, specify
`./scripts/build.ps1 -Output bin/sound-snoofer-next.exe` and quit the old app before
switching to the replacement.

If colours are missing, check whether the launching environment defines `NO_COLOR`.
The TUI intentionally honours it. New notification icons can be hidden under the
**^** beside the clock; drag the orange mascot onto the visible tray if desired.
The application verifies icon registration and reports startup failure visibly.

Attached controls prefer Cascadia Mono at an 18-pixel character height, falling
back to Consolas. The standalone `tui` command keeps the caller's terminal font.

The tray asset is generated from the painterly mascot using the built-in image
generation tool. Its transparent source is `docs/assets/sound-snoofer-tray.png`;
run `./scripts/build-icon.ps1` to package 16–256 pixel ICO frames, then rebuild.
The generation prompt and source attribution are recorded in
`docs/assets/sound-snoofer-tray.md`.
