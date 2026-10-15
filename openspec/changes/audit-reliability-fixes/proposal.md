# Audit reliability fixes
## Why
The repository audit ([docs/repo-audit.md](../../../docs/repo-audit.md), findings 1–13) found defects where a core feature does not work as intended: the deck editor crashes on decks without dials, an unplugged device can still be saved, the soundboard never becomes ready when an output slot carries it, callback registration is never retried, Stream Deck key releases can be lost, one web page can disconnect the browser bridge, some interface mics are not gated during hardware switches, pausing a recording can be blocked by an unrelated save, settings edits are shown as saved before the worker validates them, one GUI exception stops polling, Hue retries too fast and leaks streams, Discord refreshes can erase the saved token, and Now Playing stops before its listener closes.

## What Changes
Each finding is fixed at its root, with a regression test, as its own commit:
1. **Deck editor:** the editor view always publishes key and dial lists, empty when a model has none.
2. **Device selection:** the "no longer connected" check applies to the Normal and VR microphone and playback rows the plugin actually sends.
3. **Soundboard readiness:** readiness compares the plan's own soundboard-strip operations with the snapshot.
4. **Callback registration:** a failed registration is retried on the next request instead of returning the cached error.
5. **Stream Deck reads:** a read that completed is returned even if the deadline passed; only writes time out.
6. **Browser bridge:** malformed reports are skipped, report fields are coerced to their types, and artwork is fetched only over http(s) without credentials.
7. **Transition gate:** every interface-mic strip is gated, whatever its patch index.
8. **Tape ownership:** tape-routing ownership is saved only when Play will start tape playback.
9. **Audio settings edits:** an edit whose saved choices would not load is refused before it is saved.
10. **GUI polling:** polling reschedules even if rendering throws; the slot updater guards missing slots first.
11. **Hue:** retry backoff resets only after the event stream delivers a message, and closing a client cancels its stream.
12. **Discord:** only one token exchange runs at a time.
13. **Now Playing:** Stop returns after the listener closes, and a busy port is retried.

## Impact
- **Code:** `plugins/streamdeck`, `internal/control`, `plugins/audio`, `internal/voicemeeter`, `internal/streamdeck`, `plugins/nowplaying`, `extension/src`, `internal/routing`, `internal/controller`, `app/web`, `plugins/hue`, `plugins/discord`, `scripts/check-gui.cjs`.
- **Invariants:** unchanged. Save-before-apply is kept for tape playback (finding 8 narrows when the save happens, not its order). Stale-action rejection, drift checks, and the no-retry rule for uncertain one-shot commands are unchanged.
