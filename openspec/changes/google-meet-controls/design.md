# Design

## Status and scope
Proposed first release: Meet mic mute, camera and call state on Stream Deck and in
Meetings. Production implementation is authorized within the resolved scope, with
no new implementation approval gate. Resolve the transport and live feasibility
checkpoints below in this design before dependent implementation, following the
repository's OpenSpec workflow. Live browser/call control still requires an
approved session under AGENTS.md.

## Existing behavior to preserve
`app/web/src/screens/meetings.ts` currently renders Discord; `plugin-lifecycle.ts`
gates Meetings on Discord. `plugins/discord/controls.go` captures connection and
channel lifetimes and uses semantic controls and completion callbacks. Reuse those
patterns. The old Insta360 camera card was removed; it is not part of this feature.

`plugins/discord/link.go` deliberately reconciles Discord mute with audio Mic Mute
in both directions. Meet must not join that reconciliation. A host-imposed Meet
mute changes only Meet's reported state. Turning Meet's mic on does not unmute an
upstream Snoofer source. Where audio state is available, show a separate concise
source-muted indication; never present Meet's local unmuted state as proof that
sound can reach the call. Audio being disabled does not disable the Meet provider.

The current extension sends media reports over one local WebSocket owned by
`plugins/nowplaying/bridge.go` and its plugin worker. Its background already tracks
tab/frame ports and socket generations; media routing rejects obsolete instances.
Its tab-mute operation controls browser output, not the microphone. The main-world
media report relay is not a suitable command/state authority for Meet controls.

## Prerequisite: narrow transport ownership decision
Trace extension startup, browser admission, plugin registration, shutdown and the
current command/result flow before choosing the smallest shared boundary. Record
the concrete owner, package placement, registrations, cancellation, bounded queues,
message/result schema and update compatibility here before changing production code.
The source-backed candidate is one socket owner in `internal/browserbridge`, with
one concrete `Services.Browser` reference created per `app.Run`. Typed Media/Meet
leases would start the listener on first acquisition, isolate each consumer's stop,
and close it on final release. Retain `nowplaying.settings.port` as the legacy port
source even when Media is disabled; document port reload behavior before coding.
Finalize this candidate at the checkpoint, not as a generic shared-services framework.
A separate Meet listener would be a smaller local patch but would duplicate the
extension connection, status and port path.

A compatibility candidate preserves `/nowplaying` and protocol-2 Media, adding a
`meet: 1` hello capability and a host capability/enabled reply. Missing Meet support
would leave old/new sides Media-only without a coordinated update. Confirm those
handshake cases before finalizing the schema. Meet gets its own correlated result
envelope; do not rewrite the existing Media outcome contract.

Today `bridge.deliver` can block on Media's 16-item inbox. The shared reader must
isolate Meet delivery from Media snapshot, artwork and native-command stalls using
bounded per-feature delivery. Record queue/coalescing rules that preserve Media's
one-shot artwork semantics; do not gain isolation by silently dropping artwork or
losing command results. These are prerequisites for the chosen ownership design.

Required behavior: either provider can run alone; stopping one leaves the other's
connection usable; the final owner releases resources. Preserve existing media
behavior, reject-second-live-browser admission and generation checks. Do not create
competing listeners or require the Media plugin merely to enable Meet. Version or
negotiate the new capability explicitly; an old extension makes Meet unavailable
with an update reason rather than falling back to unsafe targeting. Record whether
the chosen compatibility strategy requires a coordinated extension/host update.

The current bridge accepts extension origins without authenticating the first
claimant. In the reverse direction, the extension trusts whichever local server wins
its loopback port: Origin and duplicate-client checks do not authenticate Snoofer
to the extension. Such a server could therefore request mic/camera-on actions.
At this checkpoint, explicitly document whether to retain the existing same-user
local-software trust boundary or amend the design with authentication. This is a
design decision within the authorized work, not an invented user-approval gate or
automatic authorization to implement pairing. Do not claim peer authentication from
the current origin/admission checks.
The extension adapter accepts command messages only through its current extension
runtime connection, not public page custom events. Validate the actual sender's
Meet origin, top frame, document and port rather than page-provided target IDs.

## Prerequisite: disposable-call feasibility
Ask before each session of desktop input or visible launches, following AGENTS.md.
Use an approved disposable call whose session scope explicitly includes any
camera/mic-on tests. That consent covers those steps while the approval remains
active; do not ask again per step. No production meeting or unrelated participant
is required. Record the approved browser, Meet layout/language, controls observed,
selectors, call-lifetime signals, action/readback evidence, timing bounds and cleanup.
Do not store media, participant lists or meeting content; redact call identifiers
from shared evidence.

Prove top-frame observation, manual changes, one requested change while the tab is
in the background, prejoin versus joined state, leave/rejoin and stale-document
rejection. Selectors and lifecycle signals are unproven until that probe. If they
cannot distinguish a supported call/control reliably, report the limitation and
revise the design rather than guessing from button text or sending global shortcuts.
Declare browser/layout coverage from evidence; do not advertise untested variants.

## Provider and controls
Use a narrow optional Meet provider with no Discord or audio dependency. Proposed
stable IDs are `googlemeet.mute`, `googlemeet.camera`, `googlemeet.call` and, where
selection is needed, `googlemeet.target`. Confirm naming in the source-design
checkpoint before implementation. Mute On means Meet's mic is muted; Camera On
means Meet reports its camera on. Unknown is distinct from Off. Publish call state,
control availability and concise reasons for not-in-call, selection needed,
unsupported/unreadable controls, permission/host restrictions and disconnection.
Do not promise detection of a specific restriction the adapter cannot observe.

Reuse the semantic registry, revisions, completion callbacks and existing icon/state
rendering. Meetings renders independent Discord and Meet cards/gates; failure or
disablement of one does not hide the other. Plugins remains lifecycle-only and
Third-Party Apps retains connection diagnostics. Meet mic/camera controls and call
status are bindable using existing key/status placement rules. Keep all saved and
default layout positions unchanged in this change. A separate placement request
can assign the new controls after they exist.

## Call selection and command identity
Proposed selection policy: on initial discovery in a provider run, a sole eligible
joined Meet call may be selected for display. When no valid target is selected,
multiple eligible calls require explicit selection with a distinguishable
call/browser label. Once selected, keep that exact lifetime even if another call
appears, browser focus changes or media starts playing. When it disappears, clear
the selection and require a fresh explicit selection, even if only one call remains
or the same URL rejoins. That loss must not reset the initial auto-selection rule
within the provider run. Selecting a call itself never changes mic/camera state.
Prejoin/lobby is not an eligible call: status can report Not in a call, but mic/camera
actions stay unavailable even when Meet displays its prejoin buttons. Returning to
prejoin ends any formerly selected joined-call lifetime.

Every action captures the registry/selection revision, browser connection
generation, tab, document or equivalent content-port lifetime, joined-call epoch,
control state revision and explicit desired boolean. Tab ID, meeting URL, label
and browser name are not lifetime identities. The background derives document/port
identity from its runtime sender; [Chrome's sender documentation](https://developer.chrome.com/docs/extensions/reference/api/runtime/)
provides documentId/frameId where supported. Leave/rejoin in the same document must
also create a fresh call epoch. Recheck captured identity at each routing boundary
and immediately before any DOM action; late reports/results cannot revive an old
target. Browser/profile support remains the existing bridge admission scope until
separately designed; do not silently merge multiple profiles claiming one browser.

Convert a press into a desired state using known captured state. Reject stale or
unknown state before a side effect. If a validated target is already at an explicit
requested state, it needs no click. Otherwise perform at most one action after an
immediate state check; do not blindly invert whichever state exists at execution.
An intervening manual/host change invalidates the old control revision. Serialize
overlapping requests to the same control with explicit busy feedback, while allowing
unrelated controls. This guards known races; it does not make remote Meet state
and a DOM click atomic. Only subsequent observation establishes the visible state.

## Outcomes and privacy
Correlate each command and result through extension, provider and GUI completion.
Separate sent/pending from observed state. A socket write or DOM click alone does
not prove mute/camera changed. Declare bounded freshness and completion deadlines
at the design checkpoint. Timeout/disconnection after possible submission is
uncertain, never proof of failure or success; do not retry, replay or resend on
reconnect. Late results update only their original still-valid action, and stopping
the provider settles outstanding claims according to existing outcome semantics.

On launch, reconnect, target selection, call replacement and settings/layout reload,
publish observations only. Do not persist a desired mic/camera-on preference or
restore a prior call's state. No operation captures audio/video or fetches meeting
content. Retain only the metadata needed to distinguish calls; diagnostics should
omit private meeting URLs/codes and unnecessary page titles.

## Follow-up ideas, outside implementation scope
- Show Call: reveal the selected exact tab without changing call state.
- Raise/Lower Hand: command plus observed state, after the core adapter proves
  reliable and its capability/permission behavior is separately specified.
- Step Away: an explicit action to mute and turn the camera off. A later return
  action must be explicit, belong to the same call lifetime and preserve manual
  changes; no automatic unmute/camera-on or restoration into a replacement call.

Screen sharing, Leave, push-to-talk, global linked mute, calendar triggers and
automatic meeting/audio modes require separate proposals. They are not hidden
requirements or reasons to expand this first release.
