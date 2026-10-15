# Correlated GUI action outcomes

## Why
GUI audit items 1 and 4 have one cross-layer cause: the GUI allows one global pending action,
`Desktop.Send` confirms only enqueueing, and delayed errors return through shared notices or
provider status. A second valid click can be discarded, another control's revision can clear
the first action's pending state, and an error can appear on a different screen. Simply replacing
the global flag with per-control flags would still reject rapid audio edits because they share
one worker revision.

## What Changes
- Give each GUI action a session-local unique ID and one correlated result from `Desktop.Send`.
  Use a bounded, noncoalescing reply lane separate from presentation state; preserve every
  accepted action's result without retaining a result history.
- Distinguish Submitted, Processed, Rejected, Failed and Unknown. Keep provider observations and
  reconciliation status separate; enqueueing, a setter return, a revision change and elapsed
  time never establish observed success.
- Track pending actions and feedback at their originating control and screen. Allow unrelated
  actions concurrently, reject busy groups explicitly, and never replay an uncertain action.
- Give only the two explicit global Mic Mute and Playback Mute preferences independent worker
  generations. Preserve registry revisions, saved-choice compare-and-swap and every contextual
  audio action's existing strict revision/target checks.
- Capture mutable provider targets before enqueueing, report worker validation/save/native-call
  errors where a completion boundary exists, and explicitly report Submitted where existing
  coalescing or remote reconciliation cannot produce an individual completion.
- Give host enablement proposals identities and owners so confirm/cancel cannot affect a newer
  proposal. Preserve local confirmation tickets from `gui-state-preservation`.

## Impact
Private GUI/pipe types and transport, registry dispatch's optional in-process completion path,
audio actions/worker, the listed provider adapters, shared frontend action state/widgets and
focused tests. Update the UI contract for local action feedback. Keep control IDs, persisted
settings, routing/recording invariants and existing Stream Deck command semantics. No new
dependencies, durable job system, automatic action retries or general audio transaction layer.

This proposal authorizes planning only. Implementation follows review and commit of these
artifacts. Run isolated checks at each checkpoint; the coordinating agent schedules canonical
build/restart, native smoke and live acceptance after stable source checkpoints. The user has
lifted the temporary running-app restriction. The user-owned desktop smoke script is not changed.
