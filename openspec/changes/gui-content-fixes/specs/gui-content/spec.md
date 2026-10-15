## ADDED Requirements

### Requirement: Audio choices use labels appropriate to their control
Audio selections SHALL display their documented option labels and SHALL preserve raw option
values, configured names and saved choices. Microphone labels and silent annotations SHALL NOT
be applied to processing, monitor, recording-tap or playback selections.

#### Scenario: Processing and monitor choices
- **WHEN** active, Normal or VR processing/monitor controls are published
- **THEN** Direct/Element and Off/Pre/Post labels are available to both GUI and deck renderers
- **AND** the underlying values remain direct/element and off/pre/post

#### Scenario: Microphone and playback names
- **WHEN** a microphone is silent and a playback device is explicitly selected
- **THEN** only that microphone receives its Silent annotation
- **AND** the playback name remains unchanged and its empty choice displays Automatic

### Requirement: Routing Activity reflects provider settings
The Activity card SHALL use the provider's lowercase JSON fields and SHALL submit edits using
the same field names without writing settings while rendering or focusing a control.

#### Scenario: Generated Activity settings
- **WHEN** the provider publishes check 2, silence_db -70 and silent_after_s 10
- **THEN** the card displays Top 2, -70 dBFS and 10 seconds
- **AND** changing any field sends exactly one edit for that field

#### Scenario: Metering disabled
- **WHEN** Activity is absent or Meter is changed to Off
- **THEN** absence displays Off with empty threshold fields
- **AND** choosing Off sends check with value 0

### Requirement: Audio cards show each action in its intended location
Playback source sends SHALL remain available through Routing and existing deck bindings, without
duplicating them in Audio's recovery card. Echo Cancellation SHALL show a labelled Retry action
without an empty field caption and SHALL retain its existing availability and press semantics.

#### Scenario: Playback route remains usable
- **WHEN** a playback source send is published
- **THEN** its label uses the existing source vocabulary and its group is Routing
- **AND** the Routing matrix can still dispatch the same control ID and operation
- **AND** Audio's recovery card does not repeat it

#### Scenario: Retry state and action
- **WHEN** AEC retry becomes available
- **THEN** the Retry button is enabled and pressing it sends aec.retry with operation press
- **AND** an unavailable or pending retry remains disabled

### Requirement: Names preserve plugin and binding identity
The host SHALL publish registered plugin labels. GUI plugin headings, the Not monitored list and
binding context SHALL use those labels with ID fallback, without changing action IDs or saved
bindings. Fixed-profile binding labels SHALL retain Normal/VR qualifiers.

#### Scenario: Friendly names retain raw action IDs
- **WHEN** a registered plugin named App Audio is displayed or selected
- **THEN** the GUI shows App Audio
- **AND** its selection action still targets appaudio

#### Scenario: Distinct fixed-profile choices
- **WHEN** Normal and VR audio controls both appear in the binding selector
- **THEN** their visible labels distinguish the profiles and use readable plugin context
- **AND** selecting either sends its original control ID

#### Scenario: Missing metadata and search
- **WHEN** a plugin label is missing or the user searches by plugin ID
- **THEN** the ID remains available as the display fallback or search match
- **AND** custom provider names are preserved as written

#### Scenario: Labels arrive in a later snapshot
- **WHEN** registered plugin labels change or arrive after an ID fallback was shown
- **THEN** plugin headings and binding options display the current labels
- **AND** their action targets remain unchanged

### Requirement: Existing page names remain intact
The Stream Deck page selector SHALL continue to use the provider's OptionLabels for built-in and
custom page names while retaining page IDs as values.

#### Scenario: Named page selection
- **WHEN** a page with ID home has label Home or a user-defined page has a custom name
- **THEN** the selector displays the supplied name
- **AND** selection submits the corresponding unchanged page ID
