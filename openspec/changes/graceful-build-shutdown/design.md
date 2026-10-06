# Design

`scripts/stop.ps1` targets only processes whose executable path is this repo's canonical bin/snoofer.exe. Optional ProcessId narrows the target for isolated tests. Post WM_CLOSE only to the host's hidden SystrayClass window: the installed systray implementation invokes the existing cancellation callback. The host runs Stop and the audio worker closes callbacks, logs out and releases its DLL. Retain process handles and wait for host and captured controls children. Missing tray, cleanup error or timeout is failure; never fall back to Stop-Process.

`build.ps1` calls this script before existing exclusive-file checks. The development workflow restores the prior launch configuration after validation as before. `check.ps1` already delegates to build.

GUI smoke launches __controls only, without audio plugins. Finally blocks close stdin and await exit with cleared timeout timers. If this audio-free child hangs, kill only that child, await exit and fail validation. Add an isolated disabled-plugin tray smoke for graceful stop. Existing worker tests plus a focused adapter check verify monitoring stops before Logout and DLL release.

This does not prove or repair pre-existing native registrations left by hard kills. Do not restart Voicemeeter or alter audio state to hide cleanup failure.
