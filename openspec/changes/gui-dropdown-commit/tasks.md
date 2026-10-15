# Tasks

- [x] 1. Confirm the defect in native WebView2 and specify the commitment contract.
- [x] 2. Add focused failing browser regressions; implement the shared select-event helper and
  wire semantic provider selections and Routing's direct live selects. Preserve captured revisions.
- [x] 3. Verify pointer/native-popup/keyboard cases, cancellation and stale input; retain deck
  draft and Add-form journeys. Distinguish closed-select Tab cancellation from native-popup Tab
  confirmation; cover Escape/Alt+Up/outside-click popup cancellation and subsequent editing.
  Update the UI contract and audit status. The held-Tab probe recorded in design.md establishes
  why native confirmation cannot use a click-detail, focus or timing heuristic.
- [ ] 4. Run complete frontend/GUI checks, canonical build and isolated native Processing/Monitor
  acceptance, then restore the previous host launch. Focused browser checks and frontend gates pass;
  canonical/native acceptance is deferred while the user prohibits running-host changes.
  Record hardware acceptance separately.
