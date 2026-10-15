# Design

## Existing behavior and boundaries
The harness includes geometry checks at 1280×820 and 800×600, plus one exact Deck save-row overlap exception. Remove that exception only after the actual geometry is fixed. State-preservation adds stable element keys; retain them while moving elements. The action-outcomes change owns pending/error behavior, so presentation work must not replace its request plumbing.

## Audio and shared meters
Set the Audio card grid to align items at their start. Put the Engine and Interface status rows together above live controls and gain strips. Keep the three live switches in compact label/action pairs rather than stretching each label away from its switch. Preserve their accessible names and the required Engine row on Audio and Diagnostics.

Keep -60, -30, -12 and 0 scale labels. Show the -3 label only when the meter container is at least 400px wide, using a Tailwind container query. Meter colors, positions, peak/expiry behavior and exact dB readouts are unchanged. Verify visible label bounds at both viewport sizes and retain the wide-strip -3 label.

## Routing
Add local section links for Outputs, Microphones, Interfaces, Playback, Mic Priority and Activity. Use native anchors to stable focusable section IDs; following them sends no action. Keep section order and provider payloads.

Candidate rows get an explicit three-column arrangement: priority number, flexible device/pattern field, movement/removal actions. Driver/match detail follows below, so action buttons stay beside their own entry at supported widths. Long names wrap inside their column. Do not change microphone/device priority order, identities or disable rules.

Keep the Outputs table for source switches, destination captions and state only. Move existing slot device/name/remove controls into a responsive Output Devices group below the table, keyed by the same slot control/stable ID. Existing explicit device commit, rename, add/remove and stale configuration checks remain unchanged. A microphone name's Apply button is visible only while the name differs; Enter applies and Escape cancels as before. Never turn blur into a write.

## Stream Deck and Soundboard
Give Device/Page selectors the same stacked field layout within equal-width wrappers. Move the Save/Discard row near the selectors in normal flow, with wrapping at narrow sizes. It must not overlay keys, dials or the Page panel at the top, middle or end of the document. Sticky behavior is not a UI-contract requirement; avoid a new nested scroll area just to preserve it.

Label the shared-edit switch Edit All Pages and show the concise scope note Same position on every page. Preserve the selected slot, shared/page distinction, all draft semantics, revision capture and keyboard navigation.

Soundboard shows No matching clips when the library is nonempty but the search hides every clip. Clearing search restores clips and hides that message. Keep the existing missing-library state separate; searching never dispatches. Register search restoration before computing the empty state.

## Diagnostics notices
Keep Startup as its own card. Keep Engine health and restart/confirmation commands on Diagnostics; remove duplicate Interface, ASIO warning and Automatic Recovery controls there because Audio already owns those settings/details. No capability is removed from Audio.

Build a small presentation-only notice projection from controls. Never infer action validation from display strings. Ignore hidden controls and empty statuses; exclude core.startup because its full status is already in Startup. Suppress known descriptive AEC timing/healthy-echo detail and held-send count from Current Notices, while retaining their owning-screen details. Keep unrecognised statuses conservatively so failures are not hidden. Collapse only exact VR override statuses from Normal-profile controls into Normal profile: VR override, and omit equivalent active-profile aliases. Preserve distinct labels/statuses for other controls. Actual Pending, Unknown, fallback, unavailable reasons and failures remain visible, even if the provider is unavailable.

Group notices by the control ID's plugin prefix, using PluginLabels with raw-ID fallback. Copy Notices copies the displayed groups, labels and complete status strings in stable order using the existing clipboard/fallback implementation shared with Third-Party Apps. Keep Copy and its local status outside the poll-refreshed list so polling cannot steal its focus or erase feedback. Disable Copy when there are no notices. Do not clear host/action errors as a side effect of rendering this list.

## Verification and coordination
Land commit-sized screen checkpoints. Add focused pre-fix regressions for geometry/behavior, inspect rendered screenshots at both sizes, and run affected keyboard, dropdown, state-preservation and accessibility journeys. Provider payload assertions guard the moved Routing editors. Finish with strict frontend, model, full isolated GUI and OpenSpec checks. The root agent coordinates the canonical build, packaged WebView2 inspection and restoration of the previous host launch; the user has lifted the meeting-time running-host restriction. Do not claim hardware acceptance from browser screenshots.
