# Design

## Existing paths and corrected audit baseline
Audio controls are published by `plugins/audio/audio.go:controls`; the end-of-function loop
currently assigns `ChoiceLabels` and `SilentMics` to every selection. The shared GUI widget and
deck renderer already consume `OptionLabels`, so fix the provider instead of capitalizing
arbitrary text in either renderer.

`priorityView.Activity` embeds `config.Activity`, whose JSON keys are `check`, `silence_db` and
`silent_after_s`. The TypeScript migration deliberately isolates legacy uppercase reads in
`screens/routing.ts`. The generated live/preview fixture contains 2, -70 and 10 respectively.

The Stream Deck provider already labels pages by `Page.Name`, including custom names. The old
hand fixture caused the audit's `home` example; no production fix is required for that subfinding.
Likewise, current audio gain labels are Playback/Mic, so the old `mic gain` example is historical.
The remaining binding problem is technical plugin suffixes and ambiguous fixed-profile labels.

## Audio selection labels
Use control IDs to choose the relevant map in the existing provider publication path:
- Source/normal-source/vr-profile-source: Automatic, Off, configured microphone names, and the
  existing ` · Silent` annotation only on microphones being skipped by Auto.
- Mode/normal-mode/vr-profile-mode: Direct and Element.
- Monitor/normal-monitor/vr-profile-monitor: Off, Pre and Post.
- Record-tap: Pre and Post.
- Output/normal-output/vr-profile-output: the empty automatic choice displays Automatic; explicit
  device names remain exactly as supplied. Do not copy microphone labels onto playback choices.

Do not apply option maps to commands, statuses or toggles. Keep raw option values and saved choices
unchanged. Extend the existing provider label tests across active, Normal and VR controls, including
silent microphone labels and nonselection empty values. Browser checks cover Audio options and the
Routing playback Monitor cell, whose current `pre` expectation becomes `Pre`.

Audio fixture changes must be generated live and preview first, then the Stream Deck fixture:
its golden test reads `audio.live.json`, and its binding labels depend on audio publication.

## Activity fields
Read the typed lowercase fields directly; remove the legacy-case cast/comment and `test.fail`
annotation. Retain units, bounds and apply-on-change behavior. With Activity absent, Meter stays
Off and the existing empty threshold fields remain empty. Rendering, focus and field population
must dispatch nothing. Test all three initial values and all three edit payload field names;
retain the existing stale-list revision behavior and explicit zero for Meter Off.

## Audio card content
Publish `audio.playback:*` switches in the existing Routing group and label them with the
`sourceLabels` map already in `plugins/audio/outputs.go` (for example Computer Playback), falling
back to the source ID if it has no label. The
existing Audio recovery filter then excludes the duplicate switches. Keep their IDs, availability,
values and operations intact; the Routing matrix and existing saved deck bindings still use them.

AEC already publishes `Label: Retry Echo Cancellation`, `ShortLabel: Retry` and no Value.
`control()` overwrites command captions with Value, causing the empty caption. Render this one
action using the existing `command()` helper in an actions row inside Echo Cancellation. Its
button remains Retry, keeps availability/pending disabling, and sends the same `aec.retry` press.
Do not change generic command rendering or retry/recovery behavior to solve this local copy issue.

## Plugin names and binding context
Add an additive `PluginLabels` ID-to-label map to `ViewState`. Populate it in `app.Run` from the
registered `plugins` slice already available while publishing the snapshot; do not introduce a
second hard-coded catalog or a new Host metadata API. Update production/test wire types and the
hand-written plugin-state fixture. Missing/empty labels fall back to the ID, so absent metadata
and unknown plugins remain readable. Keep the existing ID ordering and action payloads.
Include PluginLabels among the existing structural render inputs so changed metadata cannot
leave headings or binding options on an old label; retain the existing rendering mechanism.

Use these names in Plugins headings and Third-Party Apps' Not monitored list. The binding selector
keeps the full provider Label and changes its suffix from plugin ID to registered plugin label.
Search continues to find displayed names and plugin IDs; retaining ID matching avoids removing an
existing search capability. Selected and unavailable bindings retain their existing fallback.
Option values and tooltips remain control IDs. Longer registered names and profile qualifiers can increase native listbox row height in Chromium; cap the Binding listbox height while preserving its native scrolling and wrapped labels, so these content changes do not put it under the save bar. Verify both supported sizes without expanding the layout exception list.

Audio's fixed Normal/VR controls currently share full Labels such as Processing while only their
ShortLabels distinguish profiles. Qualify their provider Labels with Normal/VR (Microphone,
Processing, Monitor and Playback) so binding choices stay distinct. Audio cards already supply
local label overrides, and compact ShortLabels remain unchanged. Test both profiles side by side.
Preserve custom provider labels and custom page names as written; never title-case user content.

## Validation and boundaries
Each implementation commit includes its focused Go/browser checks and any regenerated fixtures.
Add an assertion that the page selector renders Home and a custom page name while sending raw IDs;
this protects already-correct behavior instead of creating a production change for it.

Run type/lint/format/model/browser gates, fixture drift tests, Go tests/vet and strict OpenSpec
validation. Inspect affected GUI screens at both supported sizes and native-size deck renderer
output for changed option labels. Finish with the canonical build and packaged desktop smoke,
restoring the previous host launch. Do not claim hardware acceptance. Retain unrelated expected
layout exceptions and pending/error semantics.
