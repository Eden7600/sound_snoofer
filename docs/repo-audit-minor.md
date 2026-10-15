# Repository audit: smaller findings (2026-10-14)

Lower-priority findings from the audit in [repo-audit.md](repo-audit.md), at
commit `237a8ee`. Numbering continues from that file.

## Current status (2026-10-15)

Reconciled with completed source work and final `audit-completion` evidence. Original findings and source
locations below are retained as historical evidence; current implementation is
tracked here and in the linked OpenSpec tasks.

| Findings | Current disposition | Implementation / evidence |
| --- | --- | --- |
| 21 | Fixed; callback cleanup distinguishes nothing registered from uncertain cleanup, and live Voicemeeter recovery passes | [audit-hardening](../openspec/changes/audit-hardening/tasks.md) |
| 22 | Fixed; assume released buttons when no usable startup state exists, as the user explicitly requested. First key/dial presses pass on both Plus models; held XL presses toggle once | [audit-hardening](../openspec/changes/audit-hardening/tasks.md) |
| 23 | Per-device open failures, deduplication and recovery implemented; physical refusal setup blocked because both access-zero sharing modes fail acquisition with error 32. No holder acquired; no further mode attempts | [audit-hardening](../openspec/changes/audit-hardening/tasks.md) |
| 24 | Fixed with queue epochs so old-room completions cannot alter the new room | [audit-hardening](../openspec/changes/audit-hardening/tasks.md) |
| 25 | Complete in `f1c10fc`: every second live claimant for an occupied browser name is rejected, including the same origin; reconnect is admitted after incumbent cleanup. This does not authenticate the first claimant or pin an extension ID | [audit-completion](../openspec/changes/audit-completion/tasks.md) |
| 26 | Fixed in `a5ca385`: unchanged App Audio details retain selection while Heard updates; Media retains its icon within a card until playback state changes | [gui-polling-content](../openspec/changes/gui-polling-content/tasks.md) |
| 27–28 | Shared native-build setup and buffer helpers implemented, retaining callback validation | [audit-cleanup](../openspec/changes/audit-cleanup/tasks.md) |
| 29 | Complete in `9eb59e4`: Discord and Soundboard use the exact shared `snoofer.OnOff` conversion at all eight call sites | [audit-completion](../openspec/changes/audit-completion/tasks.md) |
| 30–31 | Listed dead code removed; strict storage decoding covered for valid/default/null, duplicate, unknown and trailing input | [audit-cleanup](../openspec/changes/audit-cleanup/tasks.md) |
| 32 | Fixed; display-only controls are allowed on keys and remain inert, with audio-first startup covered | [deck-status-bindings](../openspec/changes/deck-status-bindings/tasks.md) |
| 33 | Fixed; provider, renderer, browser, packaged and physical AirPods states pass | [output-slot-detail](../openspec/changes/output-slot-detail/tasks.md) |

The user's completion instruction supersedes the former retained work. Mini, MK.2
and original XL support stays as-is; unavailable physical acceptance is explicitly
waived, not passed. Refused-deck acceptance is blocked by incompatible device sharing;
frozen-player acceptance now passes through live healthy title updates during a verified
freeze and recovery afterward. The final GUI Pause was Processed once with one observed
effect and no replay; a real uncertain native outcome remains unverified. The canonical host was
restored with the same configuration as PID 70128 after the final holder setup failed.
Native UI recovery shows both decks Connected/Display On, callback Active, Recorder
Stopped and Windows defaults Verified; no new physical press confirmation is claimed.
[The main audit](repo-audit.md#current-status-2026-10-15)
records the wider acceptance limits.

Latest completion validation: full Go tests, vet and race checks, strict frontend
gates, 297 browser tests in 42.2 seconds and all 106 OpenSpec changes pass. The initial
Stream Deck fixture hang, Now Playing fixture race and Hue readiness failure were
repaired in `07b616f` and `ad94e6e`, with repeated race coverage. The canonical build
and native async/identity/callback self-tests pass; production source is unchanged.
After the build-time restore as PID 73604, the later same-config host is PID 70128
with controls PID 11816. Persisted audio-state and user-owned smoke-script fingerprints
stayed unchanged. Owned helper processes exited; their sessions and silent WAVs are gone and the test notice
is dismissed; native Media shows only the user's Brave cards. Computer control was
released at 20:05Z. No further live attempts are planned; refused-deck and uncertain-
one-shot acceptance remain unverified.

Earlier finding 26 checkpoint: its two regressions failed before the fix and pass afterward. Full strict
frontend gates, 15 model tests and 297 browser tests pass (44.2 seconds); both
screens were inspected at 1280x820 and 800x600. All 105 OpenSpec changes validate.
The canonical build and native self-tests pass; the tray host restarted with its
previous `--config bin/snoofer.json` command. Persisted audio-state and user-owned
native-smoke-script fingerprints stayed unchanged. No desktop input was taken.
Native lost-click behavior was not reproduced; the confirmed defect was repeated
node replacement during polling.

## Original findings

21. **`SnooferStop` treats "nothing registered" as an uncertain cleanup**
    (`internal/voicemeeter/callback/monitor.c:50-59`,
    `internal/voicemeeter/callback_windows.go:241-246,315-319`)
    - **Problem:** After unregister returns 0 or 1, it returns
      `stopped ? stopped : timing`. `AudioCallbackStop` returns -2 for "no
      callback registered", so a clean state still reports -2. Go then sets
      `unsafeClose` and disables the monitor until restart. Example: Voicemeeter
      is closed and reopened while Snoofer runs, and `RestartCallback` hits this
      path.
    - **Fix:** `return (stopped && stopped != -2) ? stopped : timing;`
    - **Caveat:** This depends on Voicemeeter's real return codes, which were not
      observed.

22. **The first Stream Deck press after connecting may be swallowed**
    (`internal/streamdeck/protocol.go:60-74`)
    - **Problem:** The first key report after a connect is used as a baseline and
      produces no events. If the deck only sends reports on change, that first
      report is the user's first real press.
    - **Fix:** Read the current state with `HidD_GetInputReport` in `openDevice`
      and use it as the baseline. Or accept the first report as a baseline only
      if it arrives within about 100 ms of opening.
    - **Caveat:** It was not confirmed on hardware whether the deck sends a
      report when opened.

23. **A second deck that fails to open is never reported**
    (`internal/streamdeck/surface_windows.go:96-101,129-136`)
    - **Problem:** Open errors are reported only when no deck is running. With
      one deck working, a second deck that fails (an unsupported descriptor, or
      access denied) is retried every 2 s with no message anywhere.
    - **Fix:** Send `Event{Device: <path id>, Error: ...}` for every failed path.

24. **Changing Hue room resets the write queue while a write is in flight**
    (`plugins/hue/plugin.go:465-476`, `plugins/hue/groupdials.go:276-313`)
    - **Problem:** `resetGroupDials` clears `inFlight`, but the old `Put` still
      completes with the same generation. A new room's write can then run
      alongside it. The late result also clears `inFlight` early and can mark the
      new room's dial "Pending" or "Not applied".
    - **Fix:** Add a per-queue epoch, bump it on reset, and have `lightWritten`
      ignore results from an older epoch.

25. **Any installed extension can pose as the browser bridge**
    (`plugins/nowplaying/bridge.go:152-158,210-216`)
    - **Problem:** `CheckOrigin` accepts any `chrome-extension://` or
      `moz-extension://` origin, and a newer connection with the same browser
      name replaces the old one. Another extension can then evict the real one
      and receive its commands.
    - **Fix:** Pin the Firefox ID (`snoofer-media@sound-snoofer`) and the Chrome
      ID once published, or reject a second live connection for the same name.

26. **App Audio "More" list and the Media play icon are rebuilt at 10 Hz**
    (`app/web/app.js:787-790,822`)
    - **Problem:** The list runs `replaceChildren()` every 100 ms, so text in it
      (such as a Programs path) can't be selected. The play button's SVG is also
      replaced every tick, so a slow click may land on a removed node. The click
      loss was not confirmed in WebView2.
    - **Fix:** Guard both with a key, the way `reportCard` and `relabel` already
      do.

27. **Native build boilerplate is copied six times**
    - **Where:**
      - `scripts/build-aec.ps1`
      - `scripts/build-media.ps1`
      - `scripts/build-monitor.ps1`
      - `scripts/build-neural-aec.ps1`
      - `scripts/build-soundboard.ps1`
      - `tools/callback-probe/run.ps1`
    - **Problem:** The vswhere → vcvars64 → build.cmd setup is repeated in each,
      so a toolchain change must be made six times.
    - **Fix:** Move it into one shared script function.

28. **Duplicated native helpers**
    (`internal/aec/native/aec.cpp:202-216`, `neural.cpp`,
    `internal/voicemeeter/callback/pass_through.h`)
    - **Problem:** The `Valid`/`Has`/pass-through buffer helpers are written three
      times.
    - **Fix:** Define them once as `static inline` in `audio_buffer.h`.

29. **`onOff` duplicated**
    (`plugins/discord/controls.go:205`, `plugins/soundboard/voices.go:183`)
    - **Fix:** Keep one copy.

30. **Small dead code**
    - `Session.Current` (`internal/mediasessions/sessions.go:28`, computed in
      `media.cpp:142,186`). Nothing in production reads it. It would also be
      wrong if read: two sessions from the same app both come back as current.
    - `renderPresentation` (`internal/streamdeck/render.go:79-86`) has no callers.
      `render()` (`render.go:41-43`) is used only by a test.
    - `exitErr` (`app/controls_windows.go:26,74`) is assigned and never read.
    - `sessionFlagUnlock` (`internal/presence/presence.go:25`) is unused.
    - `fakeSessions.setPeak` (`plugins/appaudio/plugin_test.go:65`) has no
      callers.
    - **Fix:** Delete each one, along with the test assertion on
      `Session.Current`.

31. **`internal/storage` has no tests**
    (`internal/storage/json.go`)
    - **Problem:** `Decode` rejects duplicate keys, unknown fields and trailing
      documents, and it guards every config load. Nothing fails if it breaks.
    - **Fix:** One table test covering each of the three rejections.

32. **Stream Deck refuses its default layout when audio published first**
    (`plugins/streamdeck/plugin_windows.go:104`, `plugins/streamdeck/layout.go:216-231`,
    `plugins/streamdeck/default.go:73,175`)
    - **Problem:** `settings.validate` runs at plugin start against the current registry. The
      default layouts bind status controls (`audio.playback-device`, `audio.slot-*`) that have no
      `press` operation, so the plugin fails to start with "Playback Device does not support this
      position" whenever audio's controls are already published, for example when Stream Deck is
      enabled from the Plugins page while audio runs. Normal startup only works because audio
      publishes later. Found by the GUI fixture generator (`gui-test-harness`).
    - **Fix:** Accept status controls on keys (they render state), or validate only positions
      that the user edits.

33. **Healthy output-slot keys show Error on the physical deck**
    (`plugins/audio/outputs.go:45-47`, `plugins/streamdeck/meter.go:39-46,62-73`,
    `plugins/streamdeck/plugin_windows.go:285-287`)
    - **Problem:** Output slots publish their state in `Value` (for example `In use`)
      and the device name in `Status`. The hardware path calls `bindingTile`, which
      substitutes that descriptive `Status` for the value, then `controlTile` maps
      every unrecognised nonempty status to `Error`. The fixed audio fixture's healthy
      Music slot therefore renders a red Error key instead of In use. This predates
      the display-only binding fix; accepting those bindings makes it consistently
      reachable with audio-first startup.
    - **Evidence:** A scratch Go overlay passed the generated audio fixture through
      the real `bindingTile` and native 112px renderer. `audio.slot-1` became
      `Tile{Label: "Music", Value: "Error"}` despite its provider value being In use.
      Playback Device retained its scrolling device name; absent slots stayed blank.
      The existing `TestRoutingPresentation` supplies label/value strings directly,
      so it bypasses the faulty conversion.
    - **Fix:** Preserve the destination's state word while treating its device name
      as descriptive detail. Cover the actual provider-to-tile path, including In use,
      Disconnected, No output and hidden slots, and inspect native-size output. Preserve
      genuine pending, unavailable and error semantics on other controls; do not simply
      ignore every status string. Specify this separately from `deck-status-bindings`,
      whose design intentionally retains existing rendering.

    - **Source fix verified:** `output-slot-detail` moves the device caption to `ViewData.Device`.
      Provider-backed Go and Routing tests pass; the actual offline 112px renderer preserves
      In use, No device, Disconnected, No output and blank hidden slots. Canonical packaged and
      physical-device acceptance now pass: No device, cyan In use, neutral Disconnected,
      retained Routing device name/current selection, and return to In use after reconnect.
      The temporary output/key binding was removed and Automatic playback restored.
