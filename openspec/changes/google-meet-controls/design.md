# Design

The user superseded the independent/selected-only mute and camera policy on
2026-10-15 with [universal meeting state](../universal-meeting-state/design.md).
The transport and DOM safety contract remains; that follow-up owns shared intent,
all-call synchronization and the requested personal placement.

## Status and scope

The transport/source checkpoint and approved disposable-call feasibility probe are
complete as of 2026-10-15. The first release is Meet mic mute, camera and call state
on Stream Deck and in Meetings. Production implementation is authorized within
this resolved scope. Implementation and automated verification are now recorded in validation.md.
Live extension/provider/deck acceptance remains a separate checkpoint.

[Sanitized observations](observations.md) establish the supported DOM candidate and
its limits. Browser/call control still requires a separate approved session under
AGENTS.md; source work proceeds without another implementation approval gate.

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

## Decision and availability limit

Extract the existing localhost listener into `internal/browserbridge`. Use one
concrete owner and two typed consumers, Media and Meet. Keep `/nowplaying`, wire
protocol 2, the current WebSocket library, admission policy and Media messages.
Do not add another listener, a service registry, a generic event bus, a dependency
or persistent pairing credentials.

Use bounded FIFO delivery, with no snapshot compaction and no extra continuity
epochs. A full consumer queue retires the browser connection whose next report
could not be delivered. This is an explicit connection failure, not permission to
drop a report while retaining its targets. The extension reconnects using its
existing backoff; all state and artwork are reported again on the new generation.

This preserves ordered Media transitions and one-shot artwork while connected,
rejects old actions on overload, and keeps socket reading independent of provider
work. It does **not** guarantee uninterrupted Meet availability during an arbitrarily
long Media stall: if Media's queue fills, Meet in that same browser disconnects too.
Other sockets are not closed by that overflow, though their own reports can also
overflow a persistently stalled consumer's queue. Normal consumer stop, disabled
Media and a busy Windows media call below the queue limit do not close Meet.

This release accepts that bounded isolation: continued delivery up to the queue
limit, then visible connection retirement and fresh-state recovery. An arbitrary
consumer stall does not have an uninterrupted same-browser availability guarantee.

## Owner and callers

- Add `Browser *browserbridge.Bridge` to `snoofer.Services`. The bridge package must
  not import `snoofer` or either provider, avoiding an import cycle.
- Construct the inert owner in `app.Run` after controls-only, plugin-command,
  `--check` and existing-instance exits, before `host.Start`. It owns no socket or
  goroutine until acquired. Do not start it for a core-only run without consumers.
- Pass the legacy Now Playing settings to inert
  `browserbridge.New(ctx, legacySettings json.RawMessage)`. That constructor reads
  only `port` using a small ordinary JSON field extraction, not the strict
  whole-plugin decoder or a Now Playing import.
  Missing/zero means 47815; explicit values must be integers in 1024–65535.
  Unrelated disabled-plugin fields do not matter. Invalid port is a bridge status
  error; unrelated providers continue. There is no new settings field or migration.
- The public surface is concrete: `AcquireMedia(ctx)`, `AcquireMeet(ctx)`,
  `Status()`, and `Close(ctx)`. Each acquisition returns its typed lease; a duplicate
  live acquisition of that type is an error. Leases provide `Updates()`, generation
  lookup, their typed send method and idempotent `Close(ctx)`.
- First acquisition starts the listener and five-second busy-port retry. Media
  still shows Windows sessions when listening fails. Last release cancels retry,
  closes the listener and all hijacked sockets, then joins owned goroutines.
  Cancellation must precede waiting; no delivery can keep shutdown blocked.
  Reject acquisition while final shutdown is still joining; never start a second
  listener to race unfinished cleanup. The owner can be acquired again once joined.
- Closing one lease first disables its sends and settles its pending requests,
  discards its unread queue, and updates negotiated feature enablement. The other
  lease and existing sockets remain. A partial provider start releases its lease.
- Preserve whole-host restart for plugin enablement/settings changes. Port changes
  take effect on that restart, not through a new runtime reconfiguration system.
- `app.Run` joins bridge cleanup with existing bounded host cleanup and includes
  errors before releasing the instance event or relaunching. An ignored deferred
  close is insufficient. The existing five-second cleanup deadline bounds both.
- Replace Now Playing's `listen`/retry/close ownership in `plugin.go` with its Media
  lease. Preserve its Windows thread, session merge/focus, ratings, settings,
  semantic controls and existing Submitted/observation behavior. `publish.go`
  reads actual bridge status. Move only transport DTOs and validation from
  `browser.go`/`bridge.go`; provider conversions remain in Now Playing.
- The new `googlemeet` provider has no dependency on Now Playing, Discord or
  audio. It uses `googlemeet.mute`, `.camera`, `.call` and `.target`. Its semantic
  actions check `Services.Live`; preview never writes browser mic/camera state.

## Admission, queues and recovery

Keep loopback-only binding, extension-origin checking, reject-second-live-browser
admission, hello timeout 5 s, idle timeout 60 s, extension ping every 20 s, maximum
WebSocket message 1 MiB and write timeout 2 s. Preserve monotonically increasing
connection generations across listener reacquisition during this owner lifetime.
No browser name, tab ID or call label substitutes for that generation.

Each active consumer has one FIFO with **16 state updates**. Each accepted update
is immutable and carries browser name, connection generation and owner receipt
time. Media keeps its current 64-session and 128 KiB-per-art-field bounds. Meet
state frames have an additional 64 KiB wire limit and at most 64 tab records.

Readers use a nonblocking enqueue. On full queue, atomically remove that exact
connection from the active map, mark its generation unavailable, cancel its
pending Meet requests and close that socket. Do not wait for a disconnect update
to fit the same queue. Old queued reports fail the generation check. Each provider
also reconciles its remembered browsers against current owner generations before
publishing/processing, so a missed disconnect notification cannot retain targets.
New connections get new generations; old writes never resolve a replacement.

Do not decode images or invoke native/plugin code on a socket reader. Do not
start one delivery goroutine per message. Result dispatch bypasses state FIFOs:
each of at most eight pending Meet requests has a one-item result slot. A reader
can complete that slot once without waiting for the provider.

Serialize writes with a context-aware one-slot gate, retaining the existing 2 s
socket deadline. The 2 s total write budget includes waiting for the gate. Recheck
lease activity and the captured connection after acquiring it. Distinguish a
known rejection before writing from an error after a write may have started.
Never hold the owner mutex during socket I/O or cleanup waits.

There is no owner artwork cache or history journal. While Media is disabled,
Media reports are ignored. On enabling Media for an already connected new
extension, the capability transition below clears `sentArt` and requests a current
full report, followed by fresh artwork. For an old extension that cannot honor
that transition, close that old connection once to get its existing full reconnect
report; such a connection cannot carry Meet. Acquiring Media must not disconnect
a compatible connection already serving Meet.

## Negotiation and exact wire envelopes

Extension hello adds one optional capability; all existing fields remain:

```json
{"type":"hello","protocol":2,"version":"<extension version>","browser":"Brave","meet":1}
```

After admission, a new host sends this snapshot, and resends it when leases change:

```json
{"type":"capabilities","meet":1,"mediaEnabled":true,"meetEnabled":true}
```

`mediaEnabled` means the Media lease is active. `meetEnabled` means both that the
Meet lease is active and that this particular hello advertised `meet: 1`.

An extension with no capability reply continues current Media behavior and never
sends or accepts Meet work. A new host with no `hello.meet == 1` keeps Media working
and reports Meet's update requirement. Unknown Meet versions remain Media-only;
they do not change the base protocol mismatch behavior. A new extension must also
require current `meetEnabled == true` before any Meet action. On socket replacement
or `meetEnabled: false`, the background immediately disables the adapter, cancels
unsent work and pending readback, and clears old result correlations. Require fresh
enablement and a fresh observation before accepting another action. Existing
per-port identity plus enabled state supplies this boundary; add no history layer.
Cancellation never restores mic/camera state.

For Media, the first true capability snapshot, or false-to-true transition, clears
`sentArt` and schedules the current full Media report. False suppresses Media
reports but retains page/runtime observation for later activation. Before the first
capability reply, Media operates exactly as it does with an old host. Duplicate true
snapshots do not repeatedly download artwork. Base `sessions`, `command`, `refused`
and `ping` envelopes, including YTM rating fields, are otherwise unchanged.

Meet state is a full list for this browser. The background, not the page, adds
`tab` and `instance` from admitted runtime ports:

```json
{"type":"meet-state","calls":[{"tab":7,"instance":"42","sample":"29","call":"3","state":"joined","mic":{"value":true,"revision":"8","available":true,"reason":""},"camera":{"value":false,"revision":"5","available":true,"reason":""}}]}
```

- `tab`: nonnegative safe integer. Duplicate tabs in one full list are invalid.
- `instance`: nonempty decimal string, at most 20 characters, allocated from a
  background-lifetime monotonic counter on admission of each Meet content port.
  It represents the verified document/port lifetime; `sender.documentId` is also
  checked when available but need not be copied onto the wire. A background restart
  replaces the socket generation, so restarted counters cannot revive old targets.
- `state`: exactly `joined`, `prejoin` or `unknown`. `call` is a nonzero decimal
  counter string of at most 20 characters while joined, and the empty string
  otherwise. Leave/rejoin or loss/reacquisition of joined identity advances it.
- `sample`: nonzero uint64 decimal observation counter, at most 20 characters, advanced
  by the adapter on each fresh read and scoped to instance. See freshness below.
- `value`: required boolean or JSON null. For mic, true means muted; for camera,
  true means on. Missing is invalid, not false. `available` is a required boolean;
  true requires a joined call, known value and supported actionable control.
- `revision`: nonzero uint64 decimal counter string, at most 20 characters. The isolated
  adapter advances each control's revision on every observed value/availability
  transition, including changes back to a prior value. Heartbeats do not advance it.
- `reason`: exactly empty, `unknown`, `unavailable`, `not-in-call` or
  `reload-required`. Empty requires availability; unavailable requires a reason.
  No unproven host/permission diagnosis and no arbitrary page text in this field.
- No meeting URL/code, participant data, page title or media is transported. The
  provider identifies choices as browser plus stable tab number and current local
  states. More descriptive naming is a separate presentation decision, not an
  excuse to copy meeting content.

Commands use explicit state, never an unspecified toggle:

```json
{"type":"meet-command","id":"106","tab":7,"instance":"42","call":"3","control":"mic","revision":"8","value":false}
```

`id` is a nonzero decimal owner-lifetime counter string (at most 20 characters).
`control` is only `mic` or `camera`; `value` must be a boolean. The rest exactly
matches the captured state. Connection generation and provider selection revision
stay in the host's pending entry and are checked before writing. They are not
trusted page-supplied routing fields.

The matching result repeats the captured target and intent:

```json
{"type":"meet-result","id":"106","tab":7,"instance":"42","call":"3","control":"mic","revision":"8","value":false,"status":"observed","reason":""}
```

`status` is exactly `observed`, `rejected` or `unknown`. `reason` is empty for
observed; rejected uses `stale`, `unavailable`, `busy` or `disabled`; unknown uses
`not-confirmed`, `disconnected` or `stopped`. Result envelopes are capped at 2 KiB.
Every echoed field must match the pending entry. Reject malformed types before
looking up side effects; ignore unknown, duplicate, expired or obsolete results.
Results do not overwrite current state snapshots.

## Runtime path, freshness and action completion

Use a separate isolated-world Meet content script and runtime port named `meet`,
matched only to `https://meet.google.com/*`, top frame. Validate the actual sender's
URL origin, frame 0, tab and current port/document at admission and on messages.
Do not route through the generic Media main-world custom-event relay. Existing
permissions cover this; do not broaden them. On duplicate injection, retain one
adapter/port owner and never duplicate clicks or adopt an obsolete adapter result.
The isolated adapter reconnects after runtime-port loss with bounded backoff even
though observations are requested by the background. Port loss cancels pending
work/readback immediately; reconnection establishes a fresh admitted instance and
only observes. It waits for fresh enablement before publishing actionable state or
accepting commands, and never resumes a canceled click or pending activation.

The background requests a fresh adapter observation every 1 s while Meet is
enabled, and forwards event-driven changes promptly. Each adapter reply is bounded
to 8 KiB. It rebuilds state from the live document; cached reports do not count as
fresh observations. The background can bundle current per-tab observations for up
to 200 ms, preserving each adapter's call/revision counters. No reply for 3 s
removes that tab from the reported list and invalidates its selected lifetime.
Each host tab record expires after 3 s without a newly sampled adapter report.

To prevent other tabs' heartbeats from refreshing stale cached state, the required
`sample` advances on every fresh adapter read. The host updates that
tab's receipt time only when its sample advances; lower samples are invalid and
equal samples may be repeated without refreshing freshness. Samples are scoped to
instance. Do not use page wall-clock timestamps as freshness proof.

These timing constants are product limits. The approved probe observed one hidden-tab
mic transition and a slower camera startup within the page budget; it did not test
implemented heartbeats, freshness expiry or every browser scheduling condition.
A throttled/suspended page becomes unavailable; neither side guesses from its last
known state. Implementation and live acceptance must verify those paths separately.

The provider permits one pending operation per exact call/control and at most
eight total. Mic and camera can proceed independently. Capture desired state and
all identities before enqueueing; reserve before sending. Busy rejects without
writing. The total completion budget is 5 s from acceptance, including queue/gate
wait and write. The adapter's post-click observation budget is 3 s. All waits are
cancelable. Freshness expiration makes controls unavailable immediately.

At each routing boundary recheck current lease, socket, port/document, call and
control revision. Immediately before acting the adapter reads live state again.
A changed revision rejects even if the value returned to its former value. An
unchanged valid target already at the explicit desired value needs zero clicks.
Otherwise click at most once, observe the same live target, send fresh state, then
return `observed` only after that read shows the desired value. DOM dispatch alone
is not confirmation. Use only the observed DOM contract below and revalidate it on every read.

The background captures the socket reference and runtime port for each command.
It must never forward an old port's result onto a new socket or reconnect request.
Port loss before routing is rejected; after possible dispatch it is unknown.
Do not retain unsent commands for reconnect. Duplicate request IDs while pending are ignored without a second click or result;
the original request retains its bounded completion. A different ID on a reserved
control is rejected as busy. No retry path resends completed commands.

Map `observed` to `Processed` ("Meet state observed"), `rejected` to `Rejected`,
and `unknown` to `Unknown`. A successful socket write leaves the completion pending;
it does not itself finish a Meet action. Write failure after possible submission,
timeout or cancellation settles Unknown exactly once. Results from removed targets
cannot complete newer actions. A current valid observation may update displayed
state independently, but no late result changes a settled action or sends a command.

Initial sole-call selection, explicit choice when ambiguous, pinning when a second
call appears, no fallback after target loss, and prejoin exclusion remain as in the
feature spec. Reconnection observes only. It never restores unmute/camera-on or
links Meet to Discord/global Mic Mute.

## Trust boundary

Retain the existing same-user local-software trust boundary for this first release.
Origin and duplicate-browser checks do not authenticate the first extension or
authenticate Snoofer to an extension. A different local server that wins the port
can issue commands; this design does not claim otherwise. Do not add implicit
pairing/authentication scope. Runtime sender and lifetime checks still prevent
page-provided IDs or obsolete ports from impersonating another Meet document.

## Required regression checkpoints

1. Inert owner; first/final acquisition; Media-only, Meet-only and both orders;
   duplicate acquisition; partial-start release; stop-one isolation; busy-port retry;
   cancellation/cleanup deadline and port release; legacy port absent/disabled/
   invalid; optional builds without a Now Playing import.
2. Existing Media origin/protocol/duplicate-browser/generation tests, Windows merge,
   ratings and native-thread behavior. Preserve all new YTM tests during extraction.
3. Fill each 16-entry FIFO, then overflow: the offending connection is retired
   before stale input can write; no reader hangs; other sockets remain admitted;
   recovery uses a new generation. Verify the documented same-browser outage,
   discarded old reports, no replay and artwork restoration on reconnect.
4. Block Media below the bound and verify Meet state/results continue. Repeat with
   blocked Meet. Result completion never waits behind either state queue.
5. Old/new host/extension combinations; duplicate capabilities; disabled/re-enabled
   Media artwork refresh; late Media acquisition while Meet stays connected;
   old-extension-only refresh; unsupported Meet capability.
6. Exact result correlation and one completion; wrong/late/duplicate IDs and target
   fields; old socket/port/call; counter limits; malformed/oversized frames; eight
   pending limit; per-control busy; write-gate deadline; timeout/disconnect no replay.
7. Per-tab sample freshness, repeated cached sample, out-of-order sample, suspended
   adapter, background observation and same-document call replacement. No stale
   state can remain available because another tab is reporting.
8. Registry/UI/deck tests retain separate Meet/Discord gates and layouts, truthful
   unknown/pending states, explicit selection and global mute behavior. Live adapter
   acceptance remains separate from transport tests.

## Observed DOM contract and adapter boundary

The approved probe used Brave with a Chrome 155 user agent and English Meet UI,
with only the tester in a disposable call. It established one supported layout
candidate, not all Meet variants or production integration acceptance. Refer to
[observations.md](observations.md) for exact transitions and cleanup evidence.

Use a dedicated isolated-world content script. State comes directly from DOM
observations; no MAIN-world script, public custom-event relay, page-provided target
identity, Meet private API or global keyboard shortcut is required. The background
validates the actual top-frame `https://meet.google.com` runtime sender and owns
its admitted document/port instance. DOM attributes remain website-controlled
observations, not proof of remote audio/video delivery.

A joined call requires exactly one connected root matching
`[role="region"][jsname="vNB5le"]` and exactly one native Leave button
`button[jsname="CQylAd"]` inside it. The Leave button is an identity signal only;
this feature never clicks it. Mic and camera are independently identified within
that root:

| Control | Native button | Known state |
| --- | --- | --- |
| Mic mute | `button[jsname="hw0c9"]` | `data-is-muted="true"` means muted; `"false"` means unmuted. |
| Camera | `button[jsname="psRWwc"]` | `data-is-muted="false"` means on; `"true"` means off. |

Each actionable control must be unique, connected, enabled (`disabled === false`)
and visibly laid out with a nonempty rectangle. Accept only literal `true` or
`false` for the attribute. Missing, malformed, conflicting, disabled or duplicate
controls are unknown/unavailable; do not infer Off, trust a translated label or
click a guessed replacement. If joined-root/Leave identity is ambiguous, no
control is actionable. Zero joined roots/Leave buttons means not joined, even when
prejoin reuses the same mic/camera button names. Call state can report prejoin only
when its corresponding live prejoin controls are unambiguous; otherwise use unknown.

A MutationObserver watches relevant control attributes and joined-root/control
membership, publishing changes promptly. Background-requested heartbeat reads
rebuild the same bounded observation from the live DOM; an isolated-world timer or
cached last report is not freshness proof. Root loss or replacement ends the joined
call epoch; reacquisition starts a fresh epoch even within the same document.
Control node replacement/loss advances that control's revision even if its next
value is identical. Every observed value/availability transition also advances the
revision, including change away and back. Before acting, synchronously reread the
same nodes, epoch, revision, availability and explicit desired value. Invalidate
old work rather than adopting replacement nodes. Recheck after a click while
waiting for observed state on the same call/control lifetime.

The probe observed mic mute after 504 ms, hidden-tab unmute after 1004 ms, and camera
startup via an attribute observer after 1574 ms. Camera Off was then requested
immediately and observed at 1608 ms total from the initial On request. The planned
3 s page observation budget and 5 s total command deadline accommodate these
samples; they are limits to verify in implementation, not guaranteed site latency.
No observation confirms whether audio/video reached any remote participant.

Leaving detached the old joined root. Rejoining opened a fresh document, with the
probe marker absent. That supports document/call identity design; it does not test
implemented rejection of an old command, same-document rejoin, multiple calls,
extension outcomes or physical Deck input. Those remain tasks 4, 6 and 7.

## Provider and controls

Use a narrow optional Meet provider with no Discord or audio dependency. Stable
IDs are `googlemeet.mute`, `googlemeet.camera`, `googlemeet.call` and, where
selection is needed, `googlemeet.target`. Mute On means Meet's mic is muted; Camera On
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

Selection policy: on initial discovery in a provider run, a sole eligible
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
not prove mute/camera changed. Use the freshness and completion deadlines defined
above. Timeout/disconnection after possible submission is
uncertain, never proof of failure or success; do not retry, replay or resend on
reconnect. Late results never alter settled outcomes; valid fresh observations
can still update
current state. Stopping the provider settles outstanding claims exactly once.

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

## Meet-origin exclusion from generic browser Media
The existing all-sites Media adapter could discover HTML media inside a Meet call
and use a tab-title fallback. The new companion excludes exact top-frame/frame
origin https://meet.google.com from generic Media admission and publication, and
does not inject its generic Media scripts on that origin. The isolated Meet
adapter remains the only companion report source for these documents. This avoids
transporting meeting titles/codes through an unrelated envelope or letting call
media compete with music-session focus. Derive the exclusion from the actual
runtime sender origin; do not trust a page-supplied site label. Regression checks
cover generic Media ports from Meet and continued Media operation on other sites.
This intentionally narrows the new companion's browser Media discovery on Meet.
Windows media-session reporting and older companions are unchanged; the feature
does not claim to redact metadata independently published by those existing paths.
