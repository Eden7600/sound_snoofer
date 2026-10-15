# Repository audit: smaller findings (2026-10-14)

Lower-priority findings from the audit in [repo-audit.md](repo-audit.md), at
commit `237a8ee`. Numbering continues from that file.

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
