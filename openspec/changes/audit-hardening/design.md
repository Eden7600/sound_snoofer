# Design
Numbers match the audit documents.

## 16. Drawn configuration
A control's revision changes whenever any of its fields change, such as an error status, so pinning revisions would reject valid edits. The priority and output edit controls instead publish the audio configuration hash as their Value. `build()` records that hash, and the global `layoutKey` includes it, so a configuration change always rebuilds and the recorded hash always names the drawn lists. Priority and output edits send the current revision when the hash still matches, and revision 0 otherwise, which the host rejects as stale ("control changed; try again").

Deck region controls carry no list identity on the host (their revision changes only with availability), so pinning them gains nothing; they are left unchanged.

## 17. Bounded media calls
A helper waits for each WinRT async operation for up to 2 s. If the operation is still running, the helper cancels it and throws `HRESULT_FROM_WIN32(ERROR_TIMEOUT)`, which the existing handlers return. In a snapshot, a session whose metadata times out is reported without metadata, as other metadata failures already are. Synchronous property reads are unchanged.

## 18. Build staging
- `build.ps1` keeps graceful shutdown and deployed-file lock checks, then recreates its repository-owned `.local/stage`. Every selected companion, including Soundboard, writes there through `OutputDirectory`; models and licenses are staged too. Existing tag conditions stay unchanged. Only files produced by this build are published; excluded companions and unrelated `bin/` files are not removed.
- Run `npm run --silent build:gui` (esbuild and Tailwind), verify both generated assets, then run the existing Go command with `-o bin/snoofer.exe`. No alternate application output name or directory is introduced. Copy the fresh staged files into `bin/` only after the Go command succeeds, preserving nested model/license paths.
- The guarantee covers dependency preparation, native compilation, GUI generation and Go compilation/link failures before output deployment: existing deployed file contents stay unchanged. It is not a transactional deployment. On Windows, Go copies its linked executable into the requested final path; a disk/permission failure during that write or the subsequent companion copies can leave incomplete output. Report deployment failures and require a successful rebuild; do not claim rollback or print build success. Full rollback would require backups and restoration of every affected executable, companion and nested asset, plus removal of new targets, and restoration can fail under the same I/O fault; that deployment transaction is outside this compilation-failure fix.
- `build-neural-aec.ps1` extracts CMake into a fresh temporary folder and marks/moves it into the cache only after extraction completes. A directory left by an interrupted extraction, including one containing `cmake.exe`, is not sufficient evidence of completion; the next build re-extracts the verified archive.
- When the LocalVQE or GGML revision is absent or differs from the pinned one, fetch/check out the pinned LocalVQE revision and update submodules, then verify both revisions. Preserve the existing MSVC compatibility patch. Keep verified model downloads in the dependency cache and copy them into each fresh stage, so staging does not force model downloads on every build.
- Verify orchestration in an isolated temporary repository with fake compiler/package-manager/helper commands: native/asset/Go compile failures preserve deployed sentinels; success publishes the selected files and models/licenses; a later tagged build cannot publish stale staging files; deployment copy failure reports failure. Dependency setup checks cover an interrupted extraction and wrong/missing checkout revisions without network or hardware.

## 21. Clean stop
`SnooferStop` returns `timing` when `AudioCallbackStop` returns -2, which Voicemeeter documents as "no callback registered". Other non-zero results still mark the cleanup as uncertain.

## 22. Initial key state
Before its first read, a deck session asks the HID driver for the current input report (`HidD_GetInputReport`, report 1) and decodes it as the baseline. If the device refuses, behaviour is unchanged. A later identical report produces no events, so a deck that also reports on open is unaffected.

## 23. Second deck open errors
The surface remembers the last open error per HID path. It sends `Event{Device: "path:<path>", Error: ...}` when a path's error first appears or changes. When that path opens or disappears, it clears its dedupe record and sends an explicit retirement event for the temporary path identity. The plugin removes that path-only connection report; a successful session keeps its serial identity (or recreates the path identity if no serial is available). Remembered serial-based connection reports are preserved. Discovery-level errors keep their current event. Test both recovery and disappearance so an old failed path cannot leave a phantom disconnected card beside a working deck.

## 24. Light-write epoch
The light queue gets an epoch, incremented by `resetGroupDials`. `sendLight` captures it, and `lightWritten` ignores results from an older epoch. The worker generation check is unchanged.

## 25. Same-origin replacement
Firefox extension origins are random per install, so the extension ID cannot be pinned. Instead, a connection records its `Origin`. A newer hello for the same browser replaces the current connection only if the origins match; otherwise it is refused with a visible reason. An extension reload keeps its origin, so reconnecting still works.
