## ADDED Requirements

### Requirement: Only changed presentation state crosses the pipe
Each controls-child session SHALL suppress unchanged presentation frames, while retaining every change represented by the JSON view state, including telemetry excluded from control revisions. Explicit focus requests SHALL still be delivered. Transport preparation SHALL NOT mutate provider snapshots or advance its successful baseline after an encoding failure.

#### Scenario: Repeated idle state
- **WHEN** the complete presented state and asset maps remain unchanged
- **THEN** no additional presentation frame is queued
- **AND** an explicit focus request still shows the window without inventing a state change

#### Scenario: Telemetry changes without a revision change
- **WHEN** only meter or progress data changes and control revisions remain equal
- **THEN** the latest telemetry is sent in a new version of the stripped state
- **AND** unchanged artwork and animations are omitted

#### Scenario: Invalid snapshot followed by recovery
- **WHEN** encoding a snapshot fails and a later valid snapshot has the same proposed asset changes
- **THEN** the failure is reported and the valid snapshot still delivers those assets
- **AND** neither attempt modifies the provider's controls or assets

### Requirement: Complete asset replacements survive coalescing
Still artwork and animations SHALL cross the pipe as complete maps only when changed or initializing a session. Omission SHALL retain a cached map; an explicit empty map SHALL clear it. Animation comparison SHALL include every frame and delay. Queue replacement SHALL preserve the newest complete state, required map replacements and any pending focus request.

#### Scenario: Asset changes or disappears
- **WHEN** an artwork entry is added, replaced or removed, including removal of the last entry
- **THEN** the next changed asset map replaces the receiver's entire map, including an empty-map clear
- **AND** later telemetry frames do not repeat it

#### Scenario: A queued map is superseded
- **WHEN** state with changed assets is replaced in the one-slot queue by newer state without asset changes
- **THEN** the newest state retains the pending complete maps and focus request
- **AND** a newer explicit map, including an empty map or a reversion to earlier artwork, supersedes an older map

#### Scenario: Later animation frames change
- **WHEN** an animation's later image or delay changes while its first artwork stays equal
- **THEN** the complete updated animation map reaches the desktop and visible animation playback refreshes
- **AND** a delayed old lookup or playback timer cannot restore the earlier animation

### Requirement: Polling returns only state the caller needs
The private desktop polling API SHALL return no snapshot for an unchanged caller version and otherwise return the latest stripped state. It SHALL supply the full still-art map only when the caller requires initialization or has missed a map replacement. Animation frames SHALL remain outside polled state. The frontend SHALL preserve the existing hydrated control shape and local update behavior.

#### Scenario: Unchanged poll
- **WHEN** the caller already has the desktop's current version
- **THEN** polling returns null without resending state or artwork
- **AND** local progress, countdowns, meter decay/expiry, pending timeouts and deferred rebuilds after blur continue

#### Scenario: Poller misses intermediate frames
- **WHEN** the desktop receives an artwork replacement followed by telemetry before the next browser poll
- **THEN** the response includes the latest state and complete current artwork map
- **AND** the next telemetry-only response omits that map once the caller has accepted its version

#### Scenario: Controls or browser reopens
- **WHEN** a controls child starts or a fresh browser context requests version zero
- **THEN** it receives complete current state and assets, including empty maps
- **AND** an old or unknown client version cannot leave stale artwork visible

#### Scenario: Native smoke supplies legacy state
- **WHEN** the isolated native smoke sends an unversioned full State frame, with or without focus
- **THEN** the bundled GUI continues to show the supplied controls and emit the same revisioned actions
- **AND** inline legacy artwork is accepted without changing the smoke script

### Requirement: Transport efficiency preserves application behavior
The change SHALL retain configuration reload and stale-save behavior, provider contracts, action validation, polling cadence and error recovery. Automated checks SHALL demonstrate payload reduction for a representative artwork-heavy snapshot without relying only on revision changes or machine-specific timing thresholds.

#### Scenario: External configuration edit
- **WHEN** a configuration file is externally changed or becomes invalid
- **THEN** the existing reload, enablement, confirmation and error behavior is retained
- **AND** action handling still reloads and validates saved revisions as before

#### Scenario: Representative payload measurement
- **WHEN** a 60-control snapshot with 64 KiB artwork per control is measured across initial, unchanged, telemetry-only and artwork-change publications
- **THEN** measurements report pipe and IPC payload bytes and benchmark allocations
- **AND** unchanged publication emits nothing, unchanged IPC returns null, and telemetry-only payloads contain no repeated image bytes
