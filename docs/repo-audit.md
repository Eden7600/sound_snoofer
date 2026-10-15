# Repository audit (2026-10-14)

Whole-repository audit at commit `237a8ee`. It complements
[gui-audit.md](gui-audit.md); items already recorded there are not repeated.

**Assumed load:** one person on one PC with one Snoofer process, real audio
hardware, and devices that come and go.

**Original baseline at `237a8ee`:** `go vet ./...` was clean and `go test ./...`
passed. `go test -race` was not run on that cgo-disabled toolchain; the later full
race result is recorded below.

Findings are numbered so they can be referenced individually.

## Current status (2026-10-15)

Reconciled with the completed source work and final `audit-completion` evidence.
The numbered findings and source locations below retain the original
audit wording; they are historical evidence, not a list of current defects.

| Findings | Current disposition | Implementation / evidence |
| --- | --- | --- |
| 1–13 | Fixed, each with a regression | [audit-reliability-fixes](../openspec/changes/audit-reliability-fixes/tasks.md) |
| 14 | Fixed; live artwork removal/reappearance and clock/reopen checks pass | [view-state-transport](../openspec/changes/view-state-transport/tasks.md) |
| 15 | Fixed through correlated per-action outcomes and independent reservations | [gui-action-outcomes](../openspec/changes/gui-action-outcomes/tasks.md) |
| 16 | Fixed; priority/output edits retain their drawn list identity, with stale-list regression coverage | [audit-hardening](../openspec/changes/audit-hardening/tasks.md) |
| 17 | Bounded native media waits implemented and tested; live Snoofer kept receiving healthy title updates while a verified owned player was frozen, then recovered that player's title | [audit-hardening](../openspec/changes/audit-hardening/tasks.md) |
| 18 | Staged companion builds and interrupted dependency setup recovery implemented and tested | [audit-hardening](../openspec/changes/audit-hardening/tasks.md) |
| 19 | Complete: `49ea955` removes unused acknowledgement/deduplication and batch paths, preserving callback outcomes, origin notice clearing, single-edit safety and meaningful tests | [audit-completion](../openspec/changes/audit-completion/tasks.md) |
| 20 | Complete: shared decoding, journals, bus parsing, fallback and shutdown helpers; `9eb59e4` shares thumbnail fit/encode while retaining caller sizing/filtering/GIF policies, with input-limit coverage in `b002e6d` | [audit-completion](../openspec/changes/audit-completion/tasks.md) |

The user explicitly requested completing the original recommendations, superseding
the former retained work in findings 19, 20, 25 and 29. Mini, MK.2 and original XL
support stays as-is by the user's instruction; their unavailable physical acceptance
is waived, not passed. Refused-second-deck acceptance is blocked by incompatible
device sharing; a real uncertain one-shot/no-replay outcome remains unverified after
bounded attempts. The final GUI Pause reported Processed, with exactly one command
and one effect after resume. Frozen-player
acceptance passes: healthy title counters advanced in live Snoofer during the verified
freeze, the frozen session recovered after resume, and both owned players cleaned up.
The original anonymous shutdown timeout remains unexplained; later coordinated
graceful stops pass. See [GUI acceptance](gui-audit.md#current-status) for evidence.

Latest checks: full Go tests, vet and the full race suite pass using the approved,
verified portable compiler. Strict frontend gates, 297 browser tests in 42.2 seconds
and all 106 OpenSpec changes pass. The initial Stream Deck fixture hang, Now Playing
fixture race and Hue readiness failure were repaired in `07b616f` and `ad94e6e`;
production behavior was unchanged by those test repairs.
The canonical build and native async/identity/callback self-tests pass; no production
source changed afterward. Build-time host PID 73604 was later restarted as PID 70128 with
the same `--config bin/snoofer.json` command; final controls PID is 11816. Persisted
audio-state and user-owned smoke-script fingerprints are unchanged. Owned helper processes exited;
sessions disappeared and silent WAVs were removed, the test notice is dismissed, and native Media
shows only the user's Brave cards. Computer control was released at 20:05Z.
No further live attempts are planned; refused-deck and uncertain-one-shot acceptance
remain explicitly unverified.

The companion [smaller audit](repo-audit-minor.md#current-status-2026-10-15) records
the completed polling-content fix, shared OnOff helper and rejection of every second
live browser-name claimant. Source completion, validation and physical acceptance
remain separate.

## Must fix

1. **Stream Deck editor crashes for decks without dials**
   (`plugins/streamdeck/editor_view.go:113-115`, `app/web/app.js:908,924`)
   - **Problem:** On a model with no usable dials (Mini, MK.2, XL), `view.Dials`
     is never set, so the page receives `null`. It then calls `.length` on it and
     stops halfway through building the screen. The GUI fixture
     (`scripts/check-gui.cjs:418`) uses `[]`, so the tests miss it.
   - **Fix:** `view.Dials = make([]EditorSlot, 0, sh.dials)` (same for `Keys`),
     plus a fixture case with `Dials: null`.
   - **If skipped:** The deck editor is unusable on three of the five models.

2. **The "device no longer connected" check never runs**
   (`internal/control/worker.go:760-772`, `plugins/audio/audio.go:518-524`)
   - **Problem:** The check only fires for edits to the rows named `source` or
     `output`. The plugin always sends `normal-source`, `normal-output` or
     `vr-profile-*` instead. `selection_test.go` sends the bare names, so it
     passes.
   - **Fix:** Compare on the name without its `normal-`/`vr-profile-` prefix,
     check VR rows against `VRSourceOptions`/`VROutputOptions`, and make the test
     send `normal-output`.
   - **If skipped:** A device unplugged a moment earlier can be saved, and then
     sits as "unresolved".

3. **Soundboard stays "pending" forever when an output slot takes the soundboard**
   (`plugins/audio/soundboard.go:83-95`, `internal/routing/voice.go:186-188`)
   - **Problem:** One rule (which buses `Strip[7]` feeds) is written twice. The
     planner sets `Strip[7].<slot bus>=1` for output slots that receive the
     soundboard. `soundboardReady` demands every A bus except Playback be 0, so
     it never passes.
   - **Fix:** Have `soundboardReady` compare the plan's own `Strip[7].*`
     operations against the snapshot, instead of rebuilding the rule.
   - **If skipped:** Soundboard volume and readiness stay unavailable whenever an
     output slot carries the soundboard.

4. **A failed callback registration is never retried**
   (`internal/voicemeeter/callback_windows.go:33-35`)
   - **Problem:** Every failure path sets `m.enabled` but leaves `m.active`
     false. The next call with the same arguments returns the cached error and
     never tries again, and `RestartCallback` requires `m.active`. A typical
     case: Snoofer starts at login before Voicemeeter is running.
   - **Fix:** Short-circuit only when the hook is actually running:
     `enable == m.enabled && sameHook(...) && (m.active || !enable)`.
   - **If skipped:** Stall detection and the echo-cancellation insert stay dead
     until the user toggles Live, even with Voicemeeter healthy.

5. **Stream Deck drops a key report that arrives right at the 1 s read timeout**
   (`internal/streamdeck/device_windows.go:124-139`)
   - **Problem:** `device.io` checks "time is up" before "data arrived", so a
     completed read gets cancelled and its report thrown away. If that report
     was a key release, the key stays held, and after 500 ms a hold (or Home)
     fires that the user never did. The next press of that key is lost too.
   - **Fix:** Test `status == WAIT_OBJECT_0` first. After `CancelIoEx`, return
     the bytes if `GetOverlappedResult` reports any. Only writes need the
     deadline.
   - **If skipped:** Rare but real phantom hold actions.

6. **Any web page can knock the browser bridge offline**
   (`plugins/nowplaying/bridge.go:237-239`, `extension/src/bridge.js:21-29`,
   `extension/src/logic.js:34-61`)
   - **Problem:** A page can dispatch a `snoofer-media:report` with wrong types,
     such as `"title":{}`. Go then closes the whole connection, and the extension
     resends the same report on reconnect, so it loops. The page can also set
     `artwork` to any URL, and the extension fetches it with the user's cookies.
   - **Fix:** In Go, `continue` past a malformed message instead of `return`. In
     `sessions()`, coerce field types. Only fetch `http(s)` artwork, with
     `credentials: "omit"`.
   - **If skipped:** One hostile or buggy page hides every tab of that browser,
     and pages get a free logged-in GET request.

7. **The transition gate misses interface mics in positions 3-5**
   (`internal/routing/transition.go:29-38`)
   - **Problem:** The patch scan covers only `Patch.asio[0..3]`, while
     `patchCells(n)` produces `Patch.asio[2n]`/`[2n+1]` for up to five mics.
     Example: mics `[webcam, usb, desk-interface]`. Desk is strip 2 on
     `Patch.asio[4..5]`, which is never gated, so its B2/B3 sends and its Element
     return stay live through the switch.
   - **Fix:** Derive the strip as `k/2` for any patch index `k`. On an A1 change,
     gate every interface-mic strip. Add a test with an index of 4 or higher.
   - **If skipped:** Possible pops or bursts on stream or recording during
     hardware switches.

8. **Pausing a live recording can be blocked by a saved-choices write**
   (`internal/control/worker.go:335-346`)
   - **Problem:** `runTape` always saves `TapeRoutingManaged` before
     `TapePlayPause`, the command that also pauses or resumes a recording. If the
     state file was changed externally or `StateError` is set, the save fails and
     the pause never happens.
   - **Fix:** Save only when the command will start tape playback (recorder
     stopped and not paused).
   - **If skipped:** The recording can't be paused from Snoofer at exactly the
     moment something is wrong.

9. **A settings edit shows "saved" even when the worker rejects it**
   (`plugins/audio/priorities.go:250-303`)
   - **Problem:** `applyEdit` saves the settings and replaces `i.running` before
     the worker reloads and validates them. Example: the user removes the mic the
     saved choices still use. The UI shows the new list, the worker keeps the old
     config, and the next launch starts with `StateError`.
   - **Fix:** Run `config.LoadChoices` on the new config before saving, and
     refuse the edit if it reports `StateError`.
   - **If skipped:** The screen and the actual routing disagree, and the broken
     config is on disk.

10. **One exception in a GUI updater stops polling for good**
    (`app/web/app.js:1253-1260`, `953-954`)
    - **Problem:** If `update()` throws inside the `catch`, `poll` rejects before
      it reschedules itself. The slot updater reads `c(slot.Control)` before its
      `!slot` guard, which is one way to throw (a preview that shrinks while a
      deck field is focused).
    - **Fix:** Reschedule in `finally`, and move the `!slot` check before
      `c(slot.Control)`.
    - **If skipped:** The controls window freezes with stale state until it is
      reopened.

11. **Hue reconnects about once a second forever, and re-pairing leaks a stream**
    (`plugins/hue/plugin.go:290,365-367,377-388`, `plugins/hue/sync.go:126`,
    `plugins/hue/client.go:180`)
    - **Problem:** The retry delay resets to 0 as soon as the resources load,
      before the event stream works. A stream that fails at once (403, 503, or
      immediate close) therefore retries every 1 s, each time with a GET of up to
      16 MB. Separately, `reconnect` only closes idle connections, so every Pair
      press leaves the old stream connection and goroutine running.
    - **Fix:** Reset the delay only after the stream delivers its first message
      (same for `w.sync.retryDelay`). Give each connection its own
      `context.WithCancel` and cancel it in `closeClient`.
    - **If skipped:** Constant load on the bridge, and connections pile up toward
      its stream limit.

12. **Overlapping Discord token refreshes can erase the saved token**
    (`plugins/discord/plugin.go:378-386,516-558`)
    - **Problem:** Each READY starts a refresh-token exchange, even while another
      is running. If the pipe drops and reconnects mid-exchange, the losing
      exchange gets `invalid_grant` and calls `saveRefresh("")` over the good
      token. This assumes Discord invalidates a refresh token once it is used.
    - **Fix:** Don't start an exchange while one is running, and ignore
      `invalid_grant` from an older generation.
    - **If skipped:** The user has to approve Snoofer in Discord again at random
      times.

13. **Now Playing reports stopped before its listener closes, and never retries a busy port**
    (`plugins/nowplaying/plugin.go:165-178`)
    - **Problem:** `defer w.bridge.close()` is registered first, so it runs after
      `close(i.done)`, and `Stop` returns while the listener is still open. On the
      GUI's confirm-restart, the new process can find port 47815 busy, and
      `listen` is tried only at Start.
    - **Fix:** Register `defer close(i.done)` first. In `step`, retry `listen` on
      the retry timer while `w.bridge == nil`.
    - **If skipped:** After a restart, browser tabs can vanish until the app is
      restarted again.

## Should fix

14. **Full view state, artwork included, is re-sent several times a second**
    (`app/tray_windows.go:200-255`, `app/web/app.js:1256-1259`)
    - **Problem:** Every 200 ms the tray re-reads `snoofer.json` and the
      registry, then encodes and pipes the full `ViewState` even when nothing
      changed. That includes every control's base64 `Artwork`, up to 128 KiB
      each. The page then pulls it again through Wails IPC every 100 ms. 60
      soundboard clips is about 3.6 MB per frame.
    - **Fix:** Send a frame only when the snapshot changes, for example by
      comparing registry revisions. Send artwork once, the way animations
      already are.
    - **If skipped:** Steady CPU and memory churn whenever the window is open.

15. **A repeated error is invisible after it is dismissed, and controls lock for 2.5 s**
    (`app/tray_windows.go:153,265`, `app/web/app.js:65,1214`)
    - **Problem:** `notice` is never cleared on success, and the page hides a
      notice that matches the dismissed text. When the same error happens again,
      nothing shows. "Sending" also blocks every control until its 2.5 s timeout.
    - **Fix:** Clear `notice` in Go when a dispatch succeeds, or add a sequence
      number and match on that.
    - **If skipped:** Failures look like the app is ignoring clicks.

16. **Index-based edits send the click-time revision**
    (`app/web/app.js:216,400,1088-1089`)
    - **Problem:** `priorityEdit`, `outputEdit` and region clip/remove read the
      revision when clicked, not when the row was drawn. Go ties the
      `audio.priority-edit` revision to the list contents
      (`plugins/audio/priorities.go:224`), so this bypasses the stale-index check.
    - **Fix:** Capture the revision when the row is built and pass it to
      `request`, as `editRevision` already does elsewhere.
    - **If skipped:** Rarely, the wrong priority or output entry gets changed.

17. **WinRT media calls have no timeout**
    (`internal/mediasessions/native/media.cpp:153,213,219,226,242-247`)
    - **Problem:** `MSSnapshot`, `MSArt` and `MSCommand` block on `.get()` for
      cross-process calls into each media app. One hung player blocks the
      locked Now Playing goroutine forever. On shutdown, the goroutine and the
      helper handle leak.
    - **Fix:** `wait_for(2s)` before `get()`; on timeout, `Cancel()` and skip the
      session.
    - **If skipped:** One frozen media app freezes Now Playing until Snoofer
      restarts.

18. **Build leaves a mixed `bin/` on failure, and an interrupted neural-AEC fetch breaks later builds**
    (`scripts/build.ps1:24-36`, `scripts/build-neural-aec.ps1:19-31`)
    - **Problem:** The native DLLs are written into `bin/` before `npm run css`
      and `go build`. If either fails, new DLLs sit beside the old exe. Separately,
      a failed checkout or submodule update leaves `.git` behind, so later runs
      skip setup and then throw on the revision check. An interrupted
      `Expand-Archive` is never re-extracted either.
    - **Fix:** Build the DLLs into `.local/` and copy them to `bin/` after Go
      succeeds. On a revision mismatch, re-run checkout and submodule update
      instead of throwing.
    - **If skipped:** An exe can launch with DLLs from a different build, and
      recovery needs a manual folder delete.

19. **Dead protocol and write-only state in the control worker**
    (`internal/control/worker.go:94-97,241,397-398,444-469,595-611,716-720,810-822`,
    `internal/control/edits.go:141-172`, `internal/control/feedback.go:14-21`,
    `internal/control/semantics.go:12-78`, `internal/control/notice.go:23-35`)
    - **Problem:** Several pieces are left over from the removed TUI and are
      never set or read by production code (the whole tree was grepped, JS
      included):
      - The ack/dedupe protocol: `ID`, `Acks`, `EditAck`, `EditError`.
      - Batch `Action.Edits`: `SettingEdit`, `maxDeferredEdits`.
      - The `VRMic`, `VRPlayback`, `VR*Available`, `PublishedAt` and
        `ConfigPath` state fields. Filling them also makes every publish run
        `VRDevices()` for nothing.
      - Test-only helpers: `DependsOn`, `MicStrips`, `NoticeAt` and
        `NeedsAttention`, which nothing calls at all.
    - **Fix:** Delete it, along with the tests that only cover it.
    - **If skipped:** About 185 lines to maintain for nothing.

20. **Same helpers copied in several places**
    - **Where:**
      - Strict JSON decoding exists three times: `internal/config/config.go:83-97`,
        `internal/config/state.go:43-58` and `storage.Decode`. The shims in
        `internal/config/journal.go` only forward to `storage`.
      - The A1-A5 bus-index parse exists three times:
        `internal/controller/mixer.go:177`, `internal/routing/mute.go:28` and
        `plugins/audio/echo.go:79-90`.
      - `firstNonEmpty` exists four times, and the standard library's `cmp.Or`
        already does it: `plugins/hue/plugin.go:947`, `plugins/aec/plugin.go:536`,
        `plugins/appaudio/plugin.go:546` and `plugins/audio/report.go:248`.
      - The plugin `instance{cancel, done}.Stop` pattern is copied five times:
        media, vr, nowplaying, discord and appaudio.
      - Thumbnail encoding exists twice: `plugins/nowplaying/art.go:18-41` and
        `plugins/soundboard/artwork.go`.
    - **Problem:** Each set must change together, and the bus parse is exactly
      where a mismatch would mis-route audio.
    - **Fix:** Keep `storage.Decode`, keep one bus parser, use `cmp.Or`, and add
      one shared `Stop` helper in `snoofer/`.
    - **If skipped:** About 170 duplicate lines, and drift between the copies.

## Smaller findings

Thirteen lower-priority findings (21–33) are in
[repo-audit-minor.md](repo-audit-minor.md).

## Original audit summary

The following verdict and coverage notes describe the 2026-10-14 audit baseline.

**Verdict:** Fix 1-4 first. Each is a short, contained fix that makes a core
feature work as intended. Then fix 5-7, which give wrong behaviour on real
hardware.

**Lean:** about -380 lines and no dependency removals possible. All six direct
dependencies are used.

**Not checked:**
- Findings 1-6 and 13 were re-read directly at the cited lines. The rest come
  from reviewer reports and were spot-read only.
- Findings 7 and 12 rest on assumptions that were not observed: the mic layout,
  and how Discord rotates tokens.
- Nothing was run on hardware, and the race detector was not run.
- Not reviewed: `tools/` probes, `third_party/`, `scripts/check-desktop.cjs`
  and `scripts/check-extension.cjs`.
