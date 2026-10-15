# Audit hardening
## Why
The repository audit ([docs/repo-audit.md](../../../docs/repo-audit.md) and [docs/repo-audit-minor.md](../../../docs/repo-audit-minor.md)) found lower-severity defects that still give wrong behaviour in specific conditions:
- **16:** index-based edits send the revision at click time.
- **17:** media calls have no timeout.
- **18:** a failed build leaves a mixed `bin/`, and an interrupted neural-AEC setup breaks later builds.
- **21:** a clean callback stop reports an uncertain cleanup.
- **22:** the first Stream Deck press after connecting may be swallowed.
- **23:** a second deck that fails to open is never reported.
- **24:** changing Hue room resets the write queue while a write is in flight.
- **25:** any installed extension can evict the browser bridge.

During native acceptance on 2026-10-15, graceful shutdown reached its deadline and
showed only `context deadline exceeded`. The host remained in its error dialog until
the user dismissed it normally. Two Stream Decks were connected, but the error did
not identify the awaited plugin; the underlying cause is unconfirmed.

Findings 14, 15 and 26 are handled elsewhere:
- **14** gets its own change.
- **15** and **26** overlap the planned GUI changes (`gui-action-outcomes`, `gui-state-preservation`) and are resolved there.

## What Changes
- **16:** priority and output edits drawn from an older configuration are sent as stale.
- **17:** each asynchronous WinRT media operation waits at most 2 s; synchronous property reads are unchanged. A timed-out call is cancelled and reported as an error.
- **18:** native companions build into a fresh staging folder and reach `bin/` only after esbuild, Tailwind and the Go build succeed. Preparation and compilation failures preserve deployed files; deployment I/O failures are reported without a rollback guarantee. The only application output remains `bin/snoofer.exe`. Neural-AEC setup repairs a wrong checkout and a partial CMake extraction.
- **21:** `SnooferStop` treats Voicemeeter's "no callback registered" (-2) as clean.
- **22:** keys and dial buttons start released on each connection, so the first valid down report dispatches once. Remove the unsupported synchronous startup query; retain repeated-report deduplication and existing release/generation handling.
- **23:** a deck path that fails to open is reported even while another deck runs; its temporary path report is retired when that path opens or disappears.
- **24:** a Hue light-write result from before a room change is ignored.
- **25:** every second live connection claiming an occupied browser name is refused, including the same origin; after incumbent cleanup, a reconnect receives a fresh generation. This completes the original reject-second-connection option without authenticating the first claimant.
- **Shutdown diagnostic follow-up:** include the plugin being awaited when the host's
  shutdown deadline expires, preserving error identity and prior cleanup failures.
  Keep shutdown order, deadlines, error-dialog behavior and build refusal unchanged.

## Impact
- **Code:** `app/web/app.js`, `internal/mediasessions/native`, `scripts/build*.ps1`, `internal/voicemeeter/callback`, `internal/streamdeck`, `plugins/streamdeck`, `plugins/hue`, `plugins/nowplaying`, `snoofer/plugin.go`.
- **Invariants:** stale-action rejection is strengthened (16). No uncertain command is retried. The user explicitly superseded initial-held-button suppression: the first valid reported down state may activate once even if the button was already held when connected; later duplicate down reports do not repeat it.

## First-key hardware follow-up
On 2026-10-15, the user reconnected the Plus XL, waited for Home, and reported that the first Play press did nothing while the second worked. The read-only probe returned a zero-length key payload on both Plus and Plus XL, including while an XL key was held. The user then explicitly chose immediate first-press delivery over suppressing a button held during connection. Initialize keys and dial buttons as released; remove the unsupported startup query and add no grace period or other timing heuristic. This policy supersedes the conflicting initial-held-button clauses in modular-snoofer and direct-stream-deck; page/layout generation checks and suppression while dark remain unchanged.
