## Why
Sound Snoofer manages audio passively and should remain available without keeping a terminal open.

## What Changes
- Default launch is live system tray mode; controls open on demand in a separate TUI process.
- Closing controls detaches them; only tray Quit stops the worker. Native recorder transport remains untouched.
- Tray menu exposes status, Open controls, and Quit. Repeated launches reuse the existing tray instance for the same profile/mode.
- Retain explicit terminal commands and opt-in --dry-run. Ship a Windows GUI-subsystem executable with console attachment for terminal commands.

## Impact
Startup, TUI worker lifecycle, Windows process/console integration, build instructions and lifecycle tests. No audio-routing policy changes.
