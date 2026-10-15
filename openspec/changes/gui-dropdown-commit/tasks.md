# Tasks

- [x] 1. Confirm the defect in native WebView2 and specify the commitment contract.
- [x] 2. Add focused failing browser regressions; implement the shared select-event helper and
  wire semantic provider selections and Routing's direct live selects. Preserve captured revisions.
- [x] 3. Verify pointer/native-popup/keyboard cases, cancellation and stale input; retain deck
  draft and Add-form journeys. Distinguish closed-select Tab cancellation from native-popup Tab
  confirmation; cover Escape/Alt+Up/outside-click popup cancellation and subsequent editing.
  Update the UI contract and audit status. The held-Tab probe recorded in design.md establishes
  why native confirmation cannot use a click-detail, focus or timing heuristic.
- [x] 4. Run complete frontend/GUI checks, canonical build and isolated native Processing/Monitor
  acceptance, then restore the previous host launch. Source gates, canonical builds and all
  specified native dropdown paths pass; the normal host is restored with its previous config.
  Record hardware acceptance separately.

2026-10-15 native acceptance: closed browsing/typeahead cancellation and explicit Enter
commitment pass; popup Enter/Tab commit once, while Escape/Alt+Up/outside-panel click restore
the observed selection without dispatch. On normal-build fixture PID 42296, the user clicked
Monitor Pre to Post and exactly one request was captured: audio.normal-monitor, revision 36,
operation set, value post; action ID fuiquf-1ajyeq0-acfk1b-1. Production is restored as
PID 48916 with --config bin/snoofer.json. These isolated requests do not claim audio effects.
