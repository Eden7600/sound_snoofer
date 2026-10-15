## ADDED Requirements

### Requirement: Typed plugin lifecycle

The host SHALL publish plugin lifecycle independently from descriptive status text. The snapshot
SHALL distinguish missing, disabled, starting, running and failed plugins using host-owned
metadata, configuration, instances and failure state, and SHALL include compiled plugins omitted
from configuration. Existing status text and saved enablement SHALL remain available.

#### Scenario: Failure wording resembles healthy status
- **WHEN** an enabled plugin fails with the error text Running or Disabled
- **THEN** its lifecycle is failed and the original error remains visible
- **AND** the GUI does not treat it as healthy or intentionally disabled

#### Scenario: Compiled plugin absent from configuration
- **WHEN** a compiled plugin has no saved configuration entry
- **THEN** its lifecycle is disabled and the GUI offers its existing Enable action

#### Scenario: Configured plugin absent from the build
- **WHEN** configuration references a plugin not compiled into this build
- **THEN** its lifecycle is missing and no impossible Enable action is offered

### Requirement: Consistent plugin screen gates

Audio, Routing, Soundboard, Lights, App Audio, Meetings, Media and Stream Deck SHALL show one
consistent gate for their missing, disabled, starting or failed provider. Gate content SHALL use
the registered label with plugin ID fallback, retain failure detail and lead to a valid recovery
action. A running provider awaiting its first required screen publication SHALL show Starting,
without an invented timeout failure. Existing healthy empty-content states SHALL remain distinct.

#### Scenario: Disabled plugin enable journey
- **WHEN** the user activates Enable on a disabled compiled plugin's screen
- **THEN** exactly one existing selection action identifies that plugin with Enable true
- **AND** the existing dependency and restart confirmation is retained
- **AND** cancelling sends no confirmation or replayed selection

#### Scenario: Failed plugin recovery journey
- **WHEN** the user opens a failed provider's screen and activates Plugins
- **THEN** Plugins shows the failure and the existing Retry Plugins action
- **AND** retry uses the existing failed-branch lifecycle operation without a new per-plugin API

#### Scenario: Provider availability changes in place
- **WHEN** the active screen's provider is disabled, fails, disappears or becomes ready
- **THEN** the screen updates its gate without requiring navigation
- **AND** unavailable old controls cannot dispatch and removed focus uses the shared safe fallback

#### Scenario: Empty enabled provider
- **WHEN** an enabled ready provider has no clips, sessions, apps or connected deck
- **THEN** the GUI shows the appropriate content empty state or usable offline editor
- **AND** it does not claim the plugin is disabled or failed

### Requirement: Semantic attention metadata

Provider attention SHALL be classified from explicit current failure/degradation conditions,
typed connection reports or structured Routing data. Optional Control.Attention SHALL recognize
attention and critical and SHALL be presentation-only. The producer list and conditions in the
design SHALL bound this change; generic Value or Status text matching SHALL NOT create markers.

#### Scenario: Provider fault and recovery
- **WHEN** a listed provider publishes a concrete failure and subsequently recovers
- **THEN** its canonical status control gains and then clears the declared attention metadata
- **AND** descriptive labels, values and existing error lifetime remain unchanged

#### Scenario: Metadata changes independently
- **WHEN** only a control's attention metadata changes
- **THEN** the changed metadata reaches the GUI
- **AND** an otherwise identical action retains its control identity and revision

#### Scenario: Ordinary states are not problems
- **WHEN** a user mutes audio, disables Mic Stack, uses preview, leaves an optional device absent,
  has a dirty layout, enters VR override or waits for an ordinary command
- **THEN** those facts alone produce no navigation attention marker

### Requirement: Attention leads to a useful destination

Navigation SHALL show at most one marker per destination containing a current classified issue
or recovery action, using the design's explicit mapping. Critical SHALL take precedence over
attention. Markers SHALL clear when the condition clears and SHALL preserve button names,
keyboard navigation, focus, selection and readable labels at both supported viewport sizes.
Activating a marked navigation button SHALL only navigate. The shared header SHALL remain
transient-only and host reconnect feedback SHALL remain separate.

#### Scenario: Keyboard journey to a failure
- **WHEN** a plugin failure marks its gated screen and Plugins
- **THEN** keyboard activation opens the chosen screen with its current failure and recovery path
- **AND** the marker itself adds no focus stop or provider dispatch

#### Scenario: Typed connection degradation
- **WHEN** a report is error, required-peer disconnected, attention or stale while active
- **THEN** Third-Party Apps is marked with the specified severity
- **AND** an owning screen is marked only where it displays that issue or a recovery action

#### Scenario: Recovered or optional connection
- **WHEN** a connected report retains LastError, an optional peer is disconnected, or a report
  is unknown, off or ordinarily connecting
- **THEN** those conditions alone do not mark navigation

#### Scenario: Setup needs a user action
- **WHEN** an enabled Hue bridge or Discord client report is unconfigured
- **THEN** Lights or Meetings respectively receives an attention marker leading to its setup UI
- **AND** disabled plugins and absent optional media extensions remain unmarked

#### Scenario: Routing needs a choice
- **WHEN** a displayed priority entry has ambiguous matches or an output slot's provider state
  is exactly OutputNoOutput
- **THEN** Routing receives an attention marker
- **AND** unmatched optional devices, empty slots, disconnected slots with or without a held bus,
  and silence-based selection remain neutral

#### Scenario: Narrow window and accessible marker
- **WHEN** the GUI is shown at 800x600 with an attention marker
- **THEN** every navigation label remains readable, the marker has a concise accessible
  description and visible severity, and no header health badge or new focus target is added
