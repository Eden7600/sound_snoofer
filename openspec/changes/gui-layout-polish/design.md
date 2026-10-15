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

The copy regression must wait for the changed notice to be rendered before copying,
then wait for a subsequent changed notice before checking preserved focus/feedback.
Fixture edits alone do not deliver a poll; this makes the intended regression
deterministic without changing runtime polling or adding fixed delays.

## Verification and coordination
Land commit-sized screen checkpoints. Add focused pre-fix regressions for geometry/behavior, inspect rendered screenshots at both sizes, and run affected keyboard, dropdown, state-preservation and accessibility journeys. Provider payload assertions guard the moved Routing editors. Finish with strict frontend, model, full isolated GUI and OpenSpec checks. The root agent coordinates the canonical build, packaged WebView2 inspection and restoration of the previous host launch; the user has lifted the meeting-time running-host restriction. Do not claim hardware acceptance from browser screenshots.

## Compact sidebar with a classic scrollbar
The native 800×600 minimum describes the outer window; its client area is smaller. The observed capture suggests approximately 784×562 CSS pixels of content, to be confirmed in packaged acceptance. The sidebar remains 150px below its existing breakpoint. Its padding, brand margins, 42px mascot and wordmark leave too little room when a classic vertical scrollbar reserves about 17px. A full-width headless viewport and control-only bounds checks do not establish that the brand fits.

Use the smallest compact-breakpoint utility adjustment to the existing brand row's margins, gap or mascot size that keeps both the mascot and the complete wordmark within the sidebar's usable content width. Preserve the existing branding, sidebar width, typography elsewhere and vertical scrolling; do not hide the scrollbar, clip or truncate the wordmark, or introduce another scroll container.

Add a focused regression at 784×562 with an explicit classic-scrollbar gutter, verifying that the test actually reserves the gutter rather than relying on platform overlay-scrollbar defaults. Measure the mascot and the wordmark's text range against the sidebar's content bounds, and require no horizontal sidebar overflow. Scroll or keyboard-navigate to Diagnostics and verify the final navigation action remains reachable and visible without dispatch. Retain the existing 800×600 and 1280×820 checks. Record the measured native client size during packaged acceptance and inspect the corrected native sidebar at the outer minimum.

## Shared notice alignment
The shared #notice uses a floated Dismiss button beside a text node. The button's padding makes it taller than the text line, but the float does not determine the notice's normal-flow height; a short error therefore leaves the button visibly low inside the border.

Use existing Tailwind utilities for one flex row with vertically centered children and a gap. Wrap the message in a flexible, min-width-zero span and keep Dismiss from shrinking. The existing hidden attribute rule remains authoritative. Preserve the alert role, full error text and line breaks, notice colors and dismissal handler. Keep the rendered-message cache so an unchanged poll retains the same focused Dismiss button; do not alter per-action feedback or connection recovery.

Add a focused geometry regression at 1280×820 and 800×600 for a short error and a long multiline error with an unbroken identifier. Measure Dismiss against the notice's vertical center and padded bounds, require no horizontal overflow, and verify dismissal sends nothing. Demonstrate the short-message failure before the fix; retain the existing unchanged-poll focus, dismissed-notice and recovery tests. Inspect both message shapes in the packaged renderer during the root-coordinated canonical build/restore checkpoint.
