# Design

## Shared owner and ownership boundaries
One concrete coordinator owns mute/camera intent and reconciliation decisions.
The host supplies it through Services. Existing provider workers retain all RPC,
browser and audio I/O, exact call identities, observation revisions and bounded
request queues. Providers publish every eligible joined call, not only the selected
Meet tab. The coordinator must not dispatch by repeatedly invoking UI toggles.
No service registry, extensible rules engine or second browser listener is needed.

The audio adapter registers its existing MicMute snapshot and
SetMicMuteWithResult operation without introducing a hard Meet/audio dependency.
Audio's loaded, persisted preference is the sole durable mute authority. External
mute intent goes through its existing revision-checked save-before-apply path;
only a successful saved/observed preference is propagated. Failure stays visible
and does not manufacture a saved target. When audio is absent, a runtime-only mute
target seeds from the first known eligible app value and survives an empty-call
gap within that host run. If audio becomes available, its saved preference wins.
Do not create a second writer to audio configuration or reset routing/gains.

Discord's independent reconcileMute arbitration is replaced by this owner. Keep
the existing Discord/audio dependency and protocol code unless a concrete adapter
change needs otherwise. Discord deafening produces a synthetic mute; exclude that
mute from adoption/writes while deafened, then baseline and apply the current target
after undeafening. Do not automatically change Deafen, Screen Share or Leave.

## Camera lifetime and initial state
Camera has no persisted target. After no eligible joined calls remain, clear its
target and invalidate old pending work. The first fresh known writable camera
observation in a new call period seeds the target. Unknown observations cannot
seed it. If multiple initial observations conflict, process them in deterministic
provider/peer order; the first known observation seeds, later initial observations
inherit. New eligible calls while another call remains inherit the shared target,
including camera On or mute Off, as explicitly required by universal target state.
Leaving/prejoin, provider stop, stale/unavailable call identity and disconnected
transport retire the affected participation. Merely losing one camera's writable
capability does not erase the target while another compatible call remains.

## Observation, arbitration and commands
Identify each peer by a stable local source slot plus exact call lifetime. Provider
generation/document/call tokens remain captured by the provider, never inferred
from labels or URLs. First observations and control replacement establish a
baseline; only later known, writable value transitions are external candidates.
Availability/revision changes alone never change intent. The latest eligible
external value change received by Snoofer wins; no cross-app wall-clock comparison
or claim to know the last physical click. Meet reports cannot distinguish all
manual, host-imposed and delayed effects; document that observation limit.

Every outgoing operation captures the shared intent revision and exact peer/control
revision with an explicit desired value. Recheck before transport I/O; perform zero
actions when already equal, at most one otherwise. Cancel obsolete unsent work.
Allow mic and camera independently, but only one operation per peer/control.
Use the existing bounded provider completion/readback deadlines.

Track sent/uncertain expected values separately from external observations. Matching
own readback, including a delayed result after newer intent, acknowledges that work
and must not become a new intent. Retain an uncertainty fence for the source slot
across connection churn; heartbeat, changed availability/revision or reconnect
alone cannot repeat an uncertain attempt. A fresh explicit/external intent can
supersede it, or observation can resolve it. A definitely ended call followed by a
new call establishes a new baseline. No timer retries of failed target application.
Known-unsent stale/busy rejection and possibly-submitted Unknown remain distinct.

Discord camera uses the existing toggle-only RPC: require a known current value
unequal to desired, verify connection/channel lifetime and control revision, then
toggle once. An RPC reply alone is not observed success; await the camera event.
Discord initially has no proven camera query, and its camera event lacks a channel
token. Unknown camera remains unavailable. Preserve exact captured lifetime and
record actual cross-channel attribution/live support as an acceptance limitation;
do not invent a trustworthy private query or treat unknown as Off.

Meet fanout reuses worker.calls, freshness, bridge.Send and exact correlated results.
Selected-tab state may remain for per-app diagnostics, but shared intent and other
calls' work do not depend on it and selection changes cannot retarget universal work.
Preview publishes state but never saves intent or changes app/audio state.

## Controls, UI and personal placement
Publish meetings.mute and meetings.camera as existing-style semantic toggles, with
short labels Mute and Camera and the existing mic/camera icons. Values describe the
shared target; Pending, Not confirmed and N/A distinguish incomplete/unknown state.
Target acceptance/saving is distinct from every app's observed convergence. Keep
per-app actual mute/camera state visible without a duplicate set of competing
toggle actions. Existing provider gates, connection diagnostics and Discord-only
actions remain available. Audio Mic Mute stays the same saved preference.

Replace only the explicitly requested personal Meetings mute/camera bindings during
the next approved live session. Preserve compatibility for existing saved
discord.mute/discord.video and googlemeet.mute/googlemeet.camera bindings by routing
their requests to shared intent; do not silently retain local competing policies.
Do not change shipped defaults or other users' saved assignments.
On the user's Plus XL only, bind Like at Home index 8 and Dislike at index 17;
those slots were verified empty. Smaller decks remain unchanged for later redesign.

## Validation and rollout
Use focused reducer/adapter tests before full checks. Cover source save failure,
audio absence, latest-change ordering, own delayed echo, uncertainty/reconnect,
stale and ABA captures, first/new/last call, deafen, unknown camera, independent
pending operations and multiple Meet tabs. Test GUI shared-target/actual-state
presentation and native key sizes. Run full Go/vet/race, frontend/GUI checks,
strict OpenSpec validation and canonical build; preserve launch configuration.
No desktop control or live account/layout changes until the user approves a new
session. Real-site, both-app and physical-deck acceptance remain unchecked until run.
`nCompatibility bindings display the shared target they toggle. SurfaceOnly status controls`n`discord.call-mute`, `discord.call-camera`, `googlemeet.call-mute` and`n`googlemeet.call-camera` expose actual app state without competing actions. The`nGUI uses those readouts; aliases may mirror the corresponding shared controls.
