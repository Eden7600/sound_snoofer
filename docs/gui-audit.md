# GUI audit

Audit of the desktop GUI (`app/web`) against [ui-contract.md](ui-contract.md), dated 2026-10-14.
It covers every screen, navigation, layout and the GUI test harness. It is a planning input, not a
specification: each fix below goes through its own OpenSpec change before implementation.

**Baseline:** `node scripts/check-gui.cjs` passes in about 27 s, and `node --test app/web/model.test.mjs` passes.
Screenshots were reviewed at 1280×820 and 800×600 using the browser fixture.

## Current status

As of 2026-10-15, the original GUI implementation checkpoints and their shared source checks
are complete. The later repository-completion changes pass aggregate validation and build;
the remaining physical acceptance is listed separately below.
Interactive WebView2 acceptance resumed after Windows was unlocked. The isolated canonical
fixture verified keyboard commitment, state restoration, target-bound dialogs and local action
feedback. Functional native state-preservation acceptance is fulfilled.

Native inspection follow-ups corrected compact branding (9065ff7), first key/dial-button
presses after reconnect (03e74d2), and shared-notice Dismiss alignment (fa08c20). The user
confirmed first presses on both Plus and Plus XL. Named shutdown diagnostics (c3cff86),
the coordinated graceful-stop recheck and unchanged native smoke pass; the original anonymous
shutdown timeout remains unexplained.

Compact/wide content, plugin gates, marker navigation and every native dropdown path pass.
Live media artwork changes, selective removal, clocks and controls reopen pass. Native rejected
State-call injection now verifies connection feedback, bounded backoff, refusal and recovery
without replay. The Dismiss fix passes 295 browser tests and strict frontend checks.
Packaged short/wrapped notice alignment now passes at both normal and minimum window sizes.
The normal canonical build passed; production retains the same --config bin/snoofer.json path.
Rapid independent mute pairs, real recording Pause/resume/Stop, callback recovery after
Voicemeeter process restart and startup before Voicemeeter now pass. A captured stale Mic Stage
choice was explicitly rejected, and a real asynchronous native Soundboard failure reached its
originating action. The user accepted this quality improvement and requested completion of
the existing repository audits, superseding their former retained work. Mini, MK.2 and
original XL support stays as-is by explicit instruction; unavailable physical acceptance
is waived, not passed. Refused-second-deck acceptance is blocked by incompatible device
sharing: both access-zero sharing modes failed acquisition with error 32, so no holder
was acquired and no further modes will be tried. The canonical host resumed its same
configuration as PID 70128. Native UI recovery shows both decks Connected/Display On,
callback Active (Buffers 14239), Recorder Stopped and Windows defaults Verified;
Media shows only the user's Brave cards after owned cleanup. This is UI recovery,
not a new physical press confirmation. Frozen-player acceptance now passes: live healthy
title counters advanced from 29 to 33 inside the verified 15-second freeze, and the
frozen player's title returned after resume. No command was sent in that run. A final
separate run sent one GUI Pause during a verified freeze: Snoofer reported Processed
and the resumed player logged one command and one effect without replay. The snapshots
captured Sending then Processed; they did not establish native timeout or correlated
uncertainty. The final owned players exited normally, their sessions and generated WAVs were removed,
and no further live attempts are planned. Final host/controls PIDs are 70128/11816
with the same configuration and unchanged audio-state/native-smoke fingerprints;
the test notice is dismissed and Media shows only user Brave cards. Computer control
was released at 20:05Z.

The findings below retain their original audit wording and old source references;
current implementation lives in `app/web/src`.

| Work / findings | Source status | Remaining acceptance |
| --- | --- | --- |
| `gui-test-harness`, `gui-typescript-modules` | Complete; generated Audio/Deck fixtures, independent Playwright cases, strict TypeScript/esbuild, keyboard/axe/geometry checks | None for source/tooling |
| `gui-content-fixes`: 7–9, 15–16 | Complete; corrected content, native smoke and affected-screen inspection at both sizes pass | None for content presentation |
| `gui-dropdown-commit`: 2 | Complete; native closed-select/popup keyboard commitment, cancellation and exactly-once pointer selection pass | None for native dropdown acceptance |
| `gui-connection-recovery`: 5 | Source and native rejected-State fault checks pass: separate banner, bounded backoff, recovery and no replay | None for packaged recovery acceptance |
| `gui-layout-polish`: 10–13 and meter/search parts of 14 | Affected screens, compact branding, navigation and native short/wrapped notice alignment pass | None for packaged layout acceptance; physical checks remain separate |
| `gui-state-preservation`: 3, 6 | Complete; functional native restoration, dialog cancellation, polling and captured-target invalidation verified | No remaining functional native journey; aggregate validation and physical integration remain separately tracked |
| `gui-action-outcomes`: 1, 4 | Source/native checks pass; rapid mutes, captured stale Mic Stage rejection, asynchronous native Soundboard failure and one Processed GUI Pause with one effect/no replay are verified | Real uncertain native outcome remains unverified; queued-target replacement remains unclaimed native evidence |
| `gui-attention-journeys`: remaining 14 | Complete; disabled/starting/failed Audio gates, marker-to-reason keyboard navigation, Plugins link/Retry availability and captured host-proposal cancellation verified natively | None for packaged presentation; real plugin lifecycle effects remain unclaimed |
| `view-state-transport` | Complete; source/native smoke and live media artwork change, selective removal, clocks and controls reopen pass | None for the transport artwork acceptance |
| `output-slot-detail` | Provider/renderer/browser checks, packaged device caption/current selection, and physical No device/In use/Disconnected/reconnect cases pass | None for specified packaged/physical output presentation |
| Repository hardening/cleanup | Findings 19, 20, 25 and 29 implemented in `49ea955`, `9eb59e4`/`b002e6d` and `f1c10fc` under `audit-completion`; full Go/vet/race, frontend/browser/spec gates, canonical build and native self-tests pass; live healthy title updates continue during an owned-player freeze | Refused second Deck blocked by incompatible device sharing; real uncertain native outcome unverified; no-dial-model physical acceptance waived, not passed; original shutdown timeout unexplained |

### Shared source and build validation

- Current `audit-completion` checks: full Go tests/vet and
  `go test -race -p 4 ./... -timeout 90s` pass using the approved, verified portable
  compiler. Strict frontend gates, **297 browser tests in 42.2 s** and all
  **106 OpenSpec changes** pass. The initial Stream Deck fixture hang, Now Playing
  fixture race and Hue readiness failure were repaired in `07b616f` and `ad94e6e`,
  with repeated race checks; production behavior was unchanged by those repairs.
  The canonical build and native async/identity/callback self-tests pass; no production
  source changed after the build. The build-time restore was PID 73604; final host PID
  70128 retains `--config bin/snoofer.json` with unchanged persisted audio-state and
  user-owned smoke-script fingerprints. Refused-deck acceptance is blocked and real
  uncertain-native-outcome acceptance is unverified; Mini/MK.2/original XL checks
  are explicitly waived by the user, not passed.

- Follow-up reconciliation of the existing repository audits found minor item 26
  still present. `a5ca385` fixes App Audio detail selection during polling and
  Media icon replacement, with both regressions failing first. Strict frontend
  gates, 15 model tests, **297 browser tests in 44.2 s**, both affected screens at
  normal/minimum sizes, all 105 OpenSpec changes and the canonical build pass.
  The tray host resumed its prior configuration; no new desktop session or
  physical acceptance was claimed.

- Before the native follow-ups, ordinary `scripts/check.ps1` at `49e3fd9` passed full Go tests and
  vet, strict TypeScript, ESLint, Prettier, 15 model tests, **290 browser tests in 43.4 s**, and
  strict validation of all **104 OpenSpec changes**.
- The default browser concurrency is four workers. Automatic sixteen-worker execution on this
  machine caused widespread timeouts; the corrected default meets the existing 60-second budget
  without extending test timeouts or weakening assertions. That run also exposed the fixed
  shared-notice focus defect. The initial failed run is retained separately in the local logs.
- After the compact-brand fix, the complete browser suite passed **293 tests in 44.1 s**, with
  strict frontend gates passing. The first compact packaged inspection was interrupted by physical
  Escape; that fixture exited normally via EOF without actions. The resumed outer 800×600 native
  inspection now confirms full branding and keyboard access to the final navigation items.
- Compact and wide canonical fixtures verified Plugins and Third-Party Apps, including headings,
  labels and cards at the top and bottom; both closed via EOF with exit code 0 and zero actions.
- The later canonical diagnostic build verified recovery using a temporary Desktop.State rejection
  wrapper in its isolated WebView2 fixture. The banner appeared beside the retained notice;
  Plugins actions were disabled, navigation to Audio worked, and Record reported
  "Host disconnected; action not sent" with zero captured actions. Failure intervals grew through
  201, 414, 804, 1605 and 3206 ms, then remained about five seconds (5004–5016 ms).
  After the wrapper resumed State, the banner cleared while the notice and refusal remained,
  meters resumed and polling returned to 106–122 ms. The action log remained empty: no replay.
  The fixture closed via EOF with exit code 0. This is native binding-fault evidence; EOF itself
  only establishes cleanup. The earlier DevTools approval timeout was superseded by this check.
- Shared-notice alignment regressions failed at both widths with the old float's 6px offset.
  The centered layout passes short/wrapped checks, 16 focused notice/focus/recovery tests and
  **295 browser tests in 42.1 s**, plus TypeScript, ESLint and Prettier.
  The resumed canonical WebView2 check confirms centered, contained Dismiss buttons without
  overlap for short errors and wrapped 180-digit messages at captured 1266×813 and 786×593
  sizes (approximately 1280×820 and 800×600 outer windows). This verifies alignment, not a
  dismissal-click result. The isolated fixture closed via EOF with exit code 0 and zero captured
  action bytes. The normal production host remained running with its previous configuration.
- Canonical `bin/snoofer.exe` and companions built successfully. Native ABI/buffer/callback-stop,
  media async timeout and canonical COM identity self-tests pass; the actual media DLL compiles.
- The initial media DLL probe opened and snapshotted successfully with no active sessions. After
  unlock, a read-only probe found the active Brave YouTube Music session, playing position,
  duration, canonical instance token 1 and 33,390 bytes of artwork. No media command was sent.
- Live Media on host PID 48916 showed 5150 artwork at 0:35/3:09 from music.youtube.com, with
  extension 1.1.0 reporting one tab with media. When the user closed YouTube Music, that tab card
  and artwork disappeared and the extension reported zero tabs with media; the separate paused
  Windows Brave session retained its Shape Of My Heart artwork. Reopening playback showed
  Come Together with new Abbey Road artwork and a clock advancing from 0:01 to 0:18. After live
  controls window 9637604 closed normally and the user reopened window 6493016, selecting Media
  restored the current Femme Like U artwork at 0:16/2:18. This fulfills live transport acceptance
  through the artwork lifecycle, including selective removal, continuing clocks and reopen.
- The soundboard normalization probe passes using temporary WAV files, with playback disabled.
- Extension unit/build tests: 16 pass. Chrome page-script and isolated Brave protocol 2 end-to-end
  checks pass. Browser media controls require the matching extension 1.1.0; the installed user
  extension was not reloaded. Refresh older media tabs after updating if Next/Previous is unavailable.
- During the initial deployment, the prior host and controls child exited normally. The new host
  started with the same `--config bin/snoofer.json` path. Audio configuration and mute-state file
  fingerprints and the user-owned native smoke script remain unchanged. The main host config was rewritten during
  normal startup, consistent with the existing Discord refresh-token rotation; no manual config
  edits or rollback were performed.
- During the later build attempt, host PID 70268 reached the cleanup deadline and displayed
  `context deadline exceeded`. It exited after the user dismissed its error dialog normally;
  no force termination was used. The failed stop remains failed evidence. Two decks were connected,
  but the awaited plugin and underlying cause were not identified. The named-error diagnostic in
  `c3cff86` and the brand fix in `9065ff7` are now in the canonical build. On the coordinated recheck,
  PID 51076 exited normally in under one second with both decks connected. The successful recheck
  does not establish the cause or a repair for the initial failure.
- The unchanged `node scripts/check-desktop.cjs` passes against the diagnostic build: native
  accessible Lights/Start Sync controls, revisioned actions, window close/reopen, host EOF cleanup,
  and scoped graceful tray shutdown with repeated stop. Its output is retained in
  `.local/gui-audit-native-smoke.log`. Production host PID 18648 was then restored with the same
  `--config bin/snoofer.json` launch configuration.
- The opt-in Deck baseline probe in `d56c6e1` adds `baseline_probe_windows_test.go`; package tests
  pass in 15.6 s. Actual Plus and Plus XL `inputReport` calls returned success with header
  `01 00 00 00` followed by zeros, which has an invalid key count. Holding blank key 8 on XL
  produced the same invalid report. The logs are `.local/deck-baseline-probe.log` and
  `.local/deck-baseline-held-probe.log`. These reports do not provide a usable initial key state.
  After reconnecting XL, the user observed that the first Play press on key 29 did nothing;
  the second worked and redrew. The user explicitly chose all-released initialization for keys
  and dial buttons: holding a button during startup is negligible in their use, and a displayed
  button must work on the first press. This supersedes the previous held-key suppression policy;
  the OpenSpec correction is committed in `2924946` and implementation in `03e74d2`. Eight startup regressions failed before the fix and pass afterward, including repeated down, release/repress and malformed input. Reopened-session generation checks pass. Full repository Go tests and vet pass. The canonical build passed, including companion self-tests, and restored the same config as PID 69960. The user confirmed all four hardware checks: first XL Play key, first XL dial 3 press, first Plus dial 3 press and first Plus Media key after reconnect. Three two-second XL Play holds/releases each toggled once per press, with no repeats or extra release action.
- At that earlier checkpoint, changed Go files were formatted; two untouched legacy
  Voicemeeter test files retained their pre-existing import formatting. The race detector
  was unavailable with `CGO_ENABLED=0`; the current portable-compiler race run above supersedes
  that limitation.

### Native fixture acceptance after unlock

The canonical executable at `49e3fd9` ran an isolated `__controls` fixture as PID 57508, without
host plugins or device writes. Production host PID 70268 remained unchanged during that inspection.
Nine captured actions were retained in the local acceptance log.

- Processing: closed Down staged Element without dispatch; Enter sent one revision-35 `set(element)`.
  Monitor: closed Down then Tab restored Pre without dispatch; typeahead `o` then Escape also
  cancelled. Open-popup Down+Tab sent one revision-36 `set(post)`; popup Enter sent one `set(off)`.
  In the later native fixture, opening Monitor at Pre and moving Down to Post then cancelling
  with Escape, Alt+Up or an outside click on a blank panel restored Pre each time. The action log
  remained zero bytes. After computer use could not target the separate popup, the user
  completed pointer selection on normal-build fixture PID 42296: Monitor Pre to Post sent exactly
  one `audio.normal-monitor` request, revision 36, operation `set`, value `post`. All specified
  native dropdown paths now pass. Fixture PID 42296 subsequently closed via EOF with exit code 0.
- Mic Mute stayed reserved after the 2.5-second Unknown threshold while Playback Mute still sent.
  An injected Playback Failed reply stayed on Audio; navigation to Soundboard showed no Audio
  error. A later Mic Processed reply remained on Audio, gain updates continued, and no action
  replayed. These injected replies establish correlation and presentation, not actual mute effects.
- Deck Page Options, an unapplied page-name draft, binding search and scroll survived structural
  rebuilds. Delete Page and Forget Deck cancelled with Escape; Own Layout cancelled with Cancel;
  none sent an action. Making Delete unavailable while its dialog was open invalidated acceptance
  while polling continued. A fresh acceptance sent one captured revision-1 request. App Audio More,
  nickname draft and combine selection survived a label rebuild after Tab, with Rename retaining
  focus and no restoration action. These observations fulfill functional native state preservation.
- Disabled Audio exposed Enable Audio with the captured `native-fixture-config` revision. A host
  proposal kept its original text and ID after a replacement arrived; Cancel sent the original
  `native-audio-enable-1` ID. Critical Audio and warning Routing markers were visible in the later
  packaged fixture. The resumed check showed Starting Audio, then the failed Audio gate and marker
  with the exact detail `Native acceptance: audio service unavailable`. Its Plugins link worked and
  Retry was available. Home-key navigation activated the marked Audio screen and exposed that
  reason. All of these navigation checks left the action log at zero bytes; no real plugin retry
  or lifecycle effect is claimed.
- Audio, Routing including Activity/output devices, Soundboard, Deck and Diagnostics were inspected
  at 1280×820 and 800×600 outer window sizes. The compact CSS client was approximately 784×562.
  Other inspected controls remained accessible through scrolling. The final outer 800×600
  inspection confirms full branding after the classic-scrollbar clipping fix. From Audio, End
  activated Diagnostics, then Up reached Third-Party Apps and Up reached Plugins with correct
  focus. Plugins/Third-Party Apps labels and layout now pass compact and 1280×820 inspection,
  including the wide cards at both top and bottom. Navigation emitted no actions. Both later
  compact and wide fixtures closed normally via EOF with exit code 0.
- Soundboard animation visibly changed from pie to green-square frames. This is synthetic rendering
  evidence, not live artwork lifecycle acceptance. Native window Close exited zero with an App Audio
  action still unresolved, confirming that the outstanding result did not block window shutdown.

### Live audio acceptance

- The user confirmed rapid Mic Mute + Playback Mute pairs work in both directions, On then Off.
  Separate live GUI checks showed each control's Processed/Saved result and both mutes Off.
- After two physical Mic Mute presses, the captured Mic Stage choice was stale. The user
  confirmed its rejection with "control changed; try again" and Mic Stage remained Pre.
  This verifies real drawn-revision rejection, not queued-target replacement.
- The unique 39-byte invalid `zz-snoofer-native-error-20261015-571b8a.mp3` appeared Ready in
  Soundboard. One click returned the correlated result "Failed · normalize Zz Snoofer Native Error
  20261015 571b8a: soundboard native error 0xC00D36C4"; no playback started. After verifying its
  exact content, only that file was removed. The GUI showed No matching clips while retaining
  the local failed summary for the vanished clip. This completes delayed native-error acceptance,
  not a timeout or uncertain-one-shot/no-replay check.
- A real recording started with Stop Rec/Pause shown. Pause changed the playback button to Play
  while retaining Stop Rec; Rewind/Fast-forward were disabled and Third-Party Apps → Voicemeeter
  Recorder explicitly reported Paused. Resume restored Pause; Stop restored Record/Play with Stop
  disabled. The new WAV from the 10:53:33 take is 3,458,740 bytes. This verifies GUI recording
  transport, not physical Deck transport, playback audibility or recording content.
- Temporarily disabling Record Microphone while Record Computer was already Off produced
  "Recorder: no eligible recording sources enabled". Record Microphone was restored On before
  the successful take. This is a source-validation refusal, not a delayed native error.
  After that take, Mic Stack was restored Off and verified in the GUI and saved state. The user
  later enabled Mic Stack; that current choice is retained.
- Voicemeeter was shut down normally from its menu; process inspection confirmed it absent while
  Snoofer remained running. After Voicemeeter was relaunched, Third-Party Apps showed Recorder
  Stopped and Audio Callback Monitor Active, with Buffers increasing from 218493 to 219302 and
  no callback error. The Voicemeeter/Recorder cards retained the real gap's last error,
  "VBVMR_IsParametersDirty: Voicemeeter is disconnected; start Banana or Potato". This verifies
  recovery after a real Voicemeeter process restart, not startup with Voicemeeter initially absent.
- In the separate startup check, the prior host and controls child exited gracefully with code 0.
  Snoofer then started with the same configuration while Voicemeeter was absent; native Audio
  explicitly showed Disconnected. After Voicemeeter launched, Third-Party Apps reported
  "Logged in; Voicemeeter was not running", Recorder Stopped and Audio Callback Monitor Active.
  Buffers advanced from 3685 to 5280 without restarting Snoofer. This verifies startup before
  Voicemeeter and subsequent callback registration.
- A temporary Audit Test output slot had no device and all sends Off. Bound as `audio.slot-1`
  on Plus XL Home key 8, it showed No Device without Unavailable; the user confirmed pressing it
  did nothing. Key 8 was restored to Empty/Saved and removing the slot returned Processed/Reloaded.
  The base configuration fingerprint returned to its original value and the user's enabled Mic
  Stack choice was preserved. At that point only the physical no-device/inert case was verified; healthy,
  disconnected-device caption and In Use checks remained open.
- In the later approved AirPods check, Normal Playback was explicitly moved to the SteelSeries
  speakers. A temporary Audit Test slot selected Headphones (Cody’s AirPods Pro); packaged Routing
  showed that device caption, the same current selection and In use. Computer, Monitor,
  Soundboard and Tape all remained Off. The display-only `audio.slot-1` binding was saved to
  XL Home key 8. The user confirmed Audit Test / In use in cyan without Unavailable. With both
  AirPods in the closed case, the key showed Disconnected with a neutral background and white
  text; native Routing showed Disconnected while retaining the device caption and current selection. Reconnecting returned
  the physical key and native Routing to In use. Key 8 was then cleared and saved, the temporary
  slot removed, and Normal Playback restored to Automatic. The base configuration fingerprint
  matched its original value; saved state and native GUI retained Mic Stack On, both mutes Off,
  Normal Monitor Off, Mic Stage Pre and the original VR playback choice. Native Audio showed
  Applied / No stall detected with AirPods playback. This completes output-slot-detail task 4.
- A Windows Settings app-access request timed out after the initial 30-minute control window
  had expired. No Settings connection action was performed. Root reported the overrun and
  retained temporary setup, then obtained a fresh five-minute Snoofer-only window before
  resuming. The physical case cycle and full cleanup completed within that new window; control
  was released. No Windows connection settings were changed.

The initial scratch-only second-deck holder was prepared and reviewed, with atomic readiness and
eight fake-handle cases passing 20 repetitions; its entry skips without opt-in. During the
authorized live attempt, the host and controls child stopped normally and the Plus serial/model/
path were verified, but the holder's exclusive read-only open itself failed with a sharing
violation. It exited 1 in 0.01 seconds without acquiring an exclusive handle; no fallback or
other-app closure followed. This is refused setup, not proof that Snoofer reports a refused
deck while another operates. The same-config canonical host was restored; both serials were
Connected/display On, callback Active (Buffers 4025), Recorder Stopped, Mic Stack On and both
mutes Off. The original configuration fingerprint was retained, with no disposable MP3,
silent player WAV or WMP test process remaining. The recording WAV described above was retained. Task 17 stays open;
no physical Play request was needed without a hold.

An owned WMP feasibility attempt exposed no Windows WMP session (only Brave) and no targetable
window. Computer-control approval/wait timeouts
exceeded the agreed window, so control stopped. Normal close returned false; after the user
explicitly authorized cleanup of only the owned test process, its PID/path/start identity was
verified, termination and exit were confirmed, and its exact 160,044-byte silent WAV was removed.
That attempt did not complete bounded normal-close feasibility, frozen-player or
uncertain-action checks. The later owned-player freeze passed separately, as recorded above.

Final reconciliation: source and validation work is complete. Refused-deck and real
uncertain-native-outcome acceptance tasks stay unchecked with their measured limits;
the completion procedure records these blockers without counting them as passes.
Native recovery, live artwork/clocks/reopen, rapid mutes, recording Pause/resume/Stop,
delayed native Soundboard failure, Voicemeeter restart/startup recovery, both Plus
models' first key/dial presses and healthy updates during an owned-player freeze pass.
The final Processed Pause produced one command and one effect without replay, but did
not establish native uncertainty. Captured stale Mic Stage rejection does not establish queued-target
replacement; that native evidence remains unclaimed. Mini/MK.2/original XL physical
checks are waived, not passed. No additional live attempts remain scheduled.

Native WebView2 confirmed item 2 during migration acceptance: Down on a closed Processing select
sent a set action. The corrected closed-select and popup keyboard paths now pass native checks;
popup cancellation and the user-verified exactly-once pointer selection also pass.

## Reliability

1. **Clicks are silently dropped.** `send` returns early while any action is pending
   (`app/web/app.js:71`), and every control on the screen is disabled for up to 2.5 s after any
   action. Pressing Mic Mute and then Playback Mute quickly loses the second press without feedback.
   Separately, `Desktop.Send` refuses an action when its queue is full and returns "controls busy;
   try again" (`app/desktop_windows.go:56`). That reaches the user only through the same global
   error notice, so it is a second way to lose an action.
2. **Keyboard browsing of a dropdown may write live settings (unverified).** Every selection applies
   on `onchange` (`app.js:112`, plus the mic source, channel mode and region source selects). In
   Chromium on Windows, arrow keys on a closed `<select>` fire `change` per step, so arrowing through
   Processing or Monitor would apply each option. That conflicts with the rule that browsing never
   writes audio settings. The deck binding listbox (`app.js:1031`) changes the draft on every arrow
   press. This still needs to be confirmed in WebView2. If it is confirmed, it is the most severe
   finding here, because it breaks a live-audio invariant.
3. **Rebuilds discard UI state.** Any structural change rebuilds the whole screen (`layoutKey`,
   `app.js:1197`), for example a browser tab starting to play on Media or App Audio. This collapses
   open `<details>` (App Audio "More", Other Rooms, Page Options), loses scroll position and focus,
   and clears unapplied text in fields that are not focused.
4. **Errors follow the user across screens.** In the test run, "Control changed; try again" stayed
   visible from Audio through Soundboard, Lights, App Audio and Diagnostics. The contract asks for
   errors to stay local.
5. **Connection loss has no recovery state.** Each failed poll calls `showError`, with no backoff and
   no distinct reconnecting state. The error notice matches the current UI contract; introducing
   reconnecting feedback is an intentional UX change, not a correction to that contract.
6. **Confirmations are inconsistent.** Delete Page, Forget Deck and turning Own Layout off use the
   native `window.confirm`, which also blocks polling. Host confirmations use the styled `<dialog>`.

## Content and vocabulary

7. **Processing and Monitor show raw values** ("direct", "element", "off", "pre", "post"). The audio
   plugin sets no `OptionLabels` for mode or monitor (`plugins/audio/audio.go:419`). The contract
   uses Direct / Element and Off / Pre / Post.
8. **"virtual:1 Playback" appears on the Audio screen.** This internal label (`audio.go:368`,
   group "Shared Audio") shows in the Routing & Recovery card and duplicates the Computer cell of the
   Routing matrix. The card filter only excludes the "Routing" group (`app.js:210`).
9. **Internal names on other screens.**
   - Stream Deck: the page selector shows page IDs ("home"), and the binding list shows provider
     labels with plugin suffixes ("mic gain · audio").
   - Plugins: shows plugin IDs (`appaudio`, `nowplaying`) instead of names.
   - Echo Cancellation: the Retry row has no label.
10. **Diagnostics → Current Notices lists every status message.**
    - "VR override" appears four times with no profile named.
    - "Music: Speakers (Arena 7)" is not a notice, and Start at Login is repeated.
    - Notices are not grouped by plugin and cannot be copied.
    - The Audio Engine card repeats controls already on the Audio screen.

## Layout

11. **Audio.**
    - Cards stretch to their row: Recording pads to the Echo card's height, leaving a large empty block.
    - In the three-column Live Controls row, Mic Stack, Mic Mute and Playback Mute sit far from
      their switches.
    - The Interface row is stranded below the gain strips.
12. **Routing.**
    - In the half-width Playback card, Up/Down/Remove wrap onto their own line.
    - Each microphone name has its own Apply button.
    - The page is about 2,000 px tall with no in-page navigation.
    - Output slot management (device, rename, remove) is packed into the table headers.
13. **Stream Deck.**
    - The sticky Save bar covers the Page panel because the page has no bottom padding.
    - The Device and Page selectors are misaligned.
    - "Edit Shared Binding: Off" has no context.
14. **Other.**
    - Meter scale labels collide ("-3 0") on narrow strips (App Audio, 800 px).
    - Soundboard search has no "no matches" state.
    - Disabled-plugin screens are handled four different ways (Enable card, "Enable it in Plugins",
      `pluginOff`, "disabled or editor not ready").
    - The sidebar gives no hint of which screen needs attention.

## Code health and tests

- `app/web/app.js` is 1,260 lines: all 11 screens, widgets and global mutable state in one module,
  mostly in dense one-line style. Only `model.mjs` has unit tests (77 lines).
- `scripts/check-gui.cjs` is a single 681-line script:
  - One fixture is shared and mutated by every step, leaving the injected error visible in later
    screenshots. Fresh fixtures prevent test contamination, but a separate cross-screen journey
    must still verify that real action errors stay local.
  - It stops at the first failure and runs only with `scripts/check.ps1 -GUI`.
  - It writes screenshots but never compares them.
  - It has no automated accessibility scans. It already checks deck arrow navigation, range
    selection and no dispatch during focus movement; dropdown commit/cancel and complete keyboard
    journeys are missing.
- The fixture is written by hand and has drifted from the real providers. That is how items 7 and 8
  went unnoticed.

### Found by the generated fixture

15. **Activity settings never show on Routing.** The audio provider publishes the activity settings
    with their JSON names (`check`, `silence_db`, `silent_after_s`), but the GUI reads `Check`,
    `SilenceDB` and `SilentAfterS` (`app.js:360`). With real data the Meter choice always shows Off
    and the threshold fields are empty. The hand-written fixture used the GUI's spelling, so the old
    check passed. The Routing spec records it as an expected failure owned by `gui-content-fixes`.
16. **Every audio selection carries microphone option labels.** `audio.mode`, `audio.monitor`,
    `audio.output` and the record tap publish the microphone `OptionLabels` map, not labels for
    their own options. This is the cause of item 7.

## Proposed changes

Each change has its own reviewed OpenSpec design, split into commit-sized tasks with verification
at each checkpoint. The change descriptions below retain the original scope; the status table
and OpenSpec task lists track implementation and acceptance. The audit itself is a planning input;
the user subsequently authorized implementation.

The one-off WebView2 check through `__controls` confirmed item 2 during migration acceptance,
so change 3 is required and its source fix has landed. Native keyboard commitment now passes;
popup cancellation and exactly-once pointer selection also pass.

Order (decided 2026-10-14): 1 → 4 → 2 → 3 → remaining GUI checkpoints → final shared gates →
packaged/live acceptance. The harness and TypeScript prerequisites are satisfied. Independent
GUI checkpoints have progressed alongside provider outcome work; `audit-hardening` resumed and
its source checkpoints are committed. Remaining hardware acceptance stays separately tracked.

1. **`gui-test-harness`.** Move to Playwright Test:
   - First port every existing assertion with no application behavior change. Keep the existing
     model tests, use a fresh fixture per test, and group specs by screen plus cross-screen journeys.
   - Generate fixture controls through real provider publication code with fixed inputs and fake
     backends, so labels cannot drift. Include separate live-available and preview cases: preview
     deliberately disables audio gain/transport and several Hue controls that existing tests use.
     Fixture generation must not depend on installed devices, network services or personal settings;
     use isolated temporary configuration/state and inert or temporary saves.
   - Add keyboard journeys and axe-core scans. Preserve the existing keyboard assertions; add
     regressions for known behavior defects with the change that fixes them. Record existing
     accessibility failures with an owning change rather than treating a scan as proof of compliance.
   - Check layout at 1280×820 and 800×600. If screenshot comparisons are adopted, control timestamps,
     telemetry and animation, and document the browser/font environment used for baselines.
   - Fixture mechanism: generate fixtures with a Go test or a `tools/` generator, never a new
     production subcommand. Commit the generated fixture together with a test that fails when it
     no longer matches provider output. Plugins without a fake backend yet (likely Hue, App Audio,
     Now Playing and Discord) may stay hand-written at first, with a check that their IDs and labels
     still match the providers, rather than blocking the harness on building every fake.
   - Run by default in `scripts/check.ps1`; the native desktop smoke stays opt-in.
   - Acceptance:
     - The migrated suite retains every existing scenario, and tests pass independently.
     - Fixtures reproduce provider labels, and checks require no live integrations or personal state.
     - The suite has a stated time budget, measured against today's 27 s check, because it runs
       on every `check.ps1`.
     - Tests use the installed Chrome (`channel: "chrome"`). On a machine without it, the suite
       fails with a clear message and never downloads a browser.
2. **`gui-content-fixes`.** Items 7–9, 15 and 16, including the Retry label:
   - Fix labels at the owning provider or presentation layer; keep IDs, bindings and saved values
     unchanged. Verify rendered labels with the provider-backed fixtures.
3. **`gui-dropdown-commit`** (item 2 confirmed in WebView2). A small, early fix for
   a live-audio invariant breach:
   - Define browsing, explicit keyboard or pointer commitment, and Escape cancellation.
   - Apply a committed choice once. Distinguish live audio selections from deck draft edits, and
     preserve regular controls' live behavior.
   - Acceptance: a WebView2 regression check shows that arrowing through Processing and Monitor
     sends nothing until a choice is committed. Run this targeted check even though the general
     native smoke remains opt-in.
   - If the check does not confirm item 2, keep the commit/cancel definition in change 6 and add a
     regression test so the behavior stays verified.
4. **`gui-typescript-modules`** (was `gui-modules`). Rewrite `app.js` as TypeScript modules, one
   per screen plus shared widgets, state and host binding, with readable formatting, so each part
   can be tested without changing behavior:
   - Strict `tsc` (`strict`, `noUncheckedIndexedAccess`) type-checks; esbuild bundles to one
     `dist/app.js`. Types for the host's `ViewState`, `Control` and `UIAction` mirror the Go structs.
   - typescript-eslint (recommended) and Prettier enforce hygiene; type-check, lint and format all
     fail `scripts/check.ps1`. No frontend framework.
   - Update the explicit asset list in `app/desktop_windows.go` and the source list in
     `app/web/tailwind.css` in the same change. New modules must be embedded and their utility
     classes included in generated CSS; filesystem-served browser tests cannot prove packaging.
   - Add focused tests for extracted behavior. Verify the existing browser suite and canonical
     `scripts/build.ps1` output, then run native desktop smoke against that executable and inspect
     representative screens for missing imports or styles. Do not mark packaging verified from
     source-served tests alone.

Changes 5–7 replace the earlier single `gui-action-reliability` change, which covered items 1–6.
They are split by risk so each can be reviewed and reverted on its own, and so the GUI-only work in
change 6 does not wait for the cross-layer design in change 5.

5. **`gui-action-outcomes`.** Items 1 and 4. This spans the GUI, host and providers, and touches the
   stale-action and no-replay invariants, so it needs an explicit design update:
   - Design action ordering and outcome attribution across the GUI, host and providers before
     replacing the global pending flag. Audio controls share an epoch (`audio.go:430`); one saved
     edit advances it, so another action from the same snapshot can be rejected as stale.
     `Desktop.Send` acknowledges queueing only, refuses actions when its queue is full, and host
     dispatch errors currently use one global `Notice`. Per-control pending alone does not solve
     any of these.
   - Keep pending state and outcomes attached to the originating action/control. Preserve stale
     validation and target identity through queued input, edits and confirmations; do not silently
     retarget an action using newer state. An enqueue acknowledgment, unrelated revision change or
     timeout must not be presented as observed success. Never replay failed or uncertain one-shot
     commands automatically.
   - Keep action errors local to their originating screen.
   - Acceptance: rapid Mic Mute → Playback Mute applies both valid intentions through the agreed
     ordering policy; stale or unavailable targets receive explicit rejection. Cover delayed or
     failed dispatch, a full action queue ("controls busy") and cross-screen errors.
6. **`gui-state-preservation`.** Items 3 and 6, GUI only:
   - Preserve focus, scroll, expanded details, search and unapplied text during structural updates.
   - Use the styled dialog for all confirmations while preserving each action's warning, cancellation
     and draft/save semantics; polling must continue while the dialog is open.
   - Include the dropdown commit/cancel definition here if change 3 was not needed.
   - Acceptance: cover structural updates during editing and confirmation cancellation.
7. **`gui-connection-recovery`.** Item 5:
   - Define global host-disconnection feedback, polling backoff and recovery. Update
     `ui-contract.md` in this change to replace its host-loss error-notice rule with the chosen
     reconnecting behavior; specify placement and clearing on recovery, without hiding unresolved
     action errors.
   - Never replay failed or uncertain one-shot commands after reconnect.
   - Acceptance: cover disconnect and reconnect without replay.
8. **`gui-layout-polish`.** Items 10–14 except sidebar attention and disabled-plugin cards, which
   belong to change 9. Work screen by screen, update the contract for intentional UX changes, and
   verify each affected screen at both viewport sizes. Reconcile the proposed Diagnostics cleanup
   with the contract's explicit Engine row on both Audio and Diagnostics before removing controls.
9. **`gui-attention-journeys`.** Sidebar attention markers and consistent disabled-plugin cards.
   Define the states that require attention and their navigation targets in the contract; retain
   the shared header's transient-only rule. Verify navigation from an attention marker and each
   disabled-plugin screen's enable/recovery path.

## Decisions and deferred follow-up

- Resolved in `gui-test-harness`: add `@playwright/test` 1.62.1 and `@axe-core/playwright`; the
  installed Chrome only, never a browser download. DOM geometry assertions plus screenshots as
  review artifacts, not committed baselines.
- Resolved for `gui-typescript-modules`: TypeScript with strict `tsc`, an esbuild bundle,
  typescript-eslint and Prettier.
- Follow-up from `gui-test-harness`: generate the fixtures that stay hand-written (Soundboard, Hue,
  App Audio, Now Playing, Discord, AEC, VR) once those providers have fixed-input entry points.
- Resolved in `gui-action-outcomes`: bounded session-local IDs and result lanes, per-control busy
  reservations, captured provider targets and explicit Submitted/Processed/Rejected/Failed/Unknown
  boundaries. Only the two global mute preferences have independent worker generations; all
  contextual stale checks and no-replay rules remain. Frontend timeouts retain the reservation.
- Resolved in `gui-typescript-modules`: one module per screen plus shared state, host binding,
  widgets and model helpers. No frontend framework; esbuild is the only bundler.
- Acceptance remains concrete: full isolated browser suite within 60 seconds, strict source gates,
  canonical build, unchanged native smoke and packaged journeys at both sizes. Hardware-dependent
  tasks remain unchecked when the necessary device or observable result is unavailable.
