# Design
## Removal
| Area | Change |
|---|---|
| Plugin | Delete `plugins/insta360`, `internal/camera` (Go and `native/camera.cpp`) and `cmd/snoofer/insta360.go` |
| Build | Delete `scripts/build-camera.ps1`. `scripts/build.ps1` no longer builds the camera DLL, and no longer lists it among the outputs it checks for locks |
| Deck | The default Meetings page keeps the call on row 1; row 2 is empty. `drawIcon` drops `camera-privacy`, `tracking`, `framing` and `camera-reset`; the shared camera drawing stays for `discord-video`. `keyArea` drops the same names |
| GUI | The Meetings screen shows only the Discord card. The deck preview icon map drops the camera icons, and the rebuild signature drops `insta360` |
| Docs | `docs/plugins.md` drops the Insta360 section and the `no_insta360` tag. `docs/ui-contract.md` drops the Camera card and the camera keys from the Meetings page and the personal layout description |

## Personal settings
While Snoofer is stopped:
1. Back up `bin/snoofer.json` to `.local/rollback/remove-insta360/`.
2. Delete `plugins.insta360`.
3. Clear the Meetings page keys bound to `insta360.*`, leaving the positions empty.
