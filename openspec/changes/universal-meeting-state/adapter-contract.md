# Universal meeting Go API contract

The narrow coordinator lives in package `snoofer`, so its controls and completions do not introduce
an import cycle. `Services.Meetings *Meetings` is shared by audio, Discord and
Google Meet. Host/app construction owns one instance and bounded shutdown. It is
not another optional plugin and adds no dependency to Google Meet.

## Adapter contract

```go
type MeetingProperty string
const (
    MeetingMute MeetingProperty = "mute"
    MeetingCamera MeetingProperty = "camera"
)

type MeetingValue struct {
    Identity uint64 // Control node incarnation; changes establish a baseline.
    Known bool
    Value bool
    Available bool
    Revision uint64 // Adapter's captured property revision, including ABA.
}

type MeetingPeer struct {
    ID string       // Stable local slot: "discord", or browser + tab for Meet.
    Lifetime string // Exact transport/document/call lifetime; never a label.
    Label string    // Concise diagnostic identity, no meeting title/code.
    Joined bool     // Eligible joined call; this determines camera membership.
    NewCall bool    // A definite newly observed call, not reconnect metadata.
    Mute MeetingValue
    Camera MeetingValue
}

type MeetingCommand struct {
    Context context.Context // Canceled on superseding intent/peer loss/stop.
    Peer string
    Lifetime string
    Property MeetingProperty
    Revision uint64         // Exact property revision captured by coordinator.
    Value bool              // Explicit desired state, never blind toggle.
    TargetRevision uint64   // Shared intent identity.
    Complete Completion     // Exactly once; observed success is Processed.
}

func (m *Meetings) Attach(ctx context.Context, provider string) (*MeetingAdapter, error)
// provider is a unique valid provider ID; one live attachment each.
func (a *MeetingAdapter) Report(peers []MeetingPeer) error
// Complete fresh current peer list. Ownership is copied. No transport I/O.
func (a *MeetingAdapter) Changed() <-chan struct{}
// Capacity-one wake hint: inspect current state; never treat as an event history.
func (a *MeetingAdapter) Next() (MeetingCommand, bool)
// Claim current unsent work; returns false when none. At most one pending per
// exact peer/property. Provider calls after Report and after Changed wakes.
func (a *MeetingAdapter) Close()
// Removes exactly this attachment, cancels its commands; nonblocking, idempotent.
```

Meet supplies a nonzero `Identity` independently of value/availability revisions.
An identity change establishes a baseline. Providers without replaceable control
nodes may retain zero; Meet observations lacking identity are unavailable for
universal synchronization while their observed values remain readable.

Each adapter reports actual value changes as its worker receives them, not a
coalesced periodic summary that can omit an On→Off→On sequence. Repeated samples
are harmless; availability or revision changes alone are not external intent.
Report disconnected/expired peers absent immediately. A peer whose controls are
unknown can remain Joined: an unreadable button does not prove the call ended.
Only fresh, authoritatively joined peers count; stale/disconnected peers do not.

The provider remains sole owner of its transport and current call state. Before
sending a claimed command it checks Context, exact ID/Lifetime and Revision,
known/available state, and current value. Already satisfied commands complete
Processed without I/O. Definite refusal is Rejected; possible side effect without
readback is Unknown. Discord TOGGLE_VIDEO requires a known differing current
value and all those checks. Meet uses its existing explicit boolean bridge
command. Discord mute while deafened is unavailable for adoption/writes; the
synthetic mute must not become intent. Complete only after the appropriate
readback. An RPC/socket acknowledgement alone leaves this command pending; do not
call Complete with Submitted and then try to complete it again. Its one terminal
completion is Processed, Rejected, Failed or Unknown as supported by the evidence.

NewCall is an edge signal backed by a definite no-call→joined or different-call
transition while observations are authoritative. First reports after transport
reconnect are false. The initial peer has no uncertainty to suppress and can
receive current intent without claiming that reconnect proves a new call.
ID stays stable across reconnect so an uncertain operation cannot be retried
merely because Lifetime changes. A fresh distinct call can clear that inhibition.
Peers retained with NewCall=true must not repeatedly reset inhibition: only a
different Lifetime paired with this evidence is a new-call edge.

## Shared actions and state

```go
type MeetingTarget struct {
    Known bool
    Value bool
    Revision uint64
    Status string // "", "Pending", "Not confirmed"; targets are not app readback.
}
type MeetingSnapshot struct {
    Mute MeetingTarget
    Camera MeetingTarget
}
func (m *Meetings) Snapshot() MeetingSnapshot
func (m *Meetings) Request(ctx context.Context, property MeetingProperty,
    revision uint64, value bool, complete Completion) error
```

The coordinator publishes `meetings.mute` and `meetings.camera`. Their press
handlers capture Snapshot target identity and explicit opposite desired value.
Existing adapter control handlers that now represent universal actions use the
same Request; they do not send directly and then ask the coordinator to catch up.
Per-app observations remain truthful diagnostics. Source mic state and device
status stay distinct from shared target values and per-app confirmation.

Request completion establishes target acceptance (and mute save when audio is
present), not successful application everywhere. Pending/Not confirmed remains
in the shared snapshot while peer readback is unresolved/uncertain. A target
change cancels unsent old work. Its late completion/readback cannot create a new
intent or confirm a newer target. Matching expected readback is acknowledgement,
not an external edit. Uncertain writes retain inhibition across heartbeat and
reconnect; release it on a new shared intent or a definite new call, not timeout.

The coordinator serializes observations, target edits, audio saves and completion
results under one owner. The implementation may use a mutex plus a bounded owner
loop for publication/expiry; adapter callbacks and audio queue callbacks must run
outside its lock. Changed hints coalesce only notifications, never observations
or uncertain-write identities. No unbounded command/event history is needed.

## Audio authority

```go
type MeetingMicPreference struct {
    Known bool
    Muted bool
    Applied bool
    Revision uint64 // Existing audio worker revision for strict captured writes.
}
type MeetingMicWrite func(value bool, revision uint64, complete Completion) error
func (m *Meetings) AttachAudio(ctx context.Context, write MeetingMicWrite) (*MeetingAudio, error)
func (a *MeetingAudio) Report(preference MeetingMicPreference)
func (a *MeetingAudio) Close()
```

Audio's existing worker and persisted `Intent.MicMuted` are the sole durable
authority. The audio publisher reports after installing its snapshot and releasing
its mutex. The callback only queues `SetMicMuteWithResult`; it never writes the
state file or controls Voicemeeter from the coordinator. A changed external mute
or explicit request first passes that save-before-apply boundary. The coordinator
does not fan out an unsaved target. Save failure leaves the previous target and
reports failure; Unknown is not retried automatically. Audio's native mute state
does not override saved intent. Existing origin text is diagnostic, not an echo
token. Completion and preference reports may arrive in either order.

Without audio attached/known, first known app mute establishes a runtime target.
That target survives empty-call gaps. When known audio attaches later, its loaded
saved preference wins; no app baseline is written over it. Absence/failure of
audio must not prevent Meet camera or runtime mute operation.

Camera has no durable storage. First known camera observation when no camera
target exists establishes its target. New joined peers inherit the current target
while any eligible joined peer remains. When the last joined peer disappears,
discard camera target, cancel all old camera work and retire its revision. The
next known joined observation starts a new baseline; do not restore old Camera On.

## Integration ownership

- Coordinator owner: snoofer meeting source/tests, Services/host or app lifecycle,
  audio attachment/reporting/save callback. Existing audio dependency is not added
  to Meet. Discord's pre-existing Requires audio may stay for this checkpoint.
- Discord adapter: attachment, observation edges, explicit checked native writes,
  camera revision tracking, removal of its independent mute arbitration.
- Meet adapter: report all current joined calls, stable per-browser/tab IDs and
  exact lifetimes; shared actions cover every compatible call, not only selected.
- Parent: GUI/deck shared controls, documentation/acceptance and saved Plus XL
  Like/Dislike placements. Meet adds observation identity; transport and command
  formats stay unchanged.

The host creates `NewMeetings(controls *Controls, live bool) *Meetings` when no
instance was supplied through Services. Construction is inert; `Start(context.Context)
error` starts the owner before plugins. `Close(context.Context) error` cancels its
work and bounds shutdown, including when plugin cleanup fails.
