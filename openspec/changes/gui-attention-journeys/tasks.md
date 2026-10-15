# Tasks

- [x] 1. Review this complete plan against current lifecycle, provider publications and GUI
  ownership. Strictly validate before implementation; coordinate overlapping files with the
  action outcomes and state preservation changes.
- [x] 2. Commit checkpoint: add typed lifecycle snapshot and PluginStates wire field; retain
  status text. Cover missing/disabled/starting/running/failed, dependency failures, snapshots and
  failure text matching healthy labels. Update fixture/test types and transport coverage.
- [x] 3. Commit checkpoint: add narrow provider attention metadata and tests, preserving action
  identity and existing error lifetimes. Cover every producer in the design, metadata-only
  delivery/revision behavior and recovery; publish explicit Media WindowsError and verify its
  existing problem line never hides an error named Connected. Regenerate audio/deck fixtures
  and run drift checks.
- [x] 4. Commit checkpoint: unify the eight plugin screen gates and Plugins recovery actions.
  Cover exact Enable selection and confirmation, cancellation, missing/starting/failed states,
  lifecycle transitions without navigation, busy/reconnect refusal and healthy empty content.
  Preserve legacy unversioned fixture rendering through the documented absent-map normalization,
  with typed-map authority and unchanged revisioned action coverage.
- [x] 5. Commit checkpoint: add sidebar classification/mapping and accessible markers. Cover
  each positive source and the negative cases listed in the design; verify marker-to-screen
  keyboard journeys and no dispatch from navigation. Keep the header transient-only.
- [x] 6. Update UI contract/audit status; run Go tests/vet for affected packages, strict frontend
  gates, model/full GUI tests within 60 seconds, axe and layout checks at both viewport sizes,
  then strict OpenSpec validation. Inspect screenshots before marking this task done.
- [x] 7. Root-coordinated canonical build and packaged WebView2 acceptance for marker visibility,
  plugin gates, keyboard navigation and confirmation cancellation; restore the previous launch.
  Do not exercise real plugin enable/restart or hardware writes merely to produce a screenshot.

2026-10-15 packaged evidence: disabled Audio and captured host-confirmation cancellation
passed earlier. The resumed native fixture shows Starting Audio, then failed Audio with its
critical marker and exact detail "Native acceptance: audio service unavailable". The Plugins
link works and Retry is available. Home-key navigation activates the marked Audio screen and
shows its reason; all navigation leaves the action log at zero bytes. Critical Audio and
warning Routing markers are visible. These are isolated presentation/interaction checks,
not real plugin enablement, retry, restart or hardware effects. The prior host configuration
remains restored.
