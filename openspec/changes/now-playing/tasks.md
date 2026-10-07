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

## Revisions after review
- [x] 10. `docs(media)`: standalone store extension, no token or pairing, title de-duplication (§2).
- [x] 11. `refactor(nowplaying)`: drop the token, export and embedding; accept `moz-extension` origins; add the protocol handshake and title de-duplication. Tests are updated.
- [x] 12. `feat(extension)`: `extension/` project. Covers shared sources, Chrome and Firefox manifests, the build and packaging script, the popup (status, port, Firefox permission) and icons from the Snoofer mascot. Node tests cover logic and the build output; the page-script check is moved.
- [x] 13. `feat(gui)`: the extension card without export. The GUI check is updated.
- [x] 14. Validate: all checks and the canonical build; remove the legacy token from the personal config; relaunch.
- [x] 15. `docs(media)`: revise de-duplication (title only), injection into existing tabs, the single-playing focus rule and the bottom-row controls.
- [x] 16. `fix(nowplaying)`: title-only de-duplication and the single-playing focus rule. Tests cover both.
- [x] 17. `fix(extension)`: inject into open tabs on install and start; adopt media that is already playing. The live Brave check covers a tab playing before the extension loads.
- [x] 18. `feat(streamdeck)`: Media page controls on the bottom row (default and personal layout). Tests are updated.
- [x] 19. Validate: all checks and the canonical build; relaunch.
- [x] 20. `docs(media)`: refined seeking (§3a).
- [x] 21. `feat(snoofer)`: `Progress` telemetry excluded from revisions; deck rendering from interpolated progress. Tests cover revision stability and rendering.
- [x] 22. `fix(nowplaying)`: progress telemetry, coalesced scrubbing and optimistic progress. Tests cover coalescing, stale-free turning and quiet timeout.
- [x] 23. `fix(gui)`: interpolated progress on session cards and an optimistic slider. The GUI check is updated.
- [x] 24. Validate: all checks and the canonical build; relaunch.
- [x] 25. `docs(media)`: one Media deck page for apps and media, with deck filters (§4a).
- [x] 26. `feat(appaudio)`: Apps deck filter (All, Pinned, Off), saved, with filtered apps Hidden. Includes tests.
- [x] 27. `feat(nowplaying)`: Media deck filter (On, Off), saved, hiding sessions, the dial and transport. Includes tests.
- [x] 28. `feat(streamdeck)`: the combined default page; Home go-to keys. Tests cover the layout and paging. Migrate the personal layout.
- [x] 29. Validate: all checks and the canonical build; relaunch.
- [x] 30. `docs(media)`: filters free space (§4b).
- [x] 31. `feat(streamdeck)`: stacked regions, Hidden bindings yielding, and the editor label. Tests cover row allotment, overflow sets, yielding and the Stream Deck control exception.
- [x] 32. `feat(streamdeck)`: the Media page uses the stacked region and the five-dial app region (default and personal). Tests cover all four filter states.
- [x] 34. `feat(placeholder)`: coloured placeholder artwork for apps and sessions without a logo (§4c). Tests cover determinism, distinct colours, the artwork contract and plugin use.
- [ ] 33. Validate: all checks and the canonical build; migrate the personal Media page; relaunch.

## Hardware acceptance
- [ ] 9. With Brave and a Windows player:
  - the extension (loaded from `extension/dist`) connects without setup in Brave and Firefox;
  - two tabs appear separately and pause independently;
  - YouTube Next works;
  - the dial seeks and plays/pauses the focused session;
  - focus follows new playback;
  - Spotify (or another player) appears alongside;
  - the strip shows artwork and progress.
