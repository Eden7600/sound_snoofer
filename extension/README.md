# Snoofer Media

A browser extension that lets [Snoofer](../README.md) show and control the media playing in each tab. Browsers give Windows a single media session for all their tabs. With this extension, every tab or embedded player that has media becomes its own session in Snoofer, on the Stream Deck and in its Media screen.

It talks only to Snoofer on the same computer (`ws://127.0.0.1:47815`), with no pairing or account. Nothing leaves the machine.

## Develop

```sh
npm test          # logic and build tests
npm run build     # dist/chrome and dist/firefox
npm run package   # also dist/snoofer-media-<browser>-<version>.zip
```

- **Chrome, Brave, Edge:** open `brave://extensions` (or `chrome://extensions`), turn on Developer mode, choose **Load unpacked** and pick `dist/chrome`. After rebuilding, press the extension's reload button.
- **Firefox (128 or later):** open `about:debugging#/runtime/this-firefox`, choose **Load Temporary Add-on** and pick `dist/firefox/manifest.json`. Then click the toolbar button and **Allow on all sites**; Firefox makes site access optional.

Protocol 2 requires a matching Snoofer version; the popup explains which side needs updating. After upgrading to 1.2.0, reload already-open media tabs once. Snoofer shows **Reload this tab** and leaves page playback and rating commands unavailable until then; tab mute remains available. The new page helper cannot recover the old helper's captured handlers. Reloading lets the page register them again without duplicate commands.

On YouTube Music, bind **Like Track** and **Dislike Track** in the Deck editor. They follow Snoofer's selected media session. Press an inactive rating to select it, or the active rating to clear it. The keys show the observed page state, including changes while paused; unknown or unsupported player state cannot be rated. This uses the signed-in page's current player controls, so website changes can make ratings unavailable. No Google API setup is needed.

Version 1.3.1 supports shared meeting Mute and Camera with a compatible host. Enable Google Meet in Snoofer and bind the Meetings controls through the Deck editor. Changes in any compatible joined call update the shared target and propagate to the other calls; the selected Meet tab controls diagnostics only. New eligible calls inherit current targets, including camera On or mute Off. Mute follows Snoofer's saved Mic Mute preference when audio is available; camera is discarded when no eligible calls remain. Uncertain writes are not blindly replayed after reconnection. Browser output mute remains separate.

The observed Brave layout is the initial supported candidate; prejoin and ambiguous, hidden or unavailable controls cannot be changed. Website changes or a suspended tab can make controls unavailable. Version 1.3.1 adds a control-node identity observation so replacing a page button cannot masquerade as a local mute/camera change. Older helpers without identity remain readable but cannot participate in universal synchronization; reload existing Meet tabs after upgrading.

Meet uses an isolated extension script and sends only bounded call/control state and local tab identity to Snoofer, without meeting URLs, codes, titles, participants or media. Existing permissions are unchanged. New and old host/extension combinations remain Media-compatible on protocol 2; Meet requires both sides to support capability 1. Reload existing Meet tabs after updating. No separate account or pairing setup is added; the localhost connection retains its existing same-user local-software trust boundary.

This companion excludes exact `https://meet.google.com` frames from generic Media reports and transport, so call media does not become a competing music target or expose its title through that path. Other sites, Windows media sessions and older companions keep their existing behavior.

`../scripts/check-extension.cjs` checks the page script in Chrome. It also runs the built extension end to end in Brave against a stand-in bridge.

## Layout

| File | Role |
|---|---|
| `src/page.js` | Page-world content script: finds `<audio>`/`<video>` and the page's media session, reports state and runs commands. |
| `src/bridge.js` | Isolated-world content script: relays between `page.js` and the background. |
| `src/meet.js` | Isolated-world Meet adapter: reads joined-call controls and confirms one explicit action through its current runtime port. |
| `src/background.js` | Keeps the WebSocket to Snoofer, merges reports into sessions, sends artwork and runs commands. |
| `src/logic.js` | Pure helpers shared by the background and popup, tested with Node. |
| `src/popup.*` | Toolbar popup: connection status, site access (Firefox) and the Snoofer port. |
| `build.mjs` | Generates each browser's manifest and packages the zips. |

## Publish

- **Version:** set it in `package.json`, then run `npm run package`.
- **Chrome Web Store:** upload `dist/snoofer-media-chrome-<version>.zip`.
  - **Single purpose:** show and control tab media from the Snoofer desktop app.
  - **Permissions:**
    - `tabs` reads titles and mutes tabs;
    - `storage` keeps the port;
    - `scripting` reaches tabs already open at install;
    - host access runs the media script on pages and fetches artwork.
- **Firefox Add-ons:** upload `dist/snoofer-media-firefox-<version>.zip`.
  - **Add-on ID:** `snoofer-media@sound-snoofer` (`GECKO_ID` in `build.mjs`). It is permanent after the first upload; change it before then if you want another.
  - **Data collection:** none is declared, because data goes only to the local Snoofer app.
- **Protocol:** the message format is `PROTOCOL` in `src/logic.js` and `protocol` in `internal/browserbridge/bridge.go`. Change both together. Snoofer refuses a mismatch, and the popup says which side to update.

Version 1.4.0 adds guarded Leave for the focused Meet call. Snoofer provides call
focus buttons for one-shot actions; newest observed calls are selected by default,
and explicit focus is retained until that call ends. Reload an already-open Meet
tab after upgrading. Meet Deafen is N/A. Meet presentation remains unavailable
until the site-specific Share adapter is inspected and implemented; the shared
action transport does not itself establish screen-sharing support.
