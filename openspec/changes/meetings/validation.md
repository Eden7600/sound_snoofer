# Validation — 2026-10-07

Commits: 60ed57a (design), 4c79dd2 (camera access), 5bd5066 (insta360), 304826e (discord), 79d3a0d (mute link), 67a972d (GUI and deck icons).

- `go vet ./...` and `go test ./...` passed. New tests cover Link 2 packet parsing across firmware lengths, privacy gating, full body in Group, verification (Pending, then Failed), absence, preview and Stop; Discord framing, first-use approval, denial, saved-token refresh and rotation, invalid_grant, verified and failed writes, preview, reconnect without replay, setup and closed states, and credentials never appearing in controls; the mute link's connect imposing, adoption of Discord changes, acknowledgement of Snoofer's own writes, deafen, a single in-flight write without retry, and the audio plugin's Mic mute API, including Mic stack Off.
- Opt-in live read through snoofer-camera.dll from the connected Link 2, without writing: status, privacy and framing read; an unknown property set reported as not found (see the design's live read note).
- `scripts/check-gui.cjs` passed: Meetings cards, state lines, setup hint, Connect visibility, requests from Privacy, Mute and Leave call, the Enable card and no horizontal overflow at 800×600. Native-size deck keys were inspected (docs/design/meetings.png).
- Strict OpenSpec validation passed. `scripts/build.ps1` built bin/snoofer.exe and bin/snoofer-camera.dll after a graceful stop; `--check` accepted the personal snoofer.json, where both new plugins are absent and therefore disabled. Snoofer was relaunched with no arguments.
- Not verified: any camera write, Discord RPC against the real client (authorization, token exchange with the redirect, video and screen-share event payloads), and the mute link in a real call. These remain hardware acceptance tasks 8 and 9. The privacy control's value 2 is unexplained; privacy state is read from the status flags only.
- Race detector unavailable: CGO is disabled. No toolchain changes.

## Stream Deck page — 2026-10-07
- 650a69a adds the Meetings page and the Home go-to key to the default layout; default-layout tests cover the page, its dials, Leave standing apart, the Home key and the page dial reaching Meetings after Media. `go vet` passed; `go test ./...` passed except `internal/config` TestVoiceExample, which loads config.voice.json, deleted from the working tree outside this change (not committed).
- Personal layout: with Snoofer stopped, the Meetings page and the Home key at r4c6 (key 32, previously empty) were added to bin/snoofer.json. Nothing else in the file changed (174 lines added, the Home key's 2 lines replaced). `--check` passed and Snoofer was relaunched with no arguments. Until insta360 and discord are enabled, the page's camera and Discord keys show N/A.
