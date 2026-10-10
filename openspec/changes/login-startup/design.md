# Design
## Registration
- **Run entry:** the string value `Snoofer` under `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, read at sign-in by Windows for the current user only. Writing it needs no elevation.
- **Command:** `"<executable>" --config "<absolute configuration>"`. The executable is `os.Executable()`; the configuration is the tray's resolved absolute path, so a sign-in launch selects the same instance identity and resolves relative plugin paths (state, soundboard folder) beside the configuration, independent of the working directory Windows uses. `--dry-run` is never registered.
- **Windows startup apps:** Task Manager and Settings > Apps > Startup disable an entry by writing a binary value of the same name under `...\Explorer\StartupApproved\Run` without deleting the Run value. A first byte with its low bit set (`0x03`, `0x07`) means disabled; an even first byte or no value means enabled.
- **On:** write the Run value, then delete any StartupApproved value of that name, so an explicit On in Snoofer overrides an earlier Windows disable.
- **Off:** delete the Run value and any StartupApproved value. Deleting a value that does not exist succeeds.

## Observed state
The tray reads the entry on every publish tick (200 ms; two value reads). The control:

| Observed | Value | Note |
| --- | --- | --- |
| No Run value | Off | |
| This command (case-insensitive), enabled | On | |
| This command, disabled in Windows | Off | Disabled in Windows startup apps |
| Another command | Off | Set for another copy: `<command>` |
| Read failed | N/A, unavailable | The error |

Turning it on from an "another copy" state replaces that registration with this copy. After a write the tray re-reads the entry; if the observed state does not match the request, the notice reports that login startup did not change. A successful registry write alone is not reported as success.

## Control and action flow
- **Control:** `core.startup`, Label "Start at Login", Group System, Kind toggle, Operation press. It is published by the tray with `core.open-controls`.
- **Dispatch:** the handler only enqueues `UIAction{Kind: "startup", Enable: <opposite of the published value>}` onto the tray's bounded action channel, returning "controls busy" when it is full, like Open controls. Controls revision checks reject a press against a stale state. The tray goroutine performs the registry write and the read-back, then republishes.
- **Preview:** under `--dry-run` the control is published unavailable with the note "Preview" (joined to any other note), so Dispatch rejects presses.
- **Ownership:** no goroutines, handles or timers outlive a call; each read or write opens and closes its key.

## GUI
Diagnostics starts with a Startup card holding the Start at Login toggle row, above Audio Engine. The card is omitted when the control is absent. Plugins remains enable/disable/retry only.

## Code layout
- `app/startup.go`: the command format and the pure control presentation (portable, unit-tested).
- `app/startup_windows.go`: Run and StartupApproved read/write through `golang.org/x/sys/windows/registry` (already a dependency), with key paths in a value so tests use a scratch key under `HKCU\Software`.

## Limitations
- Moving or renaming `bin` leaves a registration for the old path, shown as Off with "Set for another copy" (or simply missing if the old path no longer exists); turning it on again fixes it.
- Windows may delay Run entries for a few seconds after sign-in. Snoofer's existing reconnection handles Voicemeeter or devices that are not ready yet.
