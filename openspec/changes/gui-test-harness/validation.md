# Validation

Completed after wiring the generated Stream Deck fixture and adding keyboard, accessibility
and layout coverage.

- Full GUI suite: 95 cases, 13.3 s reported by Playwright / 13.7 s wall time, below the 60 s budget.
  The tracked Routing Activity casing failure remains expected and owned by `gui-content-fixes`.
- All 13 spec files also pass independently.
- WCAG 2.0/2.1 A/AA scans cover all 11 screens with details expanded; no violations were found.
  A deliberately unlabeled input proves the scanner rejects a new defect.
- Layout covers all 11 screens at 1280×820 and 800×600. Exact known overlaps between the narrow
  Stream Deck Dial 5 and its sticky save bar are owned by `gui-layout-polish`; new and obsolete
  expectations fail. A negative self-check verifies overflow, clipping and overlap detection.
- The fake host now selects dials with the provider's key-count offset. The new inspector assertion
  failed as KEY 3 before the correction and passes as DIAL 3 afterward.
- `scripts/check.ps1` passed: all Go tests, six model tests, browser tests, vet, OpenSpec validation
  and the canonical build. Snoofer shut down normally and restarted with its previous arguments.
  The final browser-only fake-host correction was then checked with every spec independently and
  the full browser suite. The Stream Deck fixture drift test also passed independently.
- Race detection was skipped by the check script because CGO is disabled. No toolchain changes
  were made. Hardware and native WebView2 dropdown acceptance remain unperformed.

Layout measurements cover initially visible content at scroll position zero; expanded details
and later sticky-scroll positions are not claimed as layout coverage. Screenshots are review
artifacts under `.local`, not committed baselines. The fake host's region mutation math remains
limited to the + XL scenarios; smaller-model rendering/navigation are covered separately.
