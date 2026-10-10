# Login startup
## Why
Snoofer is launched by hand after every sign-in. The user wants it to start with Windows, switched on or off from the GUI.

## What Changes
- **Start at Login:** a core toggle (`core.startup`) that registers this copy of Snoofer to start when the user signs in to Windows, or removes that registration.
- **Mechanism:** the per-user Run entry `HKCU\Software\Microsoft\Windows\CurrentVersion\Run\Snoofer`, launching this executable with its absolute configuration path. No service, scheduled task, elevation or installer.
- **Truthful state:** the toggle shows the Run entry Windows actually holds, read back after every change: On only when it launches this executable and configuration and Windows' startup-apps setting has not disabled it. An entry for another copy, or one disabled in Windows, shows Off with a note. A read failure shows N/A with the error.
- **Preview:** under `--dry-run` the toggle shows the observed state but is unavailable, so development previews never change login startup.
- **GUI:** a Startup card at the top of Diagnostics holds the toggle.
- **Not persisted in snoofer.json:** Windows owns the setting; Snoofer has no saved preference that could disagree with it.

## Impact
- **Code:** `app` (Run entry read/write, core control, tray action), `app/web/app.js` (Startup card), `scripts/check-gui.cjs`.
- **Docs:** `docs/ui-contract.md`, `README.md`, `docs/application-guide.md`.
- **Data:** one per-user registry value, written only when the user turns the toggle on. No configuration changes.
