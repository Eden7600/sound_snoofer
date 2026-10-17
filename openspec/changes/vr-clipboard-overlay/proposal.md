# VR clipboard overlay

## Why
SteamVR's keyboard does not give this user Copy/Paste. Add a small controller-operated clipboard pad in the SteamVR dashboard so they can use the Windows clipboard in desktop applications while wearing the headset.

## What Changes
- Optional independent VR Clipboard plugin, disabled by default, with a dashboard tile and large Copy, Cut, Paste, Select All, Enter, Tab and Backspace buttons.
- Preserve the target desktop window when using the panel, show its executable name, capture a target per gesture, and reject stale or unsafe input rather than sending to a different application.
- Use the installed SteamVR OpenVR loader and existing Go image/font and Windows API dependencies. Run the overlay in a hidden worker of the same executable so SteamVR shutdown or native failure cannot terminate the audio host.
- Publish concise connection state through existing plugin/Third-party apps surfaces. No clipboard text, history, synchronization, full keyboard, new desktop screen, audio-policy change or saved layout migration.

## Current-session constraint
The user explicitly prohibits restarting Snoofer or interrupting their active VR session until they say otherwise. Only source changes and isolated tests are authorized now. Do not run scripts/build.ps1 or scripts/check.ps1, deploy, change personal config, start the worker, contact SteamVR, send input or read the clipboard. Canonical build and physical acceptance remain open.
