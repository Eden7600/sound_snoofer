## ADDED Requirements
### Requirement: Optional independent clip playback
The system SHALL provide an optional soundboard plugin with an audio dependency, local MP3 discovery, explicit clip and Stop controls, and one active clip.
#### Scenario: Replace playback
- **WHEN** a second clip is pressed
- **THEN** the first clip stops before the second starts
#### Scenario: Disabled or preview
- **WHEN** the plugin is disabled or preview is active
- **THEN** no native playback runs
#### Scenario: Playback failure or shutdown
- **WHEN** decoding fails, routing becomes unavailable, playback completes, or the plugin stops
- **THEN** playback resources are released without retrying or operating recorder transport

### Requirement: Observed dedicated routing
The audio plugin SHALL reserve VAIO3 explicitly and reconcile its soundboard sends through the existing serialized routing controller.
#### Scenario: Clip destinations
- **WHEN** microphone and monitor destinations are enabled
- **THEN** clips route to B3 and the selected playback bus, bypassing Element
#### Scenario: Plugin disabled
- **WHEN** soundboard is disabled but the input remains reserved
- **THEN** its A and B2/B3 sends are cleared by audio
#### Scenario: Routing not verified
- **WHEN** audio observations are stale, disconnected, or disagree with required routing
- **THEN** new playback is rejected

### Requirement: Soundboard volume dial
The soundboard SHALL expose VAIO3 playback gain as an adjustable semantic control through the existing audio worker.
#### Scenario: Turn dial
- **WHEN** the Soundboard volume dial turns
- **THEN** only Strip[7].Gain changes, clamped to -60 through +12 dB, with observed readback
#### Scenario: Press dial
- **WHEN** the volume dial is pressed
- **THEN** its gain resets to exactly 0 dB without changing any mute parameter
#### Scenario: Unavailable audio
- **WHEN** soundboard routing is disabled, audio is unavailable or the request is stale
- **THEN** gain writes are rejected

### Requirement: Matching clip artwork
The system SHALL accept matching PNG, JPEG, WebP and GIF image content as clip icons, preserving filename and playback state, with proportional fitting into a square thumbnail.
#### Scenario: Matching artwork
- **WHEN** a matching basename image exists
- **THEN** the clip displays its thumbnail, preferring PNG, JPG, JPEG, WebP, then GIF filename extensions case-insensitively
#### Scenario: Invalid or absent artwork
- **WHEN** artwork is absent, oversized or invalid
- **THEN** the play icon remains and the clip can still play
#### Scenario: Downloaded image compatibility
- **WHEN** a rectangular image or WebP content saved with a PNG extension is supplied
- **THEN** the content is decoded and fitted without stretching or modifying the source file
#### Scenario: GIF icon
- **WHEN** matching GIF artwork exists
- **THEN** its first frame is used as a static icon, preserving transparency and logical canvas positioning
#### Scenario: Image refresh
- **WHEN** artwork changes or is removed
- **THEN** the next catalogue scan refreshes the displayed icon

### Requirement: User-owned soundboard page
Stream Deck SHALL support opt-in prefix-based automatic page bindings without changing manual or shared bindings.
#### Scenario: Folder contents change
- **WHEN** MP3 files are added or removed
- **THEN** stable semantic controls and automatic page contents refresh
#### Scenario: Page overflow
- **WHEN** matching controls exceed available keys
- **THEN** additional runtime pages remain reachable through existing pagination
#### Scenario: Existing layout
- **WHEN** soundboard is configured
- **THEN** Home and all unrelated saved bindings remain unchanged
