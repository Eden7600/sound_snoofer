# Design

## Existing path
`extension/src/page.js` reports media state through `bridge.js` to `background.js`. The background bounds sessions and sends them over the existing localhost bridge. `plugins/nowplaying` captures an immutable target and desired action, rejects stale generations, and publishes controls for the Deck editor. Browser command completion currently means Submitted after a successful socket write; observed state arrives separately.

Reuse that path. Windows media sessions and unsupported websites gain no rating capability. No generic website adapter registry, new OAuth service or transport is needed for one supported site.

## Feasibility checkpoint
With a separately approved browser inspection, identify the current top-frame YouTube Music player, its authoritative playing video ID, rating buttons and observed rating. The browsing URL and media title are not sufficient: users can browse while a different song plays, and different recordings can share metadata. Record a small sanitized fixture and the tested browser/extension versions. Inspection must not change playback or account ratings.

Confirm the adapter can distinguish unknown, unrated, liked and disliked states, and disabled controls, account prompts, loading and advertisements. If it cannot establish the playing content and actionable state, report unavailable. Do not infer a rating from button colors or localized action text alone. Update this design with the verified selectors and identity source before code changes.

YouTube's documented `videos.rate` and `videos.getRating` APIs are an alternative but require an authorized Google API flow. Reusing the user's signed-in page avoids adding that setup to Snoofer. The page integration is a compatibility commitment, not a stable Google API.

References: [rate](https://developers.google.com/youtube/v3/docs/videos/rate), [getRating](https://developers.google.com/youtube/v3/docs/videos/getRating).

## State and wire contract
Add optional fields to protocol 2 browser sessions: a bounded playing-track token, `canRate`, and observed `rating` (`none`, `like`, `dislike`, `unknown`). Omitted rating means unsupported or an older companion; explicit `unknown` means a verified YouTube Music player whose rating cannot currently be read. `canRate` is false unless identity, known rating and enabled controls are established. Validate bounds and enumerations both in extension routing and Go decoding. Missing fields from older extensions leave ratings unavailable without breaking existing transport.

Only the actual `https://music.youtube.com` top-frame document can publish this capability. The background verifies the runtime port sender's frame/origin instead of trusting the tab's displayed URL or a page-supplied site label. Page code independently checks its origin and current player. Existing message/session limits remain in force.

The track token identifies the authoritative playing item and its observed playback lifetime within the content-port lifetime. Track transitions invalidate it even when metadata is unchanged or playback later returns to the same item. Browsing without changing the player does not invalidate it. Do not expose account identifiers or cookies.

Use a narrow player observer or bounded state check to publish rating changes while paused. Include player replacement and loading transitions; remove stale capabilities promptly. Avoid observing unrelated page content or adding unbounded polling/work.

Publish current control state without waiting for artwork downloads. The existing asynchronous thumbnail path captures a session snapshot before awaiting artwork; older work must not overwrite a newer rating or resurrect a replaced track. Apply artwork only to its still-current key/connection and publish the latest session state. Add a deliberately delayed-artwork regression, including a paused rating change.

## Commands and confirmation
Capture focus generation, browser connection generation, content-port instance, playing-track token and explicit desired rating when a key is pressed. A Like press captures `none` if already liked, otherwise `like`; Dislike is symmetric. Unknown state cannot be toggled. At the page, re-read the current player and track immediately before the click, with no asynchronous gap between validation and action. If the desired rating is already observed, do nothing.

Both controls write one rating property. While a rating write is unresolved, reject further rating writes for that target with local pending feedback; do not block independent play/pause or seek actions. Keep rating pending/failure state separate from the existing session transport pending state so neither operation overwrites the other's confirmation. Readback matching the captured target and desired rating confirms the state. A bounded timeout reports no confirmation, never an assumed success, and never retries automatically. A later observation still updates the displayed actual rating.

Changing song, focus or connection retires the old operation's display context and rejects queued actions. Late reports or results cannot alter a replacement target. Reconnection observes only; no queued click is retained. The existing Submitted result remains transport acceptance, not proof that Google saved the rating.

## Controls and presentation
Publish `nowplaying.like` and `nowplaying.dislike` with press operations and the existing focus epoch. Use full labels Like Track and Dislike Track, compact labels Like and Dislike, and distinct code-drawn thumb icons. The selected rating uses the Media area's active fill and an On value; the other uses Off. Unknown supported state is N/A, pending is Wait, and errors use the existing error treatment. Rating state never changes optimistically.

Follow existing media transport visibility: unsupported focused sessions have hidden/inert ratings; an identified YouTube Music session with temporarily unreadable state shows unavailable ratings. The Media on the Deck filter hides both. Do not silently redirect presses to another YouTube Music tab when an unsupported session is focused.

The Deck editor already lists published press controls. Add only its icon mappings and generated vendored icons. Do not add a separate catalogue or change saved model/device layouts. During live acceptance, choose unoccupied positions with the user and bind these controls through the existing editor. Physical acceptance covers Plus and Plus XL; other models keep their as-is support status.

## Extension update behavior
Bump the page/isolated helper installation markers and their internal event names together. Existing v2 page listeners cannot be removed by new closures, so new commands must not share the old command event name. Keep one authoritative content-port lifetime per frame and ignore superseded reports. The old page helper's captured media handlers are also inaccessible to a replacement helper. Require a one-time reload of affected tabs after the extension update rather than promising seamless handler transfer. Before reload, missing capabilities remain unavailable and the installation status must explain the reload requirement; never advertise unverified transport capability. After reload, the page registers its handlers normally. Test old-helper injection for no duplicate action and clean reload for working ratings and existing transport. Increment the package version, document this installation boundary and retain compatible wire protocol 2.

## Verification and rollout
Use deterministic Node tests for page state, routing, paused updates, unsupported/unknown states, explicit desired ratings, song replacement and helper upgrades. Use Go fake-source tests for stale targets, bounded input, overlapping controls, pending/readback, timeout and reconnect. Inspect actual renderer output at native sizes and test editor discovery and icons.

Run focused checks first, then repository Go/vet/race and relevant frontend/extension checks, strict OpenSpec validation and the canonical build. Coordinate browser reload and account-changing live acceptance under the user's desktop-control policy. Record and restore the rating of every exact track changed during live tests and preserve the exact host launch configuration. Record automated and physical evidence separately; do not check off unperformed acceptance.
