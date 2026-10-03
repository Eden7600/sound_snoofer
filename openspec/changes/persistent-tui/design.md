# Design

## Context

The Go CLI already owns serialized controller steps, verified writes, and writer locking. Voicemeeter refresh calls must remain on one OS thread. Routing may wait for readback, so it cannot execute in the UI event loop.

## Goals / Non-Goals

Goals: persistent monitoring, responsive navigation, explicit live control, and reuse of controller semantics.
Non-goals: config editing inside the UI, startup services, tray integration, or changing routing rules.

## Decisions

Use pinned Bubble Tea v2 for alternate-screen rendering, keyboard input, resizing, and terminal restoration. A dedicated actor locks its OS thread and owns login, polling, controller, and writer ownership. Immutable state snapshots cross a bounded channel; obsolete frames are coalesced. Event history retains 50 entries. The UI sends bounded commands and remains responsive during verification. Mode/reload requests wait until the current operation finishes; quit cancels controller waits immediately. No goroutine except the actor calls the DLL.

Reload validates before replacing config, then reconstructs the controller to reset debounce. DLL-load failures retry every five seconds; engine errors use existing controller backoff. Snapshot-derived views show attention/staleness on errors. Device names and error text are stripped of terminal controls. CLI tui requires interactive stdin/stdout and rejects JSON mode.

## Risks / Trade-offs

- A mode change is acknowledged only after the current operation; UI displays queued status.
- Windows console behavior varies. Verify alternate-screen restoration using a real PTY smoke test, alongside deterministic model/worker tests.
- Existing hardware routing acceptance remains pending independently of this UI change.
