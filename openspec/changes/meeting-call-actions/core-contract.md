# Focused meeting core adapter contract

The existing Meetings owner retains universal Mute/Camera unchanged. Focused
Deafen/Share/Leave are explicit one-shot actions and never reconciled or retried.

```go
type MeetingAction string
const (
    MeetingDeafen MeetingAction = "deafen"
    MeetingShare MeetingAction = "share"
    MeetingLeave MeetingAction = "leave"
)
// MeetingPeer additionally contains:
// Deafen, Share, Leave MeetingValue

type MeetingActionCommand struct {
    Context context.Context
    Peer, Lifetime string
    Action MeetingAction
    Revision uint64
    Value bool
    Complete Completion
}
func (a *MeetingAdapter) NextAction() (MeetingActionCommand, bool)
```

The adapter reports each action's exact Revision and optional Identity, then
claims work after Report/Changed. Recheck Context, Peer, Lifetime, Revision and
current availability immediately before I/O; refuse obsolete work. Deafen needs
Known state and captures its opposite. Share captures the opposite if Known,
otherwise true (open the native presentation workflow). Leave captures true.
An available Share/Leave may be unknown, but every available action needs a
nonzero Revision. No available action is inferred from mute/camera capability.

Complete every claimed command exactly once. Processed requires the action's
observed boundary; opening a presentation menu is Submitted. Definite unsent
refusal is Rejected; uncertain side effects are Unknown. Do not return Submitted
then later complete again. The core owns a five-second deadline and cancellation,
reserves at most one action of each kind, and never retries automatically.

Focus selects the newest first-observed joined lifetime in host receipt order.
An explicit focus button pins that exact lifetime until it disappears. Heartbeats
do not reorder calls. Runtime focus slot IDs are meetings.focus-N, collected under
meetings.focus, stable for an active lifetime and guarded by a fresh epoch on
reuse. App/browser-tab labels contain no meeting names, codes or participants.

Core controls meetings.deafen, meetings.share and meetings.leave capture exact
focus epoch, peer lifetime and action observation. Selection/lifetime changes
cancel unsent work; claimed work stays reserved until terminal completion or its
bounded deadline. A claimed Leave retains its exact result waiter only after
observed departure or captured lifetime replacement, because departure can precede
that correlated result. Changing focus while the captured call is still joined
in the same lifetime cancels its context, even after claim, so provider work
waiting to send remains cancelable. Adapter and host shutdown cancel it too.
Callbacks run outside the coordinator lock. Preview cannot
submit focus or action changes. Per-app explicit controls remain provider-local.


Pending preserves actual capability availability and carries Status Pending, so
native keys show Wait; the captured request guard refuses duplicate operations.
An available Share with unknown presentation state reads Ready, describing its
workflow capability rather than inventing observed Off or On. Unsupported or
absent calls retain N/A and are unavailable.
A submitted Share workflow instruction is tied to its captured call lifetime,
control revision and requested state. Keep it through unchanged or unknown
reports; clear it once fresh known presentation readback matches that request,
even when readback precedes Submitted. New focus/call lifetimes cannot inherit
it. This typed submission note never clears an Unknown or Failed action result.
