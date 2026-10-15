# Design

## Scope and behavior
Reuse native HTML selects. A shared DOM helper owns only commitment events; the caller still
owns payload construction, availability, pending state and the revision captured on focus.
Use the helper for the shared semantic selection widget except Stream Deck controls, and for
Routing microphone source, interface channel mode, Activity Meter and existing output device.
New microphone/interface/output forms already have an explicit Add action and need no change.
Numeric inputs, sliders, checkboxes, toggles and the deck binding listbox are outside this fix.

Define these interactions explicitly:

| Interaction | Result |
| --- | --- |
| Closed select arrows, Home/End, PageUp/PageDown or typeahead | Browse a local choice; send nothing |
| Enter on a staged keyboard choice | Commit the displayed choice once and leave the editor |
| Escape or blur/Tab without commitment | Restore the observed choice; send nothing |
| Pointer opens native popup and selects an option | Commit once through the native selection event |
| Keyboard opens native popup, then confirms/cancels | Confirm once or cancel with no action |
| Same choice selected again | No duplicate request |

Distinguish keyboard browsing from native popup commitment with the smallest event handling
that passes both Chromium and WebView2 checks. Do not replace the widget with a custom listbox
or require an extra click after a pointer selection. Prevent a keydown commit followed by a
native change event from sending twice. Include typeahead and Alt+Down/Space popup journeys;
testing only ArrowDown on a closed select is insufficient.

## State and stale edits
Keep the keyboard draft while focused; regular state updates already avoid overwriting focused
semantic fields. Cancelling restores the current observed value where available. A changed
options list may still defer rebuilding until blur under the existing rule.
Commit using the revision captured on focus. Routing list edits retain the drawn configuration
hash check and stale revision-zero sentinel. Never silently retarget an old choice to a new
snapshot. Existing pending disabling and provider rejection remain intact.

## Verification
Add focused browser tests for Processing and Monitor, Routing's direct selects and a non-audio
semantic select. Verify navigation/typeahead emits nothing, Enter emits exactly one correct
request, Escape/Tab cancels, pointer selection commits once, and stale edits keep their old
revision. Retain existing Stream Deck keyboard/draft journeys and Add-form behavior.

The native check must exercise the canonical executable with an isolated `__controls` fixture,
inspect emitted pipe actions and close it gracefully. Record pre-fix evidence (Processing
Down emitted set/element) and verify Processing and Monitor after the fix. Check native popup
confirmation/cancellation as well as closed-select browsing. No production debug endpoint or
new public command is added for testing; native UI automation is sufficient.
