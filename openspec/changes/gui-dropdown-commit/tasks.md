# Tasks

- [x] 1. Confirm the defect in native WebView2 and specify the commitment contract.
- [ ] 2. Add focused failing browser regressions; implement the shared select-event helper and
  wire semantic provider selections and Routing's direct live selects. Preserve captured revisions.
- [ ] 3. Verify pointer/native-popup/keyboard cases, cancellation and stale input; retain deck
  draft and Add-form journeys. Distinguish closed-select Tab cancellation from native-popup Tab
  confirmation; cover Escape/Alt+Up/outside-click popup cancellation and subsequent editing.
  Update the UI contract and audit status. The held-Tab probe recorded in design.md establishes
  why native confirmation cannot use a click-detail, focus or timing heuristic.
- [ ] 4. Run frontend gates and GUI checks, canonical build, isolated native Processing/Monitor
  acceptance, then restore the previous host launch. Record hardware acceptance separately.
