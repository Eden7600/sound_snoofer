# Tasks
Each numbered block is one reviewable commit.

## Implementation
- [x] 1. `docs(media)`: design the Now playing plugin, Windows sessions, browser bridge, deck dial artwork and Media screen; update the UI contract.
- [x] 2. `feat(mediasessions)`: the C++/WinRT companion and Go wrapper. Covers the build script, JSON snapshot, artwork and commands. Includes JSON parsing tests and an opt-in live probe.
- [x] 3. `feat(nowplaying)`: plugin core. Covers merging sources, de-duplication, focus, controls, interpolation, commands with pending/observed, and artwork scaling. Fake-source tests cover it.
- [x] 4. `feat(nowplaying)`: browser bridge server. Covers origin and token checks, the protocol, bounds, replacement and saving the extension files. Tests use a WebSocket client.
- [x] 5. `feat(extension)`: the MV3 extension: page, bridge and background scripts. Node tests cover pure logic (merging and command routing); a Playwright check loads it in Chrome against a test page.
- [x] 6. `feat(streamdeck)`: dial artwork, `m:ss / m:ss` progress and the default Media page with its Home additions. Includes tests and a native-size preview.
- [x] 7. `feat(gui)`: the Media screen and Browser extension card. The GUI check covers both.
- [x] 8. Validate: Go, JS and GUI checks; desktop check; OpenSpec; the canonical build. Add the personal Media page, Home go-to key and media dial; relaunch.

## Hardware acceptance
- [ ] 9. With Brave and a Windows player:
  - the extension installs and connects;
  - two tabs appear separately and pause independently;
  - YouTube Next works;
  - the dial seeks and plays/pauses the focused session;
  - focus follows new playback;
  - Spotify (or another player) appears alongside;
  - the strip shows artwork and progress.
