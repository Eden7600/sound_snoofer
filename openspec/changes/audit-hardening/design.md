# Design
Numbers match the audit documents.

## 16. Drawn configuration
A control's revision changes whenever any of its fields change, such as an error status, so pinning revisions would reject valid edits. The priority and output edit controls instead publish the audio configuration hash as their Value. `build()` records that hash, and the global `layoutKey` includes it, so a configuration change always rebuilds and the recorded hash always names the drawn lists. Priority and output edits send the current revision when the hash still matches, and revision 0 otherwise, which the host rejects as stale ("control changed; try again").

Deck region controls carry no list identity on the host (their revision changes only with availability), so pinning them gains nothing; they are left unchanged.

## 17. Bounded media calls
A helper waits for each WinRT async operation for up to 2 s. If the operation is still running, the helper cancels it and throws `HRESULT_FROM_WIN32(ERROR_TIMEOUT)`, which the existing handlers return. In a snapshot, a session whose metadata times out is reported without metadata, as other metadata failures already are. Synchronous property reads are unchanged. Use the shared helper for all async waits, including manager startup and artwork. Own the new MS handle until manager startup succeeds so timeout/failure releases it before apartment cleanup. A small native self-test covers completed, failed, cancelled and timed-out operations without a live player; compile the production helper with the canonical media companion build.

## 18. Build staging
- `build.ps1` keeps graceful shutdown and deployed-file lock checks, then recreates its repository-owned `.local/stage`. Every selected companion, including Soundboard, writes there through `OutputDirectory`; models and licenses are staged too. Existing tag conditions stay unchanged. Only files produced by this build are published; excluded companions and unrelated `bin/` files are not removed.
- Run `npm run --silent build:gui` (esbuild and Tailwind), verify both generated assets, then run the existing Go command with `-o bin/snoofer.exe`. No alternate application output name or directory is introduced. Copy the fresh staged files into `bin/` only after the Go command succeeds, preserving nested model/license paths.
- The guarantee covers dependency preparation, native compilation, GUI generation and Go compilation/link failures before output deployment: existing deployed file contents stay unchanged. It is not a transactional deployment. On Windows, Go copies its linked executable into the requested final path; a disk/permission failure during that write or the subsequent companion copies can leave incomplete output. Report deployment failures and require a successful rebuild; do not claim rollback or print build success. Full rollback would require backups and restoration of every affected executable, companion and nested asset, plus removal of new targets, and restoration can fail under the same I/O fault; that deployment transaction is outside this compilation-failure fix.
- `build-neural-aec.ps1` extracts CMake into a fresh temporary folder and marks/moves it into the cache only after extraction completes. A directory left by an interrupted extraction, including one containing `cmake.exe`, is not sufficient evidence of completion; the next build re-extracts the verified archive.
- When the LocalVQE or GGML revision is absent or differs from the pinned one, fetch/check out the pinned LocalVQE revision and update submodules, then verify both revisions. Preserve the existing MSVC compatibility patch. Keep verified model downloads in the dependency cache and copy them into each fresh stage, so staging does not force model downloads on every build.
- Verify orchestration in an isolated temporary repository with fake compiler/package-manager/helper commands: native/asset/Go compile failures preserve deployed sentinels; success publishes the selected files and models/licenses; a later tagged build cannot publish stale staging files; deployment copy failure reports failure. Dependency setup checks cover an interrupted extraction and wrong/missing checkout revisions without network or hardware.

## 21. Clean stop
`SnooferStop` returns `timing` when `AudioCallbackStop` returns -2, which Voicemeeter documents as "no callback registered". Other non-zero results still mark the cleanup as uncertain. Extend the existing callback probe runner with a native self-test of production SnooferStop using fake stop/unregister callbacks. Cover clean stop and -2 with unregister 0/1, other stop errors, unregister failure retaining ownership and repeated stop. No live callback registration is needed.

## 22. First input after connection
The initial current-report approach failed hardware acceptance: Plus and Plus XL returned a 512-byte buffer beginning `01 00 00 00`, with no key payload, including while an XL key was held. It cannot provide a baseline. The user explicitly chose that a displayed button must respond to its first press, accepting that a button held while connecting may activate once when first reported.

Use the Decoder's zero-valued key and dial-button arrays as released state from construction. Remove the readiness flags and first-report suppression; a valid first down report emits one press, repeated down reports emit none, and release followed by another down emits the next press. Invalid or truncated reports remain rejected without changing state. Preserve key release events, signed encoder deltas, independent state per device, stale frame-generation checks and dark/inert behavior. A reconnect creates a fresh released state and never replays queued commands from the old session. No timer, startup drain or idle-state inference is introduced.

Remove the synchronous `HidD_GetInputReport` call from the production device, `hidDeck` interface and session startup. The historical opt-in probe may retain its own test-local query helper and report decoding results; it remains skipped by default. Remove the obsolete fake baseline field/method. All production input continues through the existing cancellable normal-read path; unrelated I/O cleanup is outside this checkpoint.

Before the fix, prove focused regressions fail for the first key and first dial-button down on both Plus and Plus XL. Cover duplicate down, release/repress, malformed and zero-length reports before first input, plus fresh reconnect state. Replace the old tests that deliberately swallowed the first press, and keep the existing plugin hold, stale-generation and multi-deck tests. Run focused tests, full internal/streamdeck and plugins/streamdeck tests, vet and strict OpenSpec validation. Root owns the canonical build and physical first-key/dial-button acceptance; automated results do not complete hardware acceptance.

## 23. Second deck open errors
The surface remembers the last open error per HID path. It sends `Event{Device: "path:<path>", Error: ...}` when a path's error first appears or changes. When that path opens or disappears, it clears its dedupe record and sends an explicit retirement event for the temporary path identity. The plugin removes that path-only connection report; a successful session keeps its serial identity (or recreates the path identity if no serial is available). Remembered serial-based connection reports are preserved. Discovery-level errors keep their current event. Test both recovery and disappearance so an old failed path cannot leave a phantom disconnected card beside a working deck.

## 24. Light-write epoch
The light queue gets an epoch, incremented by `resetGroupDials`. `sendLight` captures it, and `lightWritten` ignores results from an older epoch. The worker generation check is unchanged.

## 25. Same-origin replacement
Firefox extension origins are random per install, so the extension ID cannot be pinned. Instead, a connection records its `Origin`. A newer hello for the same browser replaces the current connection only if the origins match; otherwise it is refused with a visible reason. An extension reload keeps its origin, so reconnecting still works.

## Observed shutdown deadline diagnostic
During the 2026-10-15 canonical-build attempt, production host PID 70268 exceeded
the graceful-stop wait. Its owned error dialog said only `context deadline exceeded`;
the process exited after the user dismissed the dialog. No force termination was
used. The recently connected Plus and Plus XL are context, not a confirmed cause.

`app.stopHost` already supplies a five-second cleanup context. `Host.Stop` cancels
the workers and awaits each instance in reverse dependency order. Instance-returned
errors include the plugin ID, but the host's own `ctx.Done()` branch appends only
`ctx.Err()`, losing the currently awaited ID. Wrap that branch with the same plugin
context using `%w` before joining earlier failures. The message identifies the
awaited cleanup, not a proven internal deadlock or hardware cause.

Extend the existing deterministic `TestStopDeadline`: hold a fake instance until
the test releases it, expire the stop context, and require both the plugin ID and
`errors.Is(err, context.DeadlineExceeded)`. Also retain an earlier sentinel cleanup
error through `errors.Join`. Release every fake after the assertion. Keep existing
successful shutdown and reverse-order tests. No native calls or process manipulation
are needed for these regressions.

This checkpoint changes diagnostic context only: no timeout increase, skipped join,
force termination, error-dialog change, HID change or automatic restart. After the
source checkpoint, root coordinates the canonical build and graceful-stop recheck
with the previous launch configuration. Record a successful stop or the named
failure; investigate any reproduced plugin problem separately before claiming its
cause or repair. The initial failed stop remains failed evidence.

## Initial-report diagnostic probe
Before the first-input repair, the session ignored current-report query and decode failures; its fallback intentionally consumed the first valid key report. The confirmed Plus XL first-press failure requires distinguishing refused, malformed and valid initial reports. Add a separate SNOOFER_DECK_BASELINE_PROBE opt-in test that discovers and opens supported decks, logs the current-input result and bounded raw report prefix, and validates a fresh Decoder against successful reports. It issues no image, feature, key or media writes and never runs in default tests. Root runs it with the production host gracefully stopped and a disposable go test timeout, then restores the same host configuration. A hung native query may time out that isolated test process; never terminate the production audio host. The probe is evidence only and does not change runtime behavior. Specify any resulting repair before implementing it.

## Opt-in second-deck refusal acceptance
Exercise the existing per-path open-error behavior against two real decks using a
temporary external holder, without changing production code or default tests. The
helper lives in the coordinating chat's scratch directory and runs only on explicit
invocation with a selected serial. Use existing HID discovery/model/serial metadata
helpers, through a scratch Go test overlay if convenient. Resolve exactly one known
deck serial to its verified supported model and exact HID path; ambiguous or missing
matches abort. Close discovery handles before acquiring the hold. Do not claim that
holding an arbitrary collection represents refusal of the intended deck.

Root records the canonical host launch configuration, stops that host normally with
scripts/stop.ps1 and waits for successful exit. The holder opens only the verified
path with GENERIC_READ, sharing mode 0 and OPEN_EXISTING. Its non-inheritable handle
issues no input reads, output writes, feature reports or current-input queries.
Require an immediate second open using production access/share flags to fail with
Win32 ERROR_SHARING_VIOLATION (32); close that second handle if it unexpectedly opens
and report unsuccessful setup. A metadata-only access-0 handle is not sufficient
proof of exclusivity. Failure to acquire the hold aborts without closing unrelated
applications, disabling devices or broadening the target.

The holder has a checked explicit release signal and a hard lifetime of at most 90
seconds from acquisition. It owns the handle through defer/finally and closes it on
release, deadline and every error path. Log acquisition, verified refusal and release;
the coordinating shell must remain free to restore the host and signal release. Use
no unbounded interactive wait or inherited handle. This is an opt-in acceptance tool,
not a permanent hook, alternate application build or change to HID sharing policy.

While the hold is active, root starts the existing canonical executable with the same
configuration. Verify that the selected path's connection error appears while the
other serial remains connected and responds to a known non-destructive physical
control. Then release the holder, confirm its clean exit and observe normal rescan
recovery: the selected serial connects, its temporary path error disappears and the
other deck remains usable. Cleanup always releases the hold and restores/verifies
the prior host configuration; never force-terminate the audio host. If the deadline
expires before the refusal and unaffected-deck observations, record incomplete
acceptance instead of inferring success. Record real sharing-error, GUI, physical
input and recovery evidence separately. This completes only the refused-second-deck
component of task 11; frozen-player acceptance remains separate.
