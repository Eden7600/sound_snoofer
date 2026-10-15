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

Findings 14, 15 and 26 are handled elsewhere:
- **14** gets its own change.
- **15** and **26** overlap the planned GUI changes (`gui-action-outcomes`, `gui-state-preservation`) and are resolved there.

## What Changes
- **16:** priority and output edits drawn from an older configuration are sent as stale.
- **17:** every WinRT media call waits at most 2 s. A timed-out call is cancelled and reported as an error.
- **18:** native companions build into a staging folder and reach `bin/` only after the Go build succeeds. Neural-AEC setup repairs a wrong checkout and a partial CMake extraction.
- **21:** `SnooferStop` treats Voicemeeter's "no callback registered" (-2) as clean.
- **22:** a deck session reads the current key state when it opens, so the first report is compared with real state.
- **23:** a deck path that fails to open is reported even while another deck runs.
- **24:** a Hue light-write result from before a room change is ignored.
- **25:** a newer bridge connection replaces a live one only when it comes from the same extension origin.

## Impact
- **Code:** `app/web/app.js`, `internal/mediasessions/native`, `scripts/build*.ps1`, `internal/voicemeeter/callback`, `internal/streamdeck`, `plugins/streamdeck`, `plugins/hue`, `plugins/nowplaying`.
- **Invariants:** unchanged. Stale-action rejection is strengthened (16). No uncertain command is retried.
