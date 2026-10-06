# Validation

## Automated (2026-10-05)
- gofmt (changed files), `go vet ./...` and `go test ./...` pass. New tests cover:
  - core: report deep copies, revision behavior and tracker semantics (since on state change, error retained after recovery, persistent failure keeps its first time);
  - Hue: Bridge and Sync reports against the fake bridge and app, including stream gap, reconnect count and absence of the key;
  - audio: Voicemeeter lifecycle and the callback, recorder, ASIO, Element and Windows-defaults reports from worker states;
  - SteamVR transitions; the Stream Deck device report through the surface fake (no layout status leakage); soundboard playback; media keys.
- `node --test app/web/model.test.mjs` passes (relative times, stale detection, tone including required peers, copy text).
- `check-gui.cjs` passes: cards, summary counts, required-peer tone, hidden Required detail, zero times omitted, Copy details and Copy all (clipboard stubbed), not-monitored list, no overflow at 800 px. Reference image: `docs/design/gui-apps.png`.
- `check-desktop.cjs` passes against the new binary. Strict OpenSpec validation and `scripts/build.ps1` pass; Snoofer was relaunched without automation NO_COLOR.
- `go test -race`: not run (no C compiler; toolchain unchanged).

## Live screen (real environment)
The running build published all 12 reports. Read through the window accessibility tree, the summary was 7 OK, 0 Attention, 0 Problems:
- Voicemeeter: Potato 3.1.3.0, login "Voicemeeter running", Live.
- Windows defaults: Verified, with targets Voicemeeter Input and Out B3.
- Hue Bridge: ID, name, software 1978293000, API 1.78.0, event stream Open.
- Stream Deck: serial present.
- Recorder: Stopped. Soundboard: Ready, companion not loaded yet. Media keys: Ready.

The five idle cards matched reality: callback monitor Off; Hue Sync unreachable (Third-party control off); SteamVR and Element not running; ASIO not present (A1 is the SteelSeries output). The user confirmed the badges visually. Automated screenshots of the WebView window are not usable (the GPU surface is not captured), so the accessibility tree was used instead.

## Not yet verified
Task 11: disconnect and reconnect each peer (Voicemeeter, Hue Sync, SteamVR, Stream Deck) and paste a Copy details report.
