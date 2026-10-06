# App audio
## Why
Per-app volume and mute live in the Windows volume mixer, which is slow to reach mid-session. Programs also present themselves badly: browsers and Electron apps open several sessions and processes, games keep silent sessions marked Active, Store apps report resource strings instead of names, and helper processes appear as separate entries. Snoofer needs a deck page and a GUI screen for the apps that matter, plus rules that clean up what Windows reports.
## What Changes
- **Session adapter:** `internal/windowsaudio` gains per-app session control: enumeration on every active render device, process identity, volume, mute and a peak meter.
- **`appaudio` plugin:** groups sessions into apps and publishes one control per app (`appaudio.app-<key>`):
  - Turning adjusts volume and pressing toggles mute.
  - Each control carries the app's icon, a live meter and a pending/observed state.
- **Visibility:** picked apps come first, in your order, even when silent or closed. After them come other apps that produced sound in the last 5 minutes, measured by their meter rather than by Windows' Active flag.
- **Rules:** ordered match rules on the executable can hide, rename or combine apps. Built-in defaults hide Snoofer, Voicemeeter and Windows plumbing. The GUI writes per-app rules (Hide, Rename, Combine into…), and the JSON settings accept patterns.
- **Deck:**
  - **Dial regions:** pages gain dial regions, so a collection can fill dials as well as keys, and both page together.
  - **Apps page:** the default page has five app dials, with icon, volume and meter on the strip; five mirroring mute keys; and Up/Down.
  - **Home:** a go-to key for the page.
- **GUI:** an App audio screen with a strip per app (icon, volume, mute, meter, pin), Hide/Rename/Combine actions, the hidden list and per-app diagnostics.
## Impact
- **Code:** `internal/windowsaudio` (new COM session code), a new `plugins/appaudio`, Stream Deck regions and editor, and the GUI.
- **Unchanged:** Voicemeeter routing, Snoofer's own mixer controls, and Windows' own per-app persistence. Snoofer never stores or reapplies app volumes.
