# Tasks
Each numbered block is one reviewable commit.

## Implementation
- [x] 1. `docs(apps)`: specify connection reports and the Third-party apps screen.
- [x] 2. `feat(snoofer)`: `Connection` type and `Control.Connection`, with deep copy on publish and snapshot. Tests cover copy isolation and revision behavior.
- [x] 3. `feat(gui)`: Third-party apps screen (summary, cards, relative times, stale marker, copy with fallback, not-monitored list). Add GUI model tests, check-gui fixtures and a screenshot; update the UI contract and docs/plugins.md.
- [x] 4. `feat(hue)`: Hue Bridge and Hue Sync reports, with since/last event/reconnect tracking and bridge name/version from the probe. Tests use the fake bridge and fake app.
- [x] 5. `feat(audio)`: Voicemeeter, callback monitor, recorder, ASIO, Element and Windows defaults reports. Track the DLL path, login code, version (optional export), connected since and last error; extend `windowsaudio.Result`. Tests use the existing fakes.
- [ ] 6. `feat(vr)`: SteamVR report with check timestamps.
- [ ] 7. `feat(streamdeck)`: separate device state from layout status, plus a device report. Tests use the surface fake.
- [ ] 8. `feat(soundboard)`: playback companion/renderer report.
- [ ] 9. `feat(media)`: media keys report.
- [ ] 10. Validate (gofmt, vet, test, GUI checks, OpenSpec, canonical build), relaunch, inspect the live screen against the real integrations and record the results.

## Acceptance (real environment)
- [ ] 11. Close and reopen Voicemeeter, Hue Sync and SteamVR, and unplug and replug the Stream Deck; confirm each card's state, since time and retained error, and that Copy details pastes a usable report.
