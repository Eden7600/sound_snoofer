# Focused audit cleanup

## Why

Repository audit findings [19–20](../../../docs/repo-audit.md) and
[27–31](../../../docs/repo-audit-minor.md) identify unused state, repeated helpers and
missing strict-decoder coverage. Inspection of the current tree confirms useful cleanup,
but several proposed deletions or merges need narrower scope:

- The original checkpoint retained worker acknowledgement and batch-edit paths pending
  `gui-action-outcomes`. The user's 2026-10-15 completion instruction supersedes that retention
  in `audit-completion`: GUI results now use callbacks, and no production caller uses those paths.
- Ack-based worker tests also protect live feedback, selection validation, restart
  confirmation and latency. Those tests are not disposable with an implementation detail.
- Bus parsers, native buffer validators and thumbnail encoders are not identical.
- The original checkpoint retained local thumbnail and `onOff` helpers. The user's
  `audit-completion` instruction supersedes that choice while preserving caller policies.

## What Changes

- Remove write-only worker publication fields and unused helpers while retaining edit
  validation, persistence and feedback. The original batch/acknowledgement retention is
  superseded by `audit-completion`; preserve callback outcomes and origin notice clearing.
- Test `storage.Decode`, reuse it for configuration and saved choices, and replace config's
  forwarding journal helpers with direct storage calls.
- Share hardware-bus parsing while retaining each caller's current accepted inputs. Replace
  four `firstNonEmpty` implementations with standard-library `cmp.Or`.
- Share only the identical cancel-and-wait operation used by seven plugin `Stop` methods.
- Consolidate MSVC discovery and command-file setup across the six audited build/probe
  scripts, preserving each command body and the completed build-staging work.
- Put common native buffer shape/channel/copy helpers in the existing `audio_buffer.h`,
  retaining the monitor's stricter all-channel validation.
- Remove the confirmed small dead-code items in finding 30, updating the tests that still
  protect live behavior.

`audit-completion` shares only thumbnail fit/encode through `internal/artwork.Thumbnail`,
keeping 128px Catmull-Rom and 64px ApproxBiLinear policies and local input/GIF handling.
Discord and Soundboard use the exact shared `snoofer.OnOff` conversion at all eight call sites.

## Impact

The affected areas are `internal/control`, `internal/config`, `internal/storage`, routing
and gain/echo consumers, plugin lifecycle methods, native build scripts and buffer helpers,
plus the specific dead-code sites in finding 30. No settings migration, routing change,
GUI behavior change, dependency addition or new lifecycle framework is planned.

This proposal is planning only. Commit it before implementation. Native script/media work
must follow the completed `audit-hardening` changes and be reviewed against their final
source and tests. Finding 33 and the view-state/action-outcome designs are separate.
