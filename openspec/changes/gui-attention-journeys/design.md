# Design

## Current implementation and boundaries

`Host.Status` returns Running, Disabled or arbitrary error text. A plugin can fail with text
such as Running; the GUI must not derive lifecycle from that string. Host already owns compiled
plugin metadata, desired configuration, instances and failed starts under its lifecycle mutex.
`app/tray_windows.go` publishes status, labels and saved Enabled state. The GUI has eight plugin
screens with four different missing/disabled checks, and Soundboard has no plugin gate.

`Connection.State` already describes integration health. `tone()` intentionally paints mute red
and recognizes words in Status, so it is unsuitable for navigation. Some provider failures are
only published as prose: AEC engine failure, App Audio backend errors, Soundboard scan/playback
errors and Stream Deck editor failures. These need a small semantic field, not more text parsing.

The implementation reaches host snapshot/tests, wire types and fixtures, provider publication
tests, navigation updates, screen entry checks and Plugins lifecycle actions. It preserves all
IDs, saved settings, command revisions, confirmation warnings and user configuration.

## 1. Typed lifecycle, retaining descriptive text

Add `PluginStates` to ViewState, with values `missing`, `disabled`, `starting`, `running` or
`failed`. Derive an independent snapshot in Host from the union of compiled and configured IDs:
missing means not compiled; disabled means compiled but not enabled; running means an owned
instance exists; failed means a retained start/dependency/validation failure; otherwise an
enabled compiled plugin is starting. Failure classification comes from `failed`, never the
error's wording. Keep `Plugins` as the unchanged descriptive text map and `Enabled` as saved
selection state. Include compiled plugins omitted from the config as disabled.

The snapshot is mutex-protected and copied, like Status. It does not make lifecycle asynchronous
or add a worker. Start currently holds the lifecycle lock, so starting may only be observable
before the first provider publication. Do not promise intermediate start progress.

Go and TypeScript wire definitions, the fake host, manual fixtures and transport tests must carry
the same field. Update existing lifecycle displays on Plugins and Third-Party Apps to consume
the typed state, including Not monitored. Do not retain a string-based classification fallback
in the newly packaged GUI; the host and GUI are built together.

## 2. One plugin gate and its recovery actions

The gate runs before normal screen construction, using this explicit readiness map:

| Screen | Plugin | First usable publication |
| --- | --- | --- |
| Audio | audio | audio.health |
| Routing | audio | audio.priorities with ViewData |
| Soundboard | soundboard | soundboard.status |
| Lights | hue | hue.status |
| App Audio | appaudio | appaudio.status with ViewData |
| Meetings | discord | discord.channel |
| Media | nowplaying | nowplaying.status with ViewData |
| Stream Deck | streamdeck | streamdeck.preview with ViewData |

Use a common panel with the registered plugin name (ID fallback), a short state and recovery
action. Missing: "Not part of this build", no Enable. Disabled: "Off", Enable <name>. Starting,
or a running plugin whose first required publication has not arrived: "Starting", with a Plugins
navigation button and no redundant Enable. Failed: "Could not start", full existing error text,
and a Plugins navigation button. No timeout changes Starting into an invented failure.

Enable submits the existing selection action with the exact plugin ID and true; it retains the
host's dependency closure, restart warning and explicit confirmation. It must not immediately
pretend the plugin is running. Buttons obey disconnection and the lifecycle action's busy state.
There is no automatic enable, retry or replay. Missing compiled metadata never produces an
impossible Enable action. Gate transitions update without navigation when host state changes.

Lifecycle gate/readiness transitions must be part of the structural key on every affected
screen. A provider disappearing or becoming failed/disabled must disable old actions immediately
and replace the gate safely; do not preserve a provider form across a lifecycle identity change.
Use the state-preservation helper for ordinary same-provider builds, with its removed-target
focus fallback. Keep selection dialogs and focus rules owned by the other changes.

The Plugins page gains the existing Retry Plugins action, enabled only while a compiled failed
plugin exists and that lifecycle group is free. It uses Kind retry (which retries failed
branches), not a new per-plugin API. Failed cards retain Disable; healthy cards retain Disable;
disabled compiled cards retain Enable. A missing plugin shows no enable toggle. No plugin
settings move onto Plugins. Retry failures stay visible through the existing outcome path.

Enabled-but-empty content remains the screen's ordinary empty state: no clips, no media sessions,
no apps or no remembered deck is not a plugin failure. A valid offline Deck editor stays usable.
Audio's optional AEC failure does not replace working Audio controls; Plugins still receives its
lifecycle marker and its failure is available there.

## 3. Narrow semantic attention sources

Add optional `Control.Attention` with the two recognized nonempty values `attention` and
`critical`; absence means no explicit marker. It is display metadata, omitted from JSON when
empty, copied with the control and excluded from action identity comparison like other
presentation-only telemetry. A metadata-only change must reach the view without invalidating
an otherwise identical action. Unknown values produce no marker. Do not change deck colors or
normal control tone as part of this change.

Publish it only at these existing canonical status sites, using the source conditions shown:

| Control | Current source condition | Severity |
| --- | --- | --- |
| audio.health | State.Error, State.StateError, qualified State.Stalled, or a nonempty Notice with NoticeKind Error | critical |
| audio.health | RecoveryPending with no critical condition | attention |
| audio.slot-1 through audio.slot-3 headers | slot.State is routing.OutputNoOutput | attention |
| aec.status | stats.Failed, releaseErr, retryErr, engineErr or statsErr | critical |
| aec.status | active processing with an unsupported sample rate | attention |
| appaudio.status | backendErr or editErr | critical |
| soundboard.status | scanErr or current diagnostic | critical |
| soundboard.status | current artwork failure without a critical condition; routing readiness failure already has a typed connection report | attention |
| nowplaying.status | winErr, bridgeErr, saveErr or a current session failure | critical |
| streamdeck.status | an editor failure at its existing error assignment sites, distinct from empty/Saved feedback and Dirty | critical |

Stream Deck retains a boolean/typed failure flag alongside current editor feedback, set at the
actual error source and cleared wherever that feedback clears or becomes Saved. Do not compare
the text to Saved: error text can equal it. Dirty alone is not a problem marker. Soundboard's
intentional Preview label and unavailable playback in preview are not failures; only a concrete
scan/artwork failure still displayed in preview can mark it. Do not flag a hidden diagnostic
that the destination does not show. Keep every producer's existing error lifetime.

Current Hue/Discord faults and setup use their typed connection reports. Pairing or individual
action failures already attributed by gui-action-outcomes stay local; this change does not turn
every command Status into a new global alarm or duplicate the pending/outcome mechanism.

Routing additionally derives attention from its existing structured view: a priority entry with
InUse false and more than one match is attention, matching its displayed Ambiguous badge.
Output allocation uses the exact provider OutputNoOutput state via the header metadata above;
Device present with Bus empty is insufficient because a disconnected device can have no held bus.
No match, an intentionally empty slot, disconnected optional hardware (with or without a held
bus) and silence-based microphone selection are neutral.

Connection classification reuses `connectionTone` and `stale`, with one deliberate distinction:
ordinary connecting alone is transient and does not mark the sidebar. Error and required-peer
disconnection are critical; attention and stale active reports are attention. Off, unknown,
unconfigured and optional disconnection are neutral except the explicit setup cases below.
Historical LastError on a recovered report never marks anything.

Setup exceptions: `hue.app-bridge` unconfigured means Pair is needed on Lights;
`discord.app-client` unconfigured means setup is needed on Meetings. Both are attention only
while their plugin is enabled/running. Missing optional media extensions and a closed Discord
client stay neutral. Unknown/unpublished provider state does not invent an alert.

## 4. Navigation mapping and presentation

| Source | Navigation target(s) |
| --- | --- |
| Failed lifecycle (including dependency failure) | Plugins; also its gated screen(s) when that screen displays the failure card |
| audio.health / aec.status | Audio |
| Routing structured ambiguity or audio.slot-N header attention | Routing |
| soundboard.status | Soundboard |
| appaudio.status | App Audio |
| nowplaying.status | Media |
| streamdeck.status | Stream Deck |
| hue bridge setup / discord setup | Lights / Meetings |
| Current typed connection issue or stale report | Third-Party Apps; its owning screen only when that screen displays the same current issue or recovery action |

For connection ownership, Audio owns audio reports (including AEC/engine recovery context),
Lights owns hue bridge and sync reports, Meetings owns Discord, and Soundboard owns playback
readiness. Stream Deck device reports currently only appear in Third-Party Apps, so device
connection failures mark that destination, not an editor that contains no failure detail.
Unknown/custom plugin connections mark Third-Party Apps only. Diagnostics does not receive an
aggregate marker merely because its full notice list contains text; the specific destinations
above are the recovery paths. AEC/VR lifecycle failures that lack their own gate mark Plugins.

Use at most one marker per navigation button, critical taking precedence over attention. A small
existing Lucide alert icon and the palette tokens carry visible severity. Preserve the button's
normal accessible name and append a concise accessible description such as "Needs attention" or
"Problem"; title text supplies the same hint. The marker itself is decorative, not another focus
stop or an action. Keep every navigation label readable at 800x600, especially Third-Party Apps.
If space is insufficient, place the marker in the existing nav icon area rather than clipping
text or widening the sidebar. Markers never blink, announce every poll or replace selection and
focus styling. Update them during all state updates, including unchanged polls for stale time.

Clicking or keyboard-activating the marked navigation button opens that ordinary screen, where
the reason and action are already visible. No sound, toast, automatic focus jump, count, read
tracking or persistent notification store is needed. The shared header retains Sending, VR
active and manipulation-paused badges only; disconnection retains its separate banner.

## Verification and integration

Provider tests prove explicit failures publish metadata and recovery clears it without parsing
error text; adversarial labels/errors include Running, Disabled, Saved, muted and Off. A registry
test checks metadata-only changes do not stale valid actions, and transport tests check delivery.
Regenerate affected audio/deck fixtures with their existing generators and run drift checks.

Browser tests cover all eight plugin gates, Enable plus cancel/confirm journey without replay,
Plugins retry with exact target semantics, missing and starting cases, failure/recovery while
remaining on the page, and enabled empty content. Marker tests cover critical precedence,
removal on recovery, keyboard navigation and typed connection/setup paths. Negative cases cover
mute, stack Off, disabled plugins, optional hardware including a disconnected output without a
held bus, preview, remembered LastError, ordinary
pending/connecting, VR override, dirty layout, empty library and no sessions. Inspect both
1280x820 and 800x600 screenshots and run axe/geometry checks. Run focused tests then the shared
GUI suite within its 60-second budget, strict frontend gates and strict OpenSpec validation.

Coordinate source ownership with gui-action-outcomes and gui-state-preservation; merge small
publication hunks after queued-completion edits. Root coordinates the canonical build and
packaged WebView2 inspection; this worker never independently stops or restarts the live host.
