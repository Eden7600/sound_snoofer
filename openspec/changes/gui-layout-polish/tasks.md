# Tasks

- [x] 1. Review current callers, UI contract and stable-state annotations; commit this validated plan before implementation.
- [x] 2. Audio/meter checkpoint: content-height cards, nearby live switch labels, Engine/Interface grouping and noncolliding scale labels. Verify both sizes and unchanged meter/state semantics.
- [x] 3. Routing checkpoint: local section navigation, compact candidate rows, separate slot management and conditional microphone Apply. Cover long names, exact action payloads, stale edits and keyboard commitment at both sizes.
- [x] 4. Deck/Soundboard checkpoint: aligned selectors, non-overlapping Save row, shared-edit scope and no-match state. Preserve draft/navigation/filter restoration and remove only resolved geometry exceptions.
- [x] 5. Diagnostics checkpoint: grouped/contextual notices, conservative informational filtering, shared clipboard code and compact Engine card. Cover real errors, unavailable statuses, VR duplication, copy fallback, live updates and existing restart/startup actions.
- [x] 6. Update UI contract and audit status; run type/lint/format, model, full GUI and strict OpenSpec checks. Inspect affected screens at 1280×820 and 800×600.
- [x] 7. Coordinate canonical build and packaged WebView2 acceptance, restore the previous launch configuration and record remaining hardware checks honestly.

- [x] 8. Native compact-sidebar checkpoint: reproduce classic-scrollbar brand clipping at the approximately 784×562 client viewport; apply the smallest existing-brand utility adjustment; verify text/image bounds, no horizontal overflow and last-item navigation with a reserved gutter, retain existing viewport checks, and inspect the packaged outer 800×600 window using its measured client size.

2026-10-15 packaged evidence: canonical builds and prior host restoration are recorded in
docs/gui-audit.md. Audio, Routing, Soundboard, Deck and Diagnostics were inspected at both
outer sizes. After 9065ff7, the resumed outer 800×600 inspection confirms full branding with
the classic scrollbar. End from Audio activates Diagnostics; Up then reaches Third-Party Apps
and Plugins with correct focus. Compact navigation/content inspection emitted zero action-log
bytes. The 293-test browser suite and strict frontend gates pass. Compact fixture PID 63112
closed via EOF with exit code 0 and zero actions, completing task 7. The later wide fixture
PID 63004 also closed via EOF with exit code 0 and zero actions. The normal canonical build
passed after the diagnostic DevTools attempt; production is restored as PID 48916 with the
previous bin/snoofer.json configuration. Physical acceptance remains separate.

- [ ] 9. Shared-notice checkpoint: reproduce the floated Dismiss misalignment with short/multiline geometry checks at both sizes, replace the float with a contained centered row, update the UI contract, and pass focused layout, dismissal/focus, recovery and frontend checks.
- [ ] 10. Coordinate canonical packaging and inspect short/wrapped shared notices in WebView2; restore the prior host configuration and record the native result.
