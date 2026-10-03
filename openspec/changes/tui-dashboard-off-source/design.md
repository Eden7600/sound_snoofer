# Design

## Context

The existing separate enabled row accepts Space only, and pending/error details
are easily hidden by stale recorder messages. Source Off must use the established
master-off routing semantics, including zero mic B1 despite recorder conflicts.

## Decisions

Accept source `off` in config and sidecars. Normalize old enabled=false to Off;
normalize Off to enabled=false. Keep enabled in version-1 JSON for compatibility,
but source edits explicitly set enabled according to the selected source. Use
an `MicActive` intent predicate throughout routing and conflict exceptions so
direct callers cannot accidentally route source Off. Do not erase mode, monitor
or recording preferences. Show effective source Off and suppress fallback
selection/assignment and missing-mic warnings while Off. Existing occupied-strip
ownership checks may still block routing, visibly; do not overwrite unknown inputs.

Use trusted ANSI styles after sanitizing external text; honor NO_COLOR. Keep
Bubble Tea and current dependencies. Header shows live/preview, connection and
recorder state; tabs remain keyboard accessible. Dashboard groups microphone,
playback and recording controls with a highlighted selected row. Wide terminals
show a second status/help panel; narrow terminals use one column. Clamp selection
and keep it visible on resize, with explicit footer shortcuts. Retain Routing,
Devices and Events for detailed diagnostics. Status distinguishes requested Off
from observed zero sends and never labels unknown observation as disconnected.

Enter/Space activate any control; Start/Stop remain Enter-only to prevent an
accidental recording while cycling toggles. 0 selects Off directly in Controls.
Tab changes views, arrows/j/k select, page keys scroll diagnostics. Clear stale
recorder notices when another explicit action is submitted. Keep save-before-apply
and live ownership guards; no implicit writes from preview.

## Risks / Trade-offs

Legacy binaries do not understand source Off → rebuild and use the new binary;
keep backups before personal sidecar edits. Terminal colors vary → provide
NO_COLOR and width/height tests. API failure can block Off → status remains
pending/error, not a privacy guarantee. No global OS mic muting is promised.

## Migration

Existing enabled sources are unchanged; old disabled choices display Off.
Preferences persist on explicit edits. Do not change the user's current live
audio or saved source just to demonstrate the redesign.
