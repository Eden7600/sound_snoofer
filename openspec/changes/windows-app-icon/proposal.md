## Why
The mascot was registered only as a notification-area icon. The executable has no Windows icon resource, so Explorer and the controls taskbar entry can use a generic icon.
## What Changes
Embed the existing multi-resolution mascot ICO in the Windows executable and apply it to the separately allocated controls window. Retain tray-only operation when controls are closed.
## Impact
Windows build resources and controls-window presentation only. No audio, configuration or process-lifecycle policy changes.
