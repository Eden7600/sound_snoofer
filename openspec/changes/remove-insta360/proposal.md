# Remove the Insta360 integration
## Why
The Insta360 Link 2 camera integration (privacy, tracking, framing and reset through the camera driver) is unreliable. The user wants it removed entirely.

## What Changes
- **Removed:**
  - the `insta360` plugin and its registration;
  - `internal/camera` and its native `snoofer-camera.dll` source;
  - `scripts/build-camera.ps1`, and the build's `no_insta360` tag and camera DLL handling;
  - the camera keys on the default Meetings deck page;
  - the camera-only deck icons (privacy, tracking, framing, camera reset);
  - the GUI's Camera card on the Meetings screen;
  - the integration's documentation.
- **Personal settings:** `bin/snoofer.json` drops the `insta360` plugin entry and the four Meetings keys bound to it. A backup is kept under `.local/rollback/`.
- **Kept:**
  - The Insta360 Link 2's microphone, an ordinary audio device in the microphone priorities.
  - Discord's Camera key and its icon.
  - Historical OpenSpec changes, which are a record.

## Impact
- **Code:** `plugins/insta360`, `internal/camera`, `cmd/snoofer`, `scripts/build.ps1`, `scripts/build-camera.ps1`, `plugins/streamdeck` (default layout), `internal/streamdeck` (icons, areas), `app/web/app.js`, `scripts/check-gui.cjs`, `docs/plugins.md`, `docs/ui-contract.md`.
- **Data:** a saved configuration that still names `insta360` loads. The host reports it as not compiled into the build, and bindings to its controls show N/A until removed.
- **Build output:** `bin/snoofer-camera.dll` is no longer built. An existing copy is unused and can be deleted.
