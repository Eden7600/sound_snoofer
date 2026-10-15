# YouTube Music ratings

## Why
The user wants Stream Deck Like and Dislike buttons that show the current song's actual rating. Snoofer already has browser media sessions, captured media actions and a companion extension; ratings belong on that path.

## What Changes
- Add bindable Like and Dislike controls for YouTube Music, following Snoofer's selected media session like its existing transport keys.
- Pressing an inactive rating selects it; pressing the active rating clears it. Selecting the other rating replaces the first.
- Read the playing song and rating from the YouTube Music player, including while paused. Keep unknown state distinct from an unrated song.
- Carry the captured song identity to the page and check it again immediately before acting. Never send a delayed rating to a different song or replay it after reconnect.
- Add hardware icons and Deck editor previews using the existing rendering conventions. Preserve saved layouts; bind the controls during acceptance without overwriting existing assignments.

## Scope and Decisions
The initial target is the focused media session, matching existing transport. This default was stated while the optional target question remained unanswered; amend it if the user chooses to always target YouTube Music. No target-selection preference is added merely to support both possibilities.

First release covers the two Stream Deck controls and their setup/diagnostics. Additional buttons on Media cards, bulk ratings, playlist management, default-layout redesign and a separate Google OAuth flow are outside this change.

## Impact
Extension page/bridge/background logic and tests; Now Playing wire/session/action/publication code and tests; Stream Deck icons and renderer checks; Deck editor icon mapping; extension packaging/install documentation; UI contract vocabulary. No new dependencies or browser permissions are planned. The YouTube Music DOM adapter remains subject to website changes and must be verified against the current player before its selectors are implemented.
