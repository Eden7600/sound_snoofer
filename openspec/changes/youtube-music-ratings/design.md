# Design

## Existing path
`extension/src/page.js` reports media state through `bridge.js` to `background.js`. The background bounds sessions and sends them over the existing localhost bridge. `plugins/nowplaying` captures an immutable target and desired action, rejects stale generations, and publishes controls for the Deck editor. Browser command completion currently means Submitted after a successful socket write; observed state arrives separately.

Reuse that path. Windows media sessions and unsupported websites gain no rating capability. No generic website adapter registry, new OAuth service or transport is needed for one supported site.

## Verified player contract
The approved read-only inspection on 2026-10-15 identified the current top-frame player and rating controls in Brave (Chrome 155 user agent), YouTube Music client `1.20261013.15.00`. [Sanitized observations](observations.md) record the evidence and its limits. No rating or playback mutation was performed; other browser/layout variants and live mutation behavior remain acceptance work.

Support the observed layout only. Require the actual top-frame origin `https://music.youtube.com`, one `#movie_player`, and one `ytmusic-player-bar #like-button-renderer` of type `YTMUSIC-LIKE-BUTTON-RENDERER`. The active session media must be the video inside that player. Read the playing ID from `#movie_player.getVideoData().video_id` and require it to equal the renderer's `data.target.videoId`. The browsing URL and media title are insufficient: users can browse while a different song plays, and recordings can share metadata.

Within that renderer, require exactly one Like button matching `.like-button-wrapper button[aria-pressed]` and one Dislike button matching `:scope > yt-button-shape > button[aria-pressed]`. Both must be enabled and `data.likesAllowed` must be exactly `true`. Require the renderer's `like-status` attribute, `data.likeStatus`, and the two `aria-pressed` values to agree:

| Status in both sources | Like pressed | Dislike pressed | Rating |
| --- | --- | --- | --- |
| `LIKE` | `true` | `false` | `like` |
| `INDIFFERENT` | `false` | `false` | `none` |
| `DISLIKE` | `false` | `true` | `dislike` |

The `LIKE` row was observed on the current player; `INDIFFERENT` occurred elsewhere on the page. The full unrated and disliked rows are expected mappings requiring automated coverage and live validation. Missing, conflicting or malformed fields, both buttons pressed, duplicate controls, a different layout, loading, account prompts or ambiguous player identity mean `unknown` with `canRate: false`. Do not infer state from colors or localized labels. Reject an `ad-showing` player; the inspected `getVideoData()` had no `isAd` field, so its absence is not an authoritative advertisement signal. Fail closed whenever the current item or eligibility cannot be established.

Read only the renderer's target video ID, like status and likes-allowed flag; do not copy tracking parameters, service endpoints, account data or cookies.

YouTube's documented `videos.rate` and `videos.getRating` APIs are an alternative but require an authorized Google API flow. Reusing the user's signed-in page avoids adding that setup to Snoofer. The page integration is a compatibility commitment, not a stable Google API.

References: [rate](https://developers.google.com/youtube/v3/docs/videos/rate), [getRating](https://developers.google.com/youtube/v3/docs/videos/getRating).

## State and wire contract
Add these optional fields to protocol 2 browser sessions and validate them in extension routing and Go decoding:

| Field | Contract |
| --- | --- |
| `track` | Non-empty opaque playing-track token, at most 128 characters; omit when identity is unavailable. |
| `rating` | `unknown`, `none`, `like` or `dislike`; omit for unsupported sessions and older companions. |
| `canRate` | Boolean; true only with a valid track, known rating and verified enabled controls. Missing means false. |
| `reloadRequired` | Optional boolean for the helper upgrade boundary; missing means false. |

Explicit `unknown` identifies supported YouTube Music whose current rating or eligibility cannot be established; it is distinct from observed `none`. Invalid rating fields cannot create actionable capability. Missing fields from older companions leave ratings unavailable without breaking their existing transport.

Only the actual `https://music.youtube.com` top-frame document can publish this capability. The background verifies the runtime port sender's frame/origin instead of trusting the tab's displayed URL or a page-supplied site label. Page code independently checks its origin and current player. Existing message/session limits remain in force.

The track token combines the authoritative playing video ID with its observed playback lifetime within the content-port lifetime. Observed track transitions, player replacement and loss/reacquisition of identity invalidate it, including A-to-B-to-A with unchanged metadata. Browsing or changing the rating without changing the player does not invalidate it. Existing session ID, content-port Instance and connection Generation guards remain in force; a video ID alone is not a lifetime token.

Use a narrow player observer or bounded state check to publish rating changes while paused. Include player replacement and loading transitions; remove stale capabilities promptly. Avoid observing unrelated page content or adding unbounded polling/work.

Publish current control state without waiting for artwork downloads. The existing asynchronous thumbnail path captures a session snapshot before awaiting artwork; older work must not overwrite a newer rating or resurrect a replaced track. Apply artwork only to its still-current key/connection and publish the latest session state. Add a deliberately delayed-artwork regression, including a paused rating change.

## Commands and confirmation
Capture focus generation, browser connection generation, content-port instance, playing-track token and explicit desired rating when a key is pressed. A Like press captures `none` if already liked, otherwise `like`; Dislike is symmetric. Unknown state cannot be toggled. Send command `op: "rate"` with explicit string fields `track` and `rating` (`none`, `like` or `dislike` only), alongside the existing routing identifiers. Do not overload numeric `value`. Validate the operation and fields at every routing boundary.

At the page, synchronously re-read the current player, token, renderer identity, eligibility and rating immediately before acting, with no asynchronous gap before the click. If the desired rating is already observed, do nothing. Otherwise click the desired rating's button once, or click the currently active button once to clear a rating. Never retain a click for a later player or retry it.

Both controls write one rating property. Reserve one pending rating operation per connection, content-port instance and track token. Transient `canRate`, unrelated observed rating or focus changes do not release that reservation; capability can still invalidate a queued action. While unresolved, reject further rating writes for that target with local pending feedback; do not block independent play/pause or seek actions. Keep rating pending/failure state separate from existing transport pending state so neither overwrites the other's confirmation.

Readback matching the captured target and desired rating confirms the state. Use the provider's existing 3-second default observation timeout; expiry reports no confirmation, never an assumed success or automatic retry. A socket write error may be Unknown after possible submission, so retain its reservation until matching readback, actual target retirement or timeout. A definite `errBrowserChanged` rejection requires no reservation. A later observation still updates the displayed actual rating.

Changing song, focus or connection retires the old operation's display context and rejects queued actions. Focus changes alone do not free its target's reservation, including focus away and back. Track or connection retirement ends that target's reservation. Late reports or results cannot alter a replacement target. Reconnection observes only; no queued click is retained. The existing Submitted result remains transport acceptance, not proof that Google saved the rating.

## Controls and presentation
Publish `nowplaying.like` and `nowplaying.dislike` with press operations and the existing focus epoch. Use full labels Like Track and Dislike Track, compact labels Like and Dislike, and distinct code-drawn thumb icons. The selected rating uses the Media area's active fill and an On value; the other uses Off. Unknown supported state is N/A, pending is Wait, and errors use the existing error treatment. Rating state never changes optimistically.

Follow existing media transport visibility: unsupported focused sessions have hidden/inert ratings; an identified YouTube Music session with temporarily unreadable state shows unavailable ratings. The Media on the Deck filter hides both. Do not silently redirect presses to another YouTube Music tab when an unsupported session is focused.

The Deck editor already lists published press controls. Add only its icon mappings and generated vendored icons. Do not add a separate catalogue or change saved model/device layouts. During live acceptance, choose unoccupied positions with the user and bind these controls through the existing editor. Physical acceptance covers Plus and Plus XL; other models keep their as-is support status.

## Extension update behavior
Bump the page/isolated helper installation markers and their internal event names together. Existing v2 page listeners cannot be removed by new closures, so new commands must not share the old command event name. Keep one authoritative content-port lifetime per frame and ignore superseded reports. The old page helper's captured media handlers are also inaccessible to a replacement helper. Require a one-time reload of affected tabs after the extension update rather than promising seamless handler transfer.

Before reload, publish `reloadRequired: true`, suppress page-controlled transport and ratings in both extension routing and provider capabilities, and explain `Reload this tab` through existing session status/failure presentation. The provider currently assumes browser play capabilities, so it must explicitly honor this flag. Browser tab mute can remain available through its independent browser API. Do not add a new GUI flow or advertise unverified page transport. After reload, the page registers its handlers normally and clears the flag. Test old-helper injection for no duplicate action and clean reload for working ratings and existing transport. Increment the package version, document this installation boundary and retain compatible wire protocol 2.

## Verification and rollout
Use deterministic Node tests for page state, routing, paused updates, unsupported/unknown states, explicit desired ratings, song replacement and helper upgrades. Use Go fake-source tests for stale targets, bounded input, overlapping controls, pending/readback, timeout and reconnect. Inspect actual renderer output at native sizes and test editor discovery and icons.

Run focused checks first, then repository Go/vet/race and relevant frontend/extension checks, strict OpenSpec validation and the canonical build. Coordinate browser reload and account-changing live acceptance under the user's desktop-control policy. Record and restore the rating of every exact track changed during live tests and preserve the exact host launch configuration. Record automated and physical evidence separately; do not check off unperformed acceptance.
