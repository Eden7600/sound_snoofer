# Validation
2026-10-06, commits c406a65 through 0f472db.

## Automated checks
- `go vet ./...` and `go test ./... -timeout 120s`: pass. The nowplaying tests were run 5 times in a row; they cover:
  - merging and de-duplication, focus, interpolation, pending/observed/declined, and preview;
  - WebSocket origin and token checks, size limits, replacement and token reset;
  - saving the extension files.
- `node --test plugins/nowplaying/extension/logic.test.mjs`: 4/4.
- `scripts/check-extension.cjs`: pass, run 3 times. The page script in Chrome reports metadata, state and controls, and runs toggle, the page's `nexttrack` handler and seek.
- `scripts/check-gui.cjs`: pass. It covers the Media screen's cards, play, seek, focus, mute and the extension card.
- `scripts/check-desktop.cjs` and `openspec validate now-playing --strict`: pass.
- `scripts/build.ps1`: builds `snoofer-media.dll` with C++/WinRT at `/W4 /WX`.

## Native probe
`SNOOFER_MEDIA_PROBE` against the built DLL found Brave's single session ("U2 – The Troubles", 4:45, seekable) with a 12 KB cover. `current` initially came back false because WinRT returns new session objects; it was fixed to compare by AppUserModelID.

## Incident
The first build saved the new bridge token from inside `Start`. The host holds its lock while starting plugins, so the save deadlocked the host, and plugins after `nowplaying` (Soundboard, Stream Deck, VR) never started.
- **Recovery:** the graceful stop correctly refused to kill the process. With the user's approval, only that process (PID 34576) was force-ended.
- **Fix (0f472db):** the token is created in memory at start and saved from the worker after startup.
- **Regression test:** the bridge rig's SaveSettings now fails if it is called during `Start`.

## Personal setup and relaunch
- **Personal setup:**
  - `nowplaying` enabled;
  - Media page added (sessions in r1 c1–c8, transport and Focus on r2, Up/Down, dials Media and Playback);
  - Home key 33 (r4c6) set to Go to Media, and Home dial 3 set to the media dial;
  - `snoofer.exe --check` passes; backup `snoofer.json.before-media`.
- **Relaunch:** Snoofer was relaunched. It listens on `127.0.0.1:47815`, and the token was saved after startup.

Hardware acceptance (task 9) is pending, including installing the extension in Brave.

## Revision: standalone store extension (commits bb81bc8 through d565bec)
Review found the export pattern unsuitable for store publishing, and the first install showed nothing. The repository copy lacked the generated `config.js`, so the background failed on its first line.

### What changed
- The extension is now its own project in `extension/`, with shared sources and generated Chrome and Firefox manifests.
- There is no token, export or pairing. The bridge accepts `chrome-extension://` and `moz-extension://` origins on loopback, and refuses mismatched protocol versions with a reason.
- Windows sessions whose titles match a connected browser's tab are hidden, which covers Firefox.

### Checks
- `go vet ./...` and `go test ./...`: pass. The nowplaying tests were run 3 times in a row; they cover web origins refused, both extension origins accepted, protocol refusals (4002 plus a reason), title de-duplication and a legacy token ignored.
- `npm test` in `extension/`: 7/7, covering logic, manifests and build output.
- `npm run package`: produces both zips with `manifest.json` at the root. Windows' own bsdtar is used, because Git's GNU tar cannot write zips.
- `scripts/check-extension.cjs`: pass, run 3 times.
  - The page script checks in Chrome pass.
  - **Live end to end:** the built extension, in Brave with a fresh profile, connects to a stand-in bridge. It sends a protocol 1 hello as Brave from a `chrome-extension` origin and reports the playing tab. Bridge commands toggle the page, use its `nexttrack` handler and mute the tab.
- `scripts/check-gui.cjs` (the extension card has no export or token buttons) and `openspec validate now-playing --strict`: pass.
- **Real bridge:** the built extension in a fresh Brave profile connected to the relaunched Snoofer, unrefused, and the connection is shown as ESTABLISHED to Snoofer's PID.

### Personal setup and limits
- **Config:** the legacy token was removed from the personal config (backup `snoofer.json.before-extension`), and Snoofer was relaunched.
- **Not tested:** Firefox, which is not installed here. Its manifest is covered by the build tests only.

## Revision: nothing playing, focus and bottom-row controls (commits df0124d through 8564e65)
### Cause
- **Diagnosis:** the user's Brave extension was connected (an ESTABLISHED connection to Snoofer), and the plugin, run against the real companion, saw Brave's Windows session.
- **Combination:** Snoofer hid Brave's Windows session as soon as the extension connected. Meanwhile the extension reported nothing, because the video tab was opened before the extension was installed. Content scripts reach only pages loaded afterwards, and an already-playing element never fires `play`.

### Fixes
- **De-duplication:** title only, so a Windows session is never hidden unless the extension reports its tab.
- **Existing tabs:** the extension injects into open tabs on install (`scripting` permission).
- **Already-playing media:** `page.js` adopts media already in the document, or seen through `timeupdate`.
- **Focus:** when exactly one session plays and the focused one does not, focus moves to it.
- **Media page:** transport keys moved to the bottom row (keys 28–32), with sessions on r1–r3 c1–c8, in the defaults and the personal layout (backup `snoofer.json.before-media-row`).

### Checks
- `go test ./...`: pass. New tests cover a Windows session kept while its tab is unseen, focus moving to the only playing session, and the bottom-row Media page.
- `npm test`: 7/7.
- `scripts/check-extension.cjs`: pass, run 3 times.
  - **Chrome page stage:** `page.js` injected into a tab already playing reports it at once.
  - **Brave live stage:** the install hook re-injects into open tabs without duplicating the session, and the tab stays controllable.
- **Build and relaunch:** built and relaunched; `snoofer.exe --check` passes.

### Not automated
Restarting an unpacked extension after `runtime.reload()` does not work under automation, so the update path in a real install is covered by hardware acceptance.
