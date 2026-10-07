# Validation
2026-10-06, commits c406a65 through 0f472db.

## Automated checks
- `go vet ./...` and `go test ./... -timeout 120s`: pass. The nowplaying tests were run 5 times in a row; they cover:
  - merging and de-duplication, focus, interpolation, pending/observed/declined, and preview;
  - WebSocket origin and token checks, size limits, replacement and token reset;
  - saving the extension files.
- `node --test plugins/nowplaying/extension/logic.test.mjs`: 4/4.
- `scripts/check-extension.cjs`: pass, run 3 times. The page script in Chrome reports metadata, state and controls, and runs toggle, the page's `nexttrack` handler and seek.
- `scripts/check-gui.cjs`: pass. It covers the Media screen's cards, play, seek, focus, mute and the extension card.
- `scripts/check-desktop.cjs` and `openspec validate now-playing --strict`: pass.
- `scripts/build.ps1`: builds `snoofer-media.dll` with C++/WinRT at `/W4 /WX`.

## Native probe
`SNOOFER_MEDIA_PROBE` against the built DLL found Brave's single session ("U2 – The Troubles", 4:45, seekable) with a 12 KB cover. `current` initially came back false because WinRT returns new session objects; it was fixed to compare by AppUserModelID.

## Incident
The first build saved the new bridge token from inside `Start`. The host holds its lock while starting plugins, so the save deadlocked the host, and plugins after `nowplaying` (Soundboard, Stream Deck, VR) never started.
- **Recovery:** the graceful stop correctly refused to kill the process. With the user's approval, only that process (PID 34576) was force-ended.
- **Fix (0f472db):** the token is created in memory at start and saved from the worker after startup.
- **Regression test:** the bridge rig's SaveSettings now fails if it is called during `Start`.

## Personal setup and relaunch
- **Personal setup:**
  - `nowplaying` enabled;
  - Media page added (sessions in r1 c1–c8, transport and Focus on r2, Up/Down, dials Media and Playback);
  - Home key 33 (r4c6) set to Go to Media, and Home dial 3 set to the media dial;
  - `snoofer.exe --check` passes; backup `snoofer.json.before-media`.
- **Relaunch:** Snoofer was relaunched. It listens on `127.0.0.1:47815`, and the token was saved after startup.

Hardware acceptance (task 9) is pending, including installing the extension in Brave.

## Revision: standalone store extension (commits bb81bc8 through d565bec)
Review found the export pattern unsuitable for store publishing, and the first install showed nothing. The repository copy lacked the generated `config.js`, so the background failed on its first line.

### What changed
- The extension is now its own project in `extension/`, with shared sources and generated Chrome and Firefox manifests.
- There is no token, export or pairing. The bridge accepts `chrome-extension://` and `moz-extension://` origins on loopback, and refuses mismatched protocol versions with a reason.
- Windows sessions whose titles match a connected browser's tab are hidden, which covers Firefox.

### Checks
- `go vet ./...` and `go test ./...`: pass. The nowplaying tests were run 3 times in a row; they cover web origins refused, both extension origins accepted, protocol refusals (4002 plus a reason), title de-duplication and a legacy token ignored.
- `npm test` in `extension/`: 7/7, covering logic, manifests and build output.
- `npm run package`: produces both zips with `manifest.json` at the root. Windows' own bsdtar is used, because Git's GNU tar cannot write zips.
- `scripts/check-extension.cjs`: pass, run 3 times.
  - The page script checks in Chrome pass.
  - **Live end to end:** the built extension, in Brave with a fresh profile, connects to a stand-in bridge. It sends a protocol 1 hello as Brave from a `chrome-extension` origin and reports the playing tab. Bridge commands toggle the page, use its `nexttrack` handler and mute the tab.
- `scripts/check-gui.cjs` (the extension card has no export or token buttons) and `openspec validate now-playing --strict`: pass.
- **Real bridge:** the built extension in a fresh Brave profile connected to the relaunched Snoofer, unrefused, and the connection is shown as ESTABLISHED to Snoofer's PID.

### Personal setup and limits
- **Config:** the legacy token was removed from the personal config (backup `snoofer.json.before-extension`), and Snoofer was relaunched.
- **Not tested:** Firefox, which is not installed here. Its manifest is covered by the build tests only.

## Revision: nothing playing, focus and bottom-row controls (commits df0124d through 8564e65)
### Cause
- **Diagnosis:** the user's Brave extension was connected (an ESTABLISHED connection to Snoofer), and the plugin, run against the real companion, saw Brave's Windows session.
- **Combination:** Snoofer hid Brave's Windows session as soon as the extension connected. Meanwhile the extension reported nothing, because the video tab was opened before the extension was installed. Content scripts reach only pages loaded afterwards, and an already-playing element never fires `play`.

### Fixes
- **De-duplication:** title only, so a Windows session is never hidden unless the extension reports its tab.
- **Existing tabs:** the extension injects into open tabs on install (`scripting` permission).
- **Already-playing media:** `page.js` adopts media already in the document, or seen through `timeupdate`.
- **Focus:** when exactly one session plays and the focused one does not, focus moves to it.
- **Media page:** transport keys moved to the bottom row (keys 28–32), with sessions on r1–r3 c1–c8, in the defaults and the personal layout (backup `snoofer.json.before-media-row`).

### Checks
- `go test ./...`: pass. New tests cover a Windows session kept while its tab is unseen, focus moving to the only playing session, and the bottom-row Media page.
- `npm test`: 7/7.
- `scripts/check-extension.cjs`: pass, run 3 times.
  - **Chrome page stage:** `page.js` injected into a tab already playing reports it at once.
  - **Brave live stage:** the install hook re-injects into open tabs without duplicating the session, and the tab stays controllable.
- **Build and relaunch:** built and relaunched; `snoofer.exe --check` passes.

### Not automated
Restarting an unpacked extension after `runtime.reload()` does not work under automation, so the update path in a real install is covered by hardware acceptance.

## Revision: refined seeking (commits bed3e51 through ab24de6)
### Causes
- **Dropped turns:** the dial's value changed every second, so its revision changed too. Deck turns that crossed an update were rejected as stale, and the deck dropped events from the old generation.
- **Seek per detent:** each detent sent its own seek.
- **Display:** it snapped back until the player confirmed, and slow confirmations showed "No response".
- **GUI slider:** it froze after one drag, because focus blocked updates.

### Fixes
- **`snoofer.Progress`:** display-only telemetry that the registry, like `Meter`, ignores for revisions. The deck renders interpolated `m:ss / m:ss` from it, with a status note.
- **Coalesced scrubbing:** detents gather into one seek, sent 250 ms after the dial rests.
- **Optimistic progress:** the requested position is shown until the player reports it.
- **Quiet timeout:** a seek the player never confirms clears without an error.
- **No Pending on the dial:** a status change would itself change the revision.
- **GUI:** cards interpolate progress, and the slider uses a dragging flag.

### Checks
- `go test ./...`: pass. The nowplaying tests were run 3 times in a row. New tests cover:
  - progress interpolation and text;
  - the revision staying stable while progress advances, with input still accepted;
  - three quick detents producing one seek to +15 s with an unchanged revision;
  - a follow-up turn starting from the target;
  - a quiet timeout.
- `node --test app/web/model.test.mjs`: 6/6, including `progressAt`.
- `scripts/check-gui.cjs`: pass. The slider follows the session after a seek, and a playing card advances between polls.
- `scripts/check-desktop.cjs` and `scripts/check-extension.cjs` (Chrome and Brave stages): pass.
- **Build and relaunch:** built with `scripts/build.ps1`; Snoofer relaunched.

## Revision: combined Media deck page (commits bf276e8 through 67138df)
### Checks
- `go test ./...`: pass. The appaudio and nowplaying tests were run 3 times each. They cover:
  - the Apps filter: All, Pinned, Off; filtered apps still published for the GUI; the filter surviving GUI edits; All stored as the default;
  - the Media filter: saved; sessions, dial and transport hidden while the filter key stays visible; restored on the next press;
  - the combined default page: sessions on r1, app keys over dials 2–5, the bottom row, and six apps paging keys and dials together;
  - Home's go-to keys.
- `scripts/check-gui.cjs` and `scripts/check-desktop.cjs`: pass.
- **Build and relaunch:** built with `scripts/build.ps1`; Snoofer relaunched.

### Personal layout (backup `snoofer.json.before-combined`)
- The Apps page was removed; it held only Up/Down.
- The Media page was rebuilt to the combined layout. Its Playback dial gave way to the app dials, and Playback stays on Home.
- Home key 34 now goes to Media, and key 33 is free.
- `snoofer.exe --check` passes.

## Revision: filters free space, and coloured placeholders (commits 6516d30 through ffac3d4)
- `go test ./...`: pass.
  - **Stacked regions:** rows shared by need in five member mixes; overflow continuing each source; validation; the "Media sessions + Apps" label; clone independence.
  - **Yielding:** hidden bindings yield to covering key and dial regions, while Stream Deck keys never do.
  - **Media page, four filter states:**
    - both on: sessions row 1, apps row 2, app dials 2–5;
    - Apps off: sessions take rows 1–3;
    - Media off: apps take rows 1–3 and all five dials;
    - both off: empty.
  - **Placeholders:** deterministic and distinct per name; a 64 px PNG with a transparent corner; used for apps without an icon and sessions without art, while real icons are kept.
- **Native-size preview:** six placeholder keys, each name in its own colour with a window or play glyph. Similar hues can still occur for different names.
- `scripts/check-gui.cjs` and `scripts/check-desktop.cjs`: pass.
- **Build and personal layout:** built with `scripts/build.ps1`. The personal Media page was migrated to the stacked region and the five-dial app region (backup `snoofer.json.before-stacked`); `snoofer.exe --check` passes. Snoofer was relaunched.

## Revision: hold to focus, focus dials, Playback (commits 278b6e8 through 4cf36b1)
- `go test ./...`: pass.
  - **Deck keys:** the decoder reports key releases, and a release is not repeated. On the deck, a plain key acts on key-down. A hold key sends press on a quick release, sends hold once at 500 ms and ignores that release, and still counts a tap whose release arrives after a generation change. These were run 3 times.
  - **Sessions:** a session hold focuses without toggling. The Focus control no longer exists, and the single-playing rule is reverted with its test.
  - **Apps:** the focus dial defaults to the first app on the deck. A hold moves it without writing, the dial then mutes the held app, and with the Apps filter on Pinned it shows the pinned app.
  - **Media page:** dials are media, Playback and the focused app; no Focus key. Across the four filter states: both on gives 2 app dials (4–5); Apps off gives none; Media off gives 3 (1, 4, 5); both off gives none.
- `scripts/check-gui.cjs` (no Focus button on media cards) and `scripts/check-desktop.cjs`: pass.
- **Build and personal layout:** built with `scripts/build.ps1`. The personal Media page lost the Focus key (key 32), and its dials became media, Playback and the focused app (backup `snoofer.json.before-focus`). `snoofer.exe --check` passes, and Snoofer was relaunched.

## Revision: Playback first, no duplicate dials, transport on Home (commits 862ba64 through ebd8038)
- `go test ./...`: pass.
  - **Mirrored control:** a visible stand-in dial keeps its mirrored control off the page's region dials, and the next app takes the dial. A hidden stand-in yields its dial and mirrors nothing. `appaudio.focus` mirrors the focused app.
  - **Media page dials:** Playback, media, focused app. With media off, the app dials are 2, 4 and 5.
  - **Home:** keys 28–30 are the Now playing transport.
- `scripts/check-gui.cjs` and `scripts/check-desktop.cjs`: pass.
- **Build and personal layout:** built with `scripts/build.ps1`.
  - The personal Media page's first two dials were swapped (Playback first).
  - Home's blind media keys were replaced by `nowplaying.prev`, `nowplaying.toggle` and `nowplaying.next` (backup `snoofer.json.before-home-transport`).
  - `snoofer.exe --check` passes, and Snoofer was relaunched.

## Revision: Home revamp (commits dcd561c and 3a343ed)
- `go test ./...`: pass.
  - **Clipped Home strips:** with six sessions and nine apps they show the first four of each and add no overflow sets. A clipped region beside an unclipped region of the same source still lets the unclipped one page.
  - **Default Home:** transport, Brightness and Motion on the bottom row; no Controls or Hue Sync keys.
  - **Older editor test:** it now clears Home's regions before testing the legacy prefix.
- `scripts/check-gui.cjs` and `scripts/check-desktop.cjs`: pass.
- **Build and personal Home:** built with `scripts/build.ps1`.
  - The Hue Sync, Mode, Intensity and Controls keys were removed.
  - Brightness and Motion moved to keys 31 and 32.
  - Clipped strips were added: sessions on keys 19–22 and apps on 23–26 (backup `snoofer.json.before-home`).
  - `snoofer.exe --check` passes, and Snoofer was relaunched.
