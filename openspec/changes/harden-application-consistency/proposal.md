## Why

Sound Snoofer already has a useful architecture and substantial regression coverage. Its highest-value improvements are correcting cross-layer inconsistencies, making observed state trustworthy on every control surface, and validating lifecycle behavior. A rewrite or new feature framework would increase risk without addressing these problems.

This change records the audit of commit `22d71ae` and the user-authorized implementation on 2026-10-05. The user's pre-existing AGENTS.md and personal configuration are preserved. See verification.md for executed checks and the remaining physical acceptance.

## What Changes

- Correct config-adjacent mute persistence and configuration/picker eligibility mismatches.
- Make Mic Off cover the configured VR strip consistently in planning, execution, and presentation.
- Bound Windows-default enforcement by the current live permission and profile lifetime.
- Reject stale disruptive actions and discard undispatched hardware actions on disconnect.
- Share the small pieces of control semantics that currently differ between TUI and Stream Deck.
- Improve compact-terminal usability, action explanations, error persistence, and observation freshness.
- Add focused cross-boundary regression tests and a repeatable local validation command using existing tools.
- Reconcile contradictory documentation and OpenSpec requirements without erasing unperformed hardware acceptance.

## Capabilities

### New Capabilities

- `application-consistency`: Cross-surface consistency, lifecycle correctness, and validation requirements for the existing application.

### Modified Capabilities

None in the canonical spec directory. Existing behavior is described by active changes; the design lists explicit overlaps and supersession work before implementation.

## Impact

Primary packages: `internal/cli`, `internal/config`, `internal/control`, `internal/controller`, `internal/routing`, `internal/tui`, `internal/desktop`, `internal/streamdeck`, and `internal/windowsaudio`. Native observation optimization, if measured worthwhile, also touches `internal/voicemeeter`. No new runtime dependency is planned.

Implement the independently reviewable work packages in design.md in order. Existing routing and recorder invariants remain mandatory. Named soundboard clips, new playback-target workflows, automatic stall detection, GUI replacement, automatic rollback, and new audio capture are excluded.
