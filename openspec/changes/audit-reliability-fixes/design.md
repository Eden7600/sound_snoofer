# Design
Numbers match [docs/repo-audit.md](../../../docs/repo-audit.md).

## 1. Deck editor lists
`editorView` initialises `view.Keys` and `view.Dials` with `make([]EditorSlot, 0, n)`, so JSON carries `[]` instead of `null`. A Go test marshals the view for a model without dials and requires `"Dials":[]`.

## 2. Disconnected device rows
The plugin sends `normal-source`, `normal-output`, `vr-profile-source` and `vr-profile-output`. The worker checks those rows:
- Normal rows: the chosen value must be in `routing.MicrophoneOptions` / `routing.PlaybackOptions` for a fresh snapshot, as before.
- VR rows: the chosen value must be in the VR option lists. The publish code that builds `VRSourceOptions`/`VROutputOptions` moves into one function, used by both publish and the check, with the fresh snapshot.
`off` and an empty playback stay exempt, as before. Bare `source`/`output` rows are still checked.

## 3. Soundboard readiness
`soundboardReady` stops rebuilding the A-bus rule. Every planned operation on a `Strip[7].` parameter must match the snapshot, and at least one A-bus operation must be planned. The B2 (never to Element) and B3 (microphone route) checks stay, since the planner does not write those zeros.

## 4. Callback retry
`SetCallback` short-circuits only when the requested state is already in effect: `enable == m.enabled && sameHook(...) && (m.active || !enable)`. A failed registration retries, at most every 5 s, because the worker requests the callback on every iteration. The `unsafeClose` gate is unchanged.

## 5. Deck reads at the deadline
`device.io` checks `WAIT_OBJECT_0` before the deadline. When the deadline or cancellation fires, it cancels and waits for the result; if the transfer still completed with bytes, it returns them. The 1 s deadline stays: the read loop simply reissues the read, and the HID driver buffers reports meanwhile. This is native I/O with no fake; it is verified by build and on hardware.

## 6. Browser bridge input
- Go: a message that fails to decode is skipped (`continue`), not treated as a closed connection.
- Extension: `sessions()` coerces each report field (strings with `String`, numbers with `Number.isFinite` checks, booleans with `!!`).
- Artwork: only `http:`/`https:` URLs are fetched, with `credentials: "omit"`. Other values fall back to the tab favicon.

## 7. Interface-mic gating
Each changed `Patch.asio[k]` operation gates input strip `k/2`, for any `k`. An A1 device change gates every strip that owns a `Patch.asio` cell in the plan, rather than strips 0 and 1 only.

## 8. Tape ownership save
The controller gains an optional `BeforeTapePlay func() error`, called in `Tape` right before it routes the tape for a new playback, after the recorder state has been read. The worker sets it to the existing ownership save. Pause, resume and recording commands no longer save. Save still precedes the routing write.

## 9. Audio settings validation
`applyEdit` runs `config.LoadChoices` with the edited configuration and the saved choices before saving. If the result reports a `StateError` that the running configuration did not, the edit is refused with that error and nothing is saved. Choices that were already broken stay editable, so the user can repair the configuration.

## 10. GUI poll
`poll` reschedules in `finally`. The deck slot updater checks `!slot` before reading `slot.Control`.

## 11. Hue backoff and stream ownership
- The worker resets `retryDelay` when the event stream delivers its first message for the current generation, not when resources load. The Sync link does the same on its first message.
- `Client` owns a cancellable context. `Events` stops when either its caller's context or the client's ends, and `close` cancels it, so `reconnect` ends the old stream.

## 12. Discord token exchange
The worker records an in-flight exchange. `startExchange` with a refresh token does not start a second exchange while one runs. The result is accepted while the worker is waiting for an exchange on a live connection, even if the pipe reconnected meanwhile. The rotated token is still saved in every case.

## 13. Now Playing shutdown and port retry
`close(i.done)` is deferred first, so it runs last, after the bridge is closed. While no bridge is listening, the worker retries `listen` at most every 5 s.
