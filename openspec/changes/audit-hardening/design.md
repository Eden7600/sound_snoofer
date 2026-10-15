# Design
Numbers match the audit documents.

## 16. Drawn revisions
The Routing and Stream Deck screens rebuild whenever the priority lists, output slots or deck regions change (`layoutKey`). `build()` therefore records the revision of `audio.priority-edit`, `audio.output-edit`, `streamdeck.region-clip` and `streamdeck.region-remove`, and index-based edits send that recorded revision. An edit drawn from older lists is then rejected as stale instead of changing whatever now sits at that index.

## 17. Bounded media calls
A helper waits for each WinRT async operation for up to 2 s. If the operation is still running, the helper cancels it and throws `HRESULT_FROM_WIN32(ERROR_TIMEOUT)`, which the existing handlers return. In a snapshot, a session whose metadata times out is reported without metadata, as other metadata failures already are. Synchronous property reads are unchanged.

## 18. Build staging
- `build.ps1` builds companions into `.local/stage` (models and licenses included) and copies the staged files into `bin/` only after CSS generation and `go build` succeed.
- `build-neural-aec.ps1` extracts CMake into a temporary folder and moves it into place only when complete.
- When the LocalVQE or GGML revision differs from the pinned one, it fetches, checks out the pinned revision and updates submodules, instead of throwing.

## 21. Clean stop
`SnooferStop` returns `timing` when `AudioCallbackStop` returns -2, which Voicemeeter documents as "no callback registered". Other non-zero results still mark the cleanup as uncertain.

## 22. Initial key state
Before its first read, a deck session asks the HID driver for the current input report (`HidD_GetInputReport`, report 1) and decodes it as the baseline. If the device refuses, behaviour is unchanged. A later identical report produces no events, so a deck that also reports on open is unaffected.

## 23. Second deck open errors
The surface remembers the last open error per HID path. It sends `Event{Device: "path:<path>", Error: ...}` when a path's error first appears or changes, and clears the record when the path disappears or opens. Discovery-level errors keep their current event.

## 24. Light-write epoch
The light queue gets an epoch, incremented by `resetGroupDials`. `sendLight` captures it, and `lightWritten` ignores results from an older epoch. The worker generation check is unchanged.

## 25. Same-origin replacement
Firefox extension origins are random per install, so the extension ID cannot be pinned. Instead, a connection records its `Origin`. A newer hello for the same browser replaces the current connection only if the origins match; otherwise it is refused with a visible reason. An extension reload keeps its origin, so reconnecting still works.
