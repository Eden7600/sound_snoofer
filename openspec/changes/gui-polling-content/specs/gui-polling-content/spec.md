## ADDED Requirements

### Requirement: Polling preserves unchanged App Audio detail text
App Audio SHALL retain unchanged detail nodes and text during live polling so that
selection in one detail is not cleared by an unrelated value or relative-time update.
Changed values SHALL remain visible using the existing labels and placeholders.

#### Scenario: User selects a program while Heard advances
- **WHEN** the user selects Programs text in an open More disclosure and polling advances Heard
- **THEN** the Programs selection and unchanged text nodes remain intact
- **AND** Heard reflects the new relative time

#### Scenario: App details change
- **WHEN** current device, process or other detail values change
- **THEN** the affected detail text refreshes without replacing unrelated rows

### Requirement: Media retains its transport icon until playback state changes
Within an existing card, Media SHALL replace a play/pause icon only when its
displayed state changes while continuing to update metadata, accessible labels,
progress and action availability.

#### Scenario: Session receives an unchanged poll or nonstructural metadata update
- **WHEN** the playback state is unchanged across polling or an artist/album update that does not rebuild the card
- **THEN** the existing play/pause SVG remains attached
- **AND** current labels and playback progress remain live

#### Scenario: Playback changes between playing and paused
- **WHEN** a session changes its observed playback state
- **THEN** its transport icon and accessible action label reflect that state
- **AND** an explicit click still dispatches exactly one request through the existing path
