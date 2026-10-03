# Design

## Context

See proposal.md. Voice intent enabled already controls Discord, Element send,
monitoring and mic recording. B1 is optional and recorder conflict freezes it.

## Goals / Non-Goals

Make that intent the clearly named master switch, including all sends of the
three configured mic strips and AUX return. Do not add another overlapping
enabled flag, mute a bus master shared with computer audio, detach devices,
change ASIO patches, or operate recorder transport.

## Decisions

Reuse saved `enabled` and row key `voice`; rename the source row Microphone source
and the switch Microphone, displaying On/Off. This preserves existing saved
intent and reset behavior. A separate boolean would make two redundant gates.

Off overlays zero B1 on Strip[0,1,2,6] even without a recording profile or when
recorder-specific reconciliation is frozen. Existing voice rules already zero
their A1–A5/B2/B3 sends. Mark those B1 operations with the same transition/drift
verification as other sends. The controller permits only these zero writes to
bypass recorder conflicts; no positive sends or recorder preparation bypass it.
This explicitly narrows recording-control's B1 freeze/unmanaged behavior for
master Off. On returns to normal configured ownership (unmanaged custom B1
sends are not restored automatically).

Retain normal live debounce and verification: the TUI shows pending until mixer
readback converges. Dry-run saves the choice and previews only. Master Off does
not promise hardware microphone privacy or mute apps that capture Volt directly.

## Risks / Trade-offs

External writes or API failures can prevent disconnection → expose pending/error
instead of claiming applied. Recorder conflicts may still appear as diagnostics
even after microphone sends successfully become zero. Unknown device ownership
can block the normal planner and must remain visibly an error.

## Migration Plan

No schema migration; existing enabled=false becomes Mic Off. Keep user's current
choice. On restores configured preferences, including current device fallback.
