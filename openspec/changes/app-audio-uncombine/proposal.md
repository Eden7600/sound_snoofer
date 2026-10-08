# Separate combined apps
## Why
App audio can rename an app or combine it into another, but cannot undo either from the GUI. The only way back is editing `rules` in `snoofer.json`.

## What Changes
- Each app's details list the rules that name programs into it: its own rename and every program combined into it. Each rule shows the programs it matches.
- **Separate** removes one such rule. Combined programs return as their own apps, and a renamed app returns to the name Windows reports. The change is saved like other app audio edits.
- Pins are left as they are.

## Impact
- **Code:** `plugins/appaudio` (view, edit op), `app/web` (app strip details), `scripts/check-gui.cjs`, `docs/ui-contract.md`.
