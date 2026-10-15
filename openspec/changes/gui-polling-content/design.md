# Design

Read the existing screen updater closures, structural rebuild rules and tests before
editing. App Audio's fixed detail labels are Programs, Devices, Processes, Sessions,
Rule and Heard. Create their dt/dd nodes once with the existing classes. Each update
computes the same displayed values and assigns a dd's text only when it differs.
Do not guard/rebuild the whole list with a combined key: the changing Heard value
would still destroy selection in an unchanged Programs row. Keep genuine value
changes visible and preserve the existing placeholders.

Media's icon button initially contains play. Store its current icon state in the
card's existing closure and replace its child only on a play/pause transition.
Continue updating accessible labels, capabilities, action feedback, artwork and
playback progress independently. Do not freeze an entire card or add a helper API.

Focused tests select actual text in App Audio More, advance polling and relative
time, and assert selection survives while Heard changes. Changed device/process
details must still refresh. A Media test proves unchanged polling/metadata retains
the same SVG, a playback transition updates icon and accessible label, and clicking
still dispatches exactly one request. Existing progress and state-preservation
tests remain relevant. Run new regressions before the source fix and retain the
failure evidence, then run focused tests and frontend gates/full browser suite.

This enforces the existing stable-live-update contract. Preserve layout and visual
classes; inspect the affected screens at the normal and minimum viewport through
the installed headless Chrome fixture. No desktop input or physical test is needed
to establish these DOM properties. Canonical build/deployment remains separate
from browser evidence and must preserve the running host's launch configuration.
