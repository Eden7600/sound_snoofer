## ADDED Requirements

### Requirement: Focused YouTube Music ratings
Snoofer SHALL provide bindable Like and Dislike controls for its focused YouTube Music browser media session and SHALL show the observed rating. Unsupported sessions SHALL remain ineligible for rating actions.

#### Scenario: Like and clear
- **WHEN** the user presses Like on an unrated eligible song, then presses Like again after the liked state is observed
- **THEN** the first action requests Like and the second requests no rating
- **AND** the key changes state only from the corresponding observed page state

#### Scenario: Switch rating
- **WHEN** the user presses Dislike on a liked song
- **THEN** the captured desired rating is Dislike and the two keys reflect the single mutually exclusive observed rating

#### Scenario: Unsupported focused session
- **WHEN** another media player is focused while YouTube Music is also open
- **THEN** the rating controls are hidden and inert rather than targeting an unrelated tab

#### Scenario: Several YouTube Music sessions
- **WHEN** two eligible YouTube Music tabs exist
- **THEN** only the session selected by Snoofer's existing media focus rules receives a rating action

### Requirement: Truthful rating state
The extension SHALL distinguish unrated, liked, disliked and unknown states from the actual playing item, including while playback is paused. Capability SHALL require the verified top-frame YouTube Music document and actionable player controls.

#### Scenario: Manual paused change
- **WHEN** the user changes the rating in YouTube Music while the song is paused
- **THEN** the keys update without requiring playback to resume

#### Scenario: Artwork completes late
- **WHEN** an older artwork request completes after a newer song or rating report
- **THEN** the older snapshot cannot overwrite the current rating or revive the old track
- **AND** current control state does not wait for artwork to finish

#### Scenario: Unknown player state
- **WHEN** the player is loading, an advertisement is active, controls require account interaction, or the current song/rating cannot be established
- **THEN** rating actions are unavailable and unknown is not displayed as an unrated song

#### Scenario: Browsing away from the playing item
- **WHEN** the user browses another album or song without changing playback
- **THEN** the rating controls continue to describe the playing item rather than the browsing URL

#### Scenario: Untrusted capability
- **WHEN** an embedded frame or another origin reports a YouTube Music rating capability, or rating fields exceed bounds or use invalid values
- **THEN** the capability or invalid report is rejected before it can produce an actionable rating control

### Requirement: Captured rating intent
Each rating action SHALL capture an explicit desired rating and the selected session, connection, document and playing-track lifetime. The page SHALL validate the current playing item immediately before acting. Stale or ambiguous work SHALL be rejected without clicking.

#### Scenario: Song changes before execution
- **WHEN** an action captured for one song reaches the page after another song starts, even with identical title and artist
- **THEN** the page does not click either rating button for the replacement song

#### Scenario: Focus or document changes
- **WHEN** focus changes or the tab reloads before captured work is dispatched
- **THEN** the work is rejected rather than redirected to the replacement target

#### Scenario: Desired rating is already present
- **WHEN** the captured desired rating is already observed immediately before execution
- **THEN** the page leaves that rating unchanged rather than toggling it off

### Requirement: Bounded independent confirmation
Like and Dislike SHALL share one pending rating operation per target, separate from playback operations. Submitted transport results SHALL NOT imply an observed rating change. Unconfirmed or failed actions SHALL never be automatically retried.

#### Scenario: Rapid conflicting ratings
- **WHEN** a rating action is still unresolved and the other rating key is pressed
- **THEN** no second rating action is dispatched until the first is resolved or expires
- **AND** independent playback controls remain usable

#### Scenario: Missing readback
- **WHEN** the desired rating is not observed within the bounded confirmation period
- **THEN** local feedback reports the lack of confirmation and the keys retain actual observed state
- **AND** no automatic retry occurs

#### Scenario: Disconnect and reconnect
- **WHEN** the browser bridge disconnects with a rating pending and later reconnects
- **THEN** old pending work is retired, no rating is replayed, and only fresh state enables new actions

### Requirement: Compatible installation and presentation
The controls SHALL use existing Deck state conventions and editor binding, preserve saved layouts, and remain compatible with older companions that omit rating fields. Updating helpers in an already-open page SHALL NOT duplicate existing media actions.

#### Scenario: Older companion
- **WHEN** a protocol-compatible companion reports media without rating fields
- **THEN** existing transport works and rating controls remain unavailable

#### Scenario: Helper upgrade
- **WHEN** updated helpers are injected into a page already running the previous helper
- **THEN** a single transport press invokes its action at most once and unverified capabilities remain unavailable
- **AND** installation feedback explains the required one-time tab reload

#### Scenario: Reload after upgrade
- **WHEN** the user reloads the affected tab after updating the extension
- **THEN** the new rating capability and the page's newly registered existing transport handlers work without duplicate execution

#### Scenario: Saved layouts and filtering
- **WHEN** the feature is installed or Media on the Deck is turned Off
- **THEN** existing assignments are preserved and the filter hides the new controls with the other media controls
