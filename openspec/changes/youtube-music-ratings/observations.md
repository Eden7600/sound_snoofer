# Read-only YouTube Music observations

Approved inspection on 2026-10-15 used Brave with a Chrome 155 user agent and YouTube Music client `1.20261013.15.00`. The document was top-level at `https://music.youtube.com`. This probe changed neither playback nor account ratings. The companion version was not captured by this DOM inspection.

| Observed source | Sanitized result |
| --- | --- |
| Player toolbar | `YTMUSIC-PLAYER-BAR` |
| `ytmusic-player-bar #like-button-renderer` | `YTMUSIC-LIKE-BUTTON-RENDERER`, `like-status="LIKE"` |
| Renderer `data.target.videoId` | Equal to `#movie_player.getVideoData().video_id`; actual ID omitted |
| Renderer `data.likeStatus`, `data.likesAllowed` | `LIKE`, `true` |
| Renderer `.like-button-wrapper button[aria-pressed]` | One button, pressed `true` |
| Renderer `:scope > yt-button-shape > button[aria-pressed]` | One button, pressed `false` |
| Player | Paused (`getPlayerState()` returned 2), video element present |
| Advertisement marker | `ad-showing` class absent; `getVideoData().isAd` absent |

The renderer data contained `target`, `likeStatus`, `trackingParams`, `likesAllowed` and `serviceEndpoints`. The adapter needs only target video ID, like status and likes-allowed. No tracking, endpoint or account payload is retained here. Button labels read Like and Dislike, but selectors must not depend on localized labels.

`LIKE` and `INDIFFERENT` statuses occurred among page renderers. Only the current player's liked state and corresponding pressed bits were verified together. `DISLIKE` was not observed. The expected `INDIFFERENT`/both-false and `DISLIKE`/dislike-only mappings remain subject to automated tests and live acceptance.

This evidence supports only the inspected layout and identity sources. It does not establish mutation success, clearing/switching behavior, paused manual updates, account-prompt/ad variants, another browser, extension upgrades or physical Deck behavior. Those checks remain open. Fail closed when identity, eligibility or the agreeing state sources required by the design cannot be established. Personal URLs, track IDs/titles and account metadata are intentionally omitted.


## Connected-browser recheck — 2026-10-16
The user reported that pressing Like changed the key to N/A. After connecting the
ChatGPT browser extension in Brave, a read-only inspection successfully read the
current player: one expected renderer, live `like-status=LIKE`, Like pressed
`true`, Dislike pressed `false`, and both buttons enabled. No rating or playback
was changed. The browser read-only DOM scope did not expose page-world
`renderer.data`, so it did not establish the current value of `data.likeStatus`.

The unit fixture previously updated `data.likeStatus` together with every live
button change. Holding that auxiliary value fixed reproduces unknown/N/A after
a valid rating transition. The corrected contract trusts agreeing live status
and pressed bits, retaining target identity and eligibility checks. This is a
regression for that failure mode; real-account mutation acceptance is still open.
