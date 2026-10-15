## ADDED Requirements

### Requirement: Meet controls publish local observed state

The Meet provider SHALL publish separate mic mute, camera and call status controls
through the existing semantic registry. Known On/Off, unknown, pending and
unavailable states SHALL remain distinct. Mic/camera action availability SHALL
require an eligible joined call and a supported, observed control; call status
SHALL remain readable without one. The first release SHALL exclude prejoin actions
and audio/video capture.

#### Scenario: Meet state changes outside Snoofer

- **WHEN** the user or meeting host changes the selected call's mic/camera state
- **THEN** Snoofer and bound deck controls reflect the newly observed local state
- **AND** the provider does not impose a saved desired state or change global audio

#### Scenario: A control cannot be observed or changed

- **WHEN** the call is absent, the adapter is unsupported, a control is unreadable or a restriction is observed
- **THEN** the affected control is unavailable with a truthful concise reason
- **AND** unknown state is never displayed as Off and no action guesses its state

#### Scenario: Meet displays its prejoin mic and camera buttons

- **WHEN** a tab is in prejoin or a lobby rather than an eligible joined call
- **THEN** Snoofer may report that call status but its mic/camera actions stay unavailable
- **AND** a transition from a selected joined call to prejoin ends that call lifetime

### Requirement: Meet has an independent provider lifecycle

Meet SHALL operate when Discord, audio and Now Playing are disabled. Meetings SHALL
gate each provider independently. Shared extension transport SHALL preserve existing
Media behavior, version/admission checks, bounded lifecycle and connection identity;
its concrete owner SHALL use typed consumer leases and the bounded delivery
contract in design.md. A full state FIFO SHALL retire the exact offending
connection and its generation instead of silently dropping a report.

#### Scenario: Only Meet is enabled

- **WHEN** Meet is enabled with a compatible extension and the other providers disabled
- **THEN** its connected call controls are available without enabling another provider
- **AND** Discord's disabled state does not replace or hide the Meet card

#### Scenario: One transport consumer stops

- **WHEN** Media and Meet are running and either provider releases its lease during normal stop or partial-start failure
- **THEN** the other provider retains its valid transport and controls
- **AND** final shutdown releases shared resources without accepting obsolete messages

#### Scenario: A consumer stalls within the queue bound

- **WHEN** one consumer pauses with fewer than 16 unread state updates
- **THEN** the socket reader continues delivering the other consumer's state and correlated results
- **AND** result completion does not wait behind either state FIFO

#### Scenario: A consumer state queue overflows

- **WHEN** an update cannot enter a consumer's 16-entry FIFO
- **THEN** the owner retires that exact socket and generation without waiting for a disconnect notification to enter the queue
- **AND** both Media and Meet in that browser may be temporarily unavailable until fresh reconnection
- **AND** other sockets remain admitted, stale queued reports cannot restore the retired target, and no command is replayed

#### Scenario: The extension lacks the Meet capability

- **WHEN** the host connects to an incompatible extension version or missing capability
- **THEN** Meet reports the required update or incompatibility
- **AND** no fallback sends commands using mutable tab labels or output-mute APIs

### Requirement: Meet mute remains separate from global and Discord mute

Meet mic mute SHALL affect only the selected Meet call. Existing Discord/global
mute linking SHALL remain unchanged. Where Snoofer audio state is available, its
upstream inhibition SHALL be shown separately from Meet's local mic state.

#### Scenario: Discord and Meet are both active

- **WHEN** a Meet mute request is processed or Meet reports a host-imposed mute
- **THEN** neither Discord mute nor Snoofer's global Mic Mute preference changes
- **AND** a camera request affects only the selected Meet call

#### Scenario: The upstream microphone remains muted

- **WHEN** Meet reports its mic unmuted while Snoofer's source is muted
- **THEN** Meet's local state remains truthful with a separate source-muted indication
- **AND** the request does not unmute Snoofer or claim audio reaches the call

### Requirement: Call selection is explicit when ambiguous and remains stable

On initial discovery in a provider run, a sole eligible joined call MAY be selected
for display. Multiple calls without a valid selection SHALL require explicit
selection with a distinguishable call/browser label. A selected lifetime SHALL
remain pinned when another call appears, until removed or explicitly changed.
Removal SHALL clear selection and require a fresh explicit selection, even if only
one call remains or the same URL rejoins; it SHALL NOT reactivate initial automatic
selection within that provider run. Selection SHALL NOT itself change mic/camera
state.

#### Scenario: Two Meet calls coexist

- **WHEN** two eligible calls exist without a chosen target
- **THEN** mic/camera controls require target selection
- **AND** browser focus, title order and media playback do not choose a target

#### Scenario: A second call appears after initial sole-call selection

- **WHEN** the initially selected call is still valid and another eligible call appears
- **THEN** the existing selection and its mic/camera controls stay bound to the first lifetime
- **AND** no additional selection or automatic switch occurs merely because there are now two calls

#### Scenario: The selected call ends

- **WHEN** the chosen call ends while another eligible call remains
- **THEN** the former target becomes unavailable and the selection is cleared
- **AND** neither queued input nor later presses silently control the remaining call

#### Scenario: The same meeting rejoins after selection was lost

- **WHEN** the former target returns as the only eligible call in the same provider run
- **THEN** it has a new call lifetime and needs a fresh explicit selection
- **AND** initial sole-call selection does not reactivate or restore its former commands

### Requirement: Every action retains its captured call and control identity

Actions SHALL carry the captured selection/revision, connection generation, tab,
document or content-port lifetime, joined-call epoch and control-state revision.
Routing SHALL validate those identities through dispatch and immediately before a
side effect. The extension SHALL validate the actual Meet top-frame runtime sender
and SHALL NOT accept Meet commands through public page events. Reused IDs, URLs,
labels and browser names SHALL NOT substitute for lifetime identity.

#### Scenario: A tab reloads or leaves and rejoins the same meeting

- **WHEN** a captured request reaches a new document or joined-call lifetime with the same tab or URL
- **THEN** it is rejected before a known side effect and does not target the replacement
- **AND** late reports/results from the former lifetime cannot recreate its selection

#### Scenario: Selection or connection changes while work is queued

- **WHEN** a newer selection or extension connection replaces the captured one
- **THEN** the old command is rejected without resolving a current replacement target
- **AND** recovery does not replay the old command

#### Scenario: An unrelated frame or page attempts to publish Meet control messages

- **WHEN** the sender is not the admitted current Meet top-frame runtime connection
- **THEN** its state and command messages are rejected
- **AND** page-supplied IDs cannot impersonate the admitted document or call

### Requirement: Desired-state commands have bounded correlated outcomes

A press SHALL derive an explicit desired boolean from known captured state. The
adapter SHALL recheck identity and state before at most one action, rejecting stale
state rather than toggling a newer value. A validated already-satisfied explicit
request SHALL need no click. Per-control busy handling SHALL prevent overlapping
requests. Correlated outcomes SHALL distinguish submission from observation and
settle under bounded deadlines/cancellation. The total command budget SHALL be
5 seconds from acceptance and the page observation budget SHALL be 3 seconds. A write or click alone SHALL NOT be
presented as observed success. Uncertainty SHALL NOT cause replay or retry.

#### Scenario: A manual change precedes dispatch

- **WHEN** the selected control's revision changes after a press is captured
- **THEN** the old request is rejected before clicking even if its value later returns
- **AND** no compensating toggle is sent

#### Scenario: A click is submitted but readback does not establish its effect

- **WHEN** observation or connection fails after an action may have reached Meet
- **THEN** its result remains uncertain rather than claiming success or no effect
- **AND** timeout, reconnect and later unrelated state never resend the action

#### Scenario: A matching result arrives late

- **WHEN** a bounded action has settled or its call lifetime ended before a reply arrives
- **THEN** that reply cannot settle a different action or operate another call
- **AND** the current control state comes only from valid current observations

### Requirement: Recovery never restores microphone or camera activation

Launch, reconnect, call replacement, selection and settings/layout reload SHALL
observe state only. Meet SHALL NOT persist or automatically restore a mic-unmuted
or camera-on intent from an earlier call or connection.

#### Scenario: Snoofer or the extension reconnects

- **WHEN** the last observed call had mic or camera on before disconnection
- **THEN** fresh state is read without any automatic mic/camera command
- **AND** a new call cannot inherit an old activation or pending request

#### Scenario: The socket is replaced or Meet is disabled during pending work

- **WHEN** the extension replaces its socket or receives `meetEnabled: false`
- **THEN** it disables the adapter, cancels unsent work and pending readback, and removes old result correlations
- **AND** new actions require fresh enablement and a fresh observation on the current admitted port
- **AND** reconnect never resumes a canceled command or restores mic/camera state

### Requirement: New controls preserve user layouts and existing presentation contracts

Meet mic/camera and call status SHALL be bindable using the existing deck editor
and semantic action/status rules, with truthful labels, icons and accessible GUI
state. This change SHALL NOT relocate default controls or overwrite personal deck
layouts. Broader meeting actions SHALL require a separate scoped design.

#### Scenario: A user enables Meet with a saved layout

- **WHEN** the provider publishes its controls
- **THEN** they become available for binding without changing saved positions
- **AND** existing Discord/audio controls retain their meanings and rendering

### Requirement: Live adapter claims require approved evidence

Actual selectors, joined-call detection and supported browser/layout coverage SHALL
be established in a separately approved disposable probe before dependent
implementation. Probe and acceptance SHALL obey desktop-control permission and
cleanup rules and SHALL avoid collecting meeting content or media.

#### Scenario: A proposed adapter cannot reliably distinguish call state

- **WHEN** the approved probe finds ambiguous controls or lifecycle signals
- **THEN** the design records that blocker or narrows supported coverage before implementation
- **AND** no untested live compatibility claim or guessed global shortcut replaces it

### Requirement: State freshness belongs to each observed tab

The isolated-world adapter SHALL read the live DOM for event-driven changes and
background-requested heartbeats. Each fresh read SHALL advance a sample counter
scoped to its document/port instance. A cached repeated sample SHALL NOT renew a
tab's freshness. Three seconds without a newly sampled report SHALL make that tab
unavailable, independent of reports from other tabs. Joined-root loss/replacement
SHALL invalidate the call epoch; control-node loss/replacement SHALL invalidate
that control revision even if its next observed value is unchanged.

#### Scenario: Another tab continues reporting while the selected tab is suspended

- **WHEN** the selected tab supplies no new sample for 3 seconds while other tabs report
- **THEN** the selected target expires and its controls become unavailable
- **AND** repeated cached samples and other tabs' heartbeats cannot retain its eligibility

#### Scenario: A control node is replaced with an identical-looking button

- **WHEN** a new DOM node replaces the captured mic or camera control
- **THEN** the old revision is invalid before another action can click the replacement
- **AND** the replacement needs a fresh observed control state
