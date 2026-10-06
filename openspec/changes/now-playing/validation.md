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
