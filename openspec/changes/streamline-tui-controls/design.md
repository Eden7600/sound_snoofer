# Design

## Context

See proposal.md. The source of the requested changes is the user's annotated
dashboard screenshot supplied on 2026-10-03. Current code in dashboard.go reserves
a title row, an unconditional notice row and a second contextual footer. rules.go
cycles Source immediately and interleaves recording settings/actions. tui.go
binds 0 directly to Off. The worker already serializes revisioned edits and saves
before applying. Existing dashboard tests assert the old shortcut and cycling.

## Goals / Non-Goals

Implement the screenshot annotations with a compact layout and explicit source
selection. Keep current routing, logical source identities, saved schema and
transport safeguards. No new mouse support, hardware-device picker, extra source
discovery, bus policy changes or broad redesign of unrelated tabs is required.
The annotation requests a list for Source; Processing, Monitor and stage keep
their current cycling interaction.

## Decisions

### Compact screen chrome

Start the screen with the existing mode/connection/recorder status row and tabs;
remove the app-title row (the terminal window title can remain Sound Snoofer).
Show a notice row only when there is a meaningful pending operation, transient
result or error. Remove the default 'Choose a control...' filler. Use one keyboard
footer, updating its content while the source picker is open. Remove the second
context-help/mic-status footer and the side-panel exit reminder. Preserve the
Signal Status panel, B1/B2/B3 legend, L/R/X help and responsive single-column mode.

Suggested ordinary layout (illustrative values):

```text
LIVE · Connected · Recorder: Stopped
[Controls]  Routing  Devices  Events
╭ CONTROLS ────────────────────────┬ SIGNAL STATUS ───────────╮
│ MICROPHONE                      │ Microphone               │
│ Source                    Off ▾ │ Off · sends disconnected │
│ Processing               Direct │                          │
│ Monitor                     Off │ Playback output          │
│                                 │ A1 · SteelSeries Arena 9 │
│ COMPUTER AUDIO                  │                          │
│ Playback virtual:1           On │ Native recorder          │
│                                 │ Stopped                  │
│ RECORDING                       │                          │
│ Record Computer Audio       Off │ QUICK HELP               │
│ Record Microphone           Off │ L  Live / preview        │
│ Recording Mic Stage         Pre │ R  Reload                │
│                                 │ X  Reset choices         │
│ ACTIONS                         │                          │
│ Start Recording                 │ B1 Recording             │
│ Stop Recording                  │ B2 Element · B3 Apps     │
╰─────────────────────────────────┴──────────────────────────╯
Tab views · ↑↓ select · Enter/Space change · L live · Q quit
```

### Source selection is a draft until confirmed

Use a local picker state holding highlighted source and opening revision. Enter
or Space on Source opens a bounded list anchored near the row, falling back to
a centered list when there is insufficient space. Options are Desk, Lav, Webcam,
Off, with the saved choice marked and initially highlighted. These are logical
preferences, not raw device names. Keep all four choices available when devices
are absent; normal fallback rules decide the effective mic after confirmation.

Up/Down and j/k move the highlight, clamped at the ends. Enter commits once through
the existing editRule/source worker path and closes the list. Escape cancels;
Tab cancels and changes view. Q keeps its existing quit behavior. While open,
Space does not commit and unrelated edit/live/recording shortcuts do not dispatch.
Confirming the unchanged choice closes without saving or submitting work.

Opening, browsing, cancelling, resizing and ordinary device refreshes never save
or write audio. If a new state revision makes the draft stale, close the picker
and show 'Choices changed; reopen Source'. Keep worker revision checking as a
second guard. Device observation updates without an intent revision refresh the
background without resetting the highlighted option. An unavailable or ambiguous
device never silently changes the highlighted logical choice.

Remove key 0 from key handling, quick help, all footers (including tiny-terminal
fallback), tests and README. Off remains accessible through the same source list.

### Settings and actions have different presentation

Keep stable row keys for saved choices and worker actions. Reorder recording rows:
record-computer, record-mic, record-tap, then a nonselectable blank line and ACTIONS
heading, then record-start, record-stop. Render labels exactly as in the mockup.
Only the user-facing label changes from 'Recording mic tap' to 'Recording Mic
Stage'; retain recording.mic_tap in JSON. Render Pre and Post without explanatory
suffixes, including Monitor values for consistent compact presentation.

Headers and spacers are render-only and cannot receive keyboard focus. Start/Stop
remain Enter-only commands; Space never issues transport. Hidden recording profile
means no recording settings or recording-actions group. Recalculate selection to
rendered-row mapping after reorder and resize; preserve focus by row key on reload.

### Concise notices that reflect current state

Replace presentation strings or use typed notice state; do not derive freshness
by parsing arbitrary error text. Use these compact ordinary messages:

| Situation | Notice |
| --- | --- |
| Queued worker action | Queued |
| Saved in preview | Saved · Preview |
| Saved, live routes not yet converged | Saved · Pending |
| Verified live routing | Applied |
| Reload completed | Reloaded |

Successful transient results disappear after 3 seconds. Pending remains until
resolved, failed or superseded, and becomes Applied only from current successful
readback with no pending changes or unresolved routes. Preview never becomes
Applied. Errors persist until recovery or explicit successful replacement and
outrank success notices. Keep a concise actionable error summary on screen and
the full diagnostic in Events; indicate truncation. Do not clear an error merely
because a success timer fires. Use deterministic clock/message tests for expiry.
Update queued/saved notices at their producer sites as well as dashboard rendering;
avoid leaving the live screenshot's obsolete dry-run explanation in worker state.

## Risks / Trade-offs

- Picker can become stale while the worker runs → preserve revision checks and
  cancel stale drafts rather than sending an edit with old assumptions.
- Variable chrome height and render-only separators can hide selection → test
  selected-row mapping at small/large sizes with notices and the picker open.
- Shorter messages could hide failures → retain error priority and full Events
  detail; keep mode visible independently of notices.
- Earlier dashboard spec requires the removed shortcut → this change explicitly
  supersedes that clause and immediate source cycling; consolidate during archive.

## Migration Plan

No config or sidecar migration. Preserve source Off behavior and all current
preferences. Update UI tests/docs, build a replacement binary and smoke-test an
isolated preview profile. Separate UI acceptance from unperformed hardware/audio
acceptance; this presentation change does not complete earlier recording tests.
