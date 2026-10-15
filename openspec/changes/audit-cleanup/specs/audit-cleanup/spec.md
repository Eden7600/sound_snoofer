## ADDED Requirements

### Requirement: Worker cleanup preserves live action behavior
The worker SHALL remove only confirmed unused publication state, presentation helpers and
the unused acknowledgement/deduplication and batch protocol identified in `audit-completion`.
Edit stale validation, device availability checks, save-before-apply,
feedback, callback outcomes, origin notice clearing and shutdown behavior SHALL remain unchanged.
The original acknowledgement/batch retention is superseded by the user's 2026-10-15
completion instruction. GUI result identities and native defaults acknowledgements SHALL remain.

#### Scenario: A device disconnects before a saved-choice edit
- **WHEN** a normal or VR source/output edit names a device no longer available
- **THEN** the existing validation rejects it without saving or applying partial choices
- **AND** its feedback and callback outcome remain available through the existing paths

#### Scenario: Saving a single edit fails
- **WHEN** persistence refuses an otherwise valid edit
- **THEN** the previous intent and routing remain active and the error is reported

#### Scenario: A profile changes or reconnects
- **WHEN** providers disappear, reconnect or change the active normal/VR profile
- **THEN** profile options, observation freshness and routing behavior remain available without
  computing the removed display-only publication fields

### Requirement: Shared decoding retains strictness and domain validation
Configuration and saved choices SHALL use the existing strict storage decoder. It SHALL reject
duplicate keys at any nesting depth, unknown struct fields, malformed/type-invalid input and
trailing JSON documents or junk. Caller defaults, normalization, domain validation, state tokens
and contextual error reporting SHALL be preserved.

#### Scenario: JSON contains an ambiguous duplicate
- **WHEN** an object, including one nested in an array, repeats a key
- **THEN** decoding fails before those values can become active configuration
- **AND** identical key names in distinct sibling objects remain valid

#### Scenario: Saved choices are malformed or invalid
- **WHEN** saved choices fail syntax, field or domain validation
- **THEN** the existing state error is retained and partially decoded intent is not installed

#### Scenario: Valid data relies on defaults
- **WHEN** valid configuration omits fields that have initialized defaults
- **THEN** those defaults and subsequent domain checks behave as before consolidation

### Requirement: Shared Go operations preserve caller contracts
Bus parsing SHALL preserve the accepted inputs and native bus mapping of each consumer.
First-nonempty selection SHALL preserve argument priority. Shared shutdown SHALL cancel before
waiting and honor the caller's context without changing worker resource ownership.

#### Scenario: A bus name has a noncanonical numeric suffix
- **WHEN** a gain/mute caller receives `A01` or `A+1`
- **THEN** it retains its existing rejection
- **AND** the echo parser retains its existing numeric-suffix acceptance without changing A1–A5 mappings

#### Scenario: A plugin cannot finish before its stop deadline
- **WHEN** the caller's stop context expires before the worker's done channel closes
- **THEN** Stop returns the context error after requesting cancellation
- **AND** successful joins still return normally

### Requirement: Native setup consolidation preserves build behavior
Shared toolchain setup SHALL preserve each script's command body, compiler settings, staging,
output location, self-tests and failure propagation. The canonical application output SHALL
remain `bin/snoofer.exe`, built through `scripts/build.ps1`.

#### Scenario: Toolchain setup or compilation fails
- **WHEN** a required tool is missing, vcvars setup fails or a compiler/self-test returns failure
- **THEN** the invoking script fails with useful context
- **AND** existing build-staging guarantees remain in effect

#### Scenario: Paths contain spaces
- **WHEN** the repository, toolchain or selected output directory contains spaces
- **THEN** quoting preserves those paths and produces the same intended artifacts

### Requirement: Native buffer sharing preserves validation strength
Shared audio-buffer helpers SHALL preserve the C ABI, bounded shape checks, per-channel
validation and overlap-safe pass-through copying. The monitor SHALL continue rejecting any
missing read/write channel before callback processing; AEC/neural error and fallback behavior
SHALL remain unchanged.

#### Scenario: A buffer has an invalid count or missing channel
- **WHEN** a buffer has invalid sample/channel counts or a required channel pointer is absent
- **THEN** the relevant caller rejects or falls back exactly as before
- **AND** helper consolidation does not weaken the monitor's all-channel validation

#### Scenario: Input and output alias or overlap
- **WHEN** a valid pass-through buffer uses in-place or overlapping channel storage
- **THEN** samples are preserved using overlap-safe copying without allocation or locking

### Requirement: Dead-code removal retains observable behavior
Removal of the confirmed unused media field, renderer wrappers, child exit storage, presence
constant and test helper SHALL retain live callers and meaningful behavior tests.

#### Scenario: The GUI child exits
- **WHEN** the GUI child process exits or its pipe closes
- **THEN** it is still waited for and the session closes and joins using the existing lifecycle

#### Scenario: Media and deck state refresh
- **WHEN** sessions reconnect or a deck redraws
- **THEN** session identity/transport and actual renderer output remain unchanged by removal of
  the unused current-session flag and renderer wrappers
