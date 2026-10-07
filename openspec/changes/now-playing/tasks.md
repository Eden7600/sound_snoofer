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
- [x] 33. Validate: all checks and the canonical build; migrate the personal Media page; relaunch.
- [x] 35. `docs(media)`: hold to focus, removal of the Focus key, reverting the single-playing rule, and the Playback dial (§4d).
- [x] 36. `feat(streamdeck)`: key releases and the opt-in `hold` operation. Tests cover tap, hold, latency for other keys and a release after a generation change.
- [x] 37. `refactor(nowplaying)`: session hold focuses; remove the Focus control and the GUI Focus button; revert the single-playing rule. Tests are updated.
- [x] 37a. `feat(appaudio)`: app keys offer hold, and the `appaudio.focus` dial follows the focused app. Includes tests.
- [x] 38. `feat(streamdeck)`: Media page dials (media, Playback, focused app, apps) and no Focus key (default and personal). Tests are updated.
- [x] 39. Validate: all checks and the canonical build; relaunch.
- [x] 40. `docs(media)`: dial order, no duplicate dials and transport on Home (§4e).
- [x] 41. `feat(streamdeck)`: the `Mirrors` field, which keeps the focused app off region dials, with `appaudio.focus` setting it. Includes tests.
- [x] 42. `feat(streamdeck)`: Playback first on the Media page, and Now playing transport on Home (default and personal). Tests are updated.
- [x] 43. Validate: all checks and the canonical build; relaunch.
- [x] 44. `docs(media)`: the Home strip (§4f).
- [x] 45. `feat(streamdeck)`: `Region.Fixed` (no overflow sets), and the Home strip in the default and personal layouts. Includes tests.
- [x] 46. Validate: all checks and the canonical build; relaunch.
- [x] 44. `docs(deck)`: Home revamp and clipped regions (§4f).
- [x] 45. `feat(streamdeck)`: clipped regions, the default Home revamp and the personal Home migration. Tests cover clipping (no overflow sets) and the Home layout.
- [x] 46. Validate: all checks and the canonical build; relaunch.
- [x] 47. `docs(media)`: sticky focus, Home focus strips and Reset focus (§4g).
- [ ] 48. `feat(streamdeck)`: per-key `op`, region `op`, and the Reset focus key (publishing, local press, binding eligibility). Includes tests.
- [ ] 49. `feat(nowplaying)`, `feat(appaudio)`: sticky focus and `reset`; taps no longer focus. Includes tests.
- [ ] 50. `feat(streamdeck)`: Home strips with `op: hold` and Reset focus (default and personal). Includes tests.
- [ ] 51. Validate: all checks and the canonical build; relaunch.

## Hardware acceptance
- [ ] 9. With Brave and a Windows player:
  - the extension (loaded from `extension/dist`) connects without setup in Brave and Firefox;
  - two tabs appear separately and pause independently;
  - YouTube Next works;
  - the dial seeks and plays/pauses the focused session;
  - focus follows new playback;
  - Spotify (or another player) appears alongside;
  - the strip shows artwork and progress.
