## ADDED Requirements

### Requirement: Audio controls and scales remain compact and readable
Audio cards SHALL retain their content height, related engine/interface information SHALL be adjacent, and live switch labels SHALL remain near their controls. Meter scale labels SHALL NOT overlap at either supported viewport size. Meter values, ballistics, color semantics and unknown-state handling SHALL remain unchanged.

#### Scenario: Narrow Audio and App Audio strips
- **WHEN** the window is 800×600 and meters update
- **THEN** visible scale labels do not collide
- **AND** exact readings, stale readings and peak indications retain their existing behavior

### Requirement: Routing edits retain identity in a navigable layout
Routing SHALL provide local section navigation and separate device management from source-switch table headers. Priority movement/removal SHALL remain visibly associated with the correct entry. Layout changes SHALL preserve explicit commitment, stable payload identity and stale-list rejection.

#### Scenario: Navigate and edit a slot
- **WHEN** the user follows Outputs and explicitly selects another slot device
- **THEN** navigation itself sends nothing and the edit retains that slot's stable ID and raw device identity
- **AND** a stale drawn configuration is rejected rather than retargeted

#### Scenario: Rename a microphone
- **WHEN** a microphone name is unchanged
- **THEN** its redundant Apply button is hidden
- **AND WHEN** the user changes it
- **THEN** Apply and Enter commit explicitly while Escape cancels and blur sends nothing

### Requirement: Deck editing and Soundboard filtering expose their scope
Deck save actions SHALL NOT obscure editing controls at supported sizes or scroll positions. Device/Page selectors SHALL align and shared editing SHALL identify its all-page scope. Soundboard SHALL distinguish an empty search result from an empty library.

#### Scenario: Edit a narrow Deck layout
- **WHEN** the window is 800×600 and the user scrolls through the Deck editor
- **THEN** the Save row does not overlap any key, dial, field or Page action
- **AND** Save/Discard, selection and shared/page draft semantics remain unchanged

#### Scenario: Search without matches
- **WHEN** a nonempty Soundboard library has no match for the current search
- **THEN** No matching clips is visible and searching sends no action
- **AND** clearing the search restores the matching clips and removes that message

### Requirement: Diagnostics notices identify actionable context without losing errors
Diagnostics SHALL group displayed notices by registered plugin name, consolidate exact repeated Normal-profile VR override notices and omit known descriptive statuses that remain on their owning screens. Unknown statuses and actual failures SHALL remain visible. Startup and Engine health SHALL remain available alongside existing restart commands.

#### Scenario: Descriptive and failing controls coexist
- **WHEN** an AEC timing description, startup note, Normal VR override and provider failure are published
- **THEN** Current Notices excludes duplicate descriptive/startup entries, shows one contextual Normal-profile override and retains the full failure under its plugin
- **AND** the owning screens retain the omitted details

#### Scenario: Copy the displayed notices
- **WHEN** the user chooses Copy Notices
- **THEN** the displayed plugin groups, labels and full statuses are copied with local confirmation
- **AND** polling does not steal focus or erase the confirmation, clipboard fallback remains available and no provider action is sent

### Requirement: Native compact sidebar branding stays visible
At the supported minimum outer window size, the sidebar SHALL fit the existing mascot and complete wordmark inside its usable client width even when a classic scrollbar reserves space. The sidebar SHALL retain vertical scrolling, full navigation access and its existing width, without horizontal overflow or wordmark truncation.

#### Scenario: Classic scrollbar at the native compact client size
- **WHEN** the 800×600 outer native window yields an approximately 784×562 content area and the sidebar needs a vertical scrollbar
- **THEN** the mascot and every letter of the wordmark remain inside the usable sidebar content bounds
- **AND** the last navigation item remains reachable and visible through sidebar scrolling or keyboard navigation
- **AND** navigation sends no provider action and no horizontal sidebar scrolling is needed

### Requirement: Shared notices align their dismissal control
The shared error notice SHALL vertically center Dismiss within its padded bounds and keep the complete message readable beside it at supported window sizes. Long identifiers and explicit line breaks SHALL wrap without overlap or horizontal overflow. Alert semantics, local dismissal and focus during unchanged polls SHALL remain unchanged.

#### Scenario: Short and multiline shared errors
- **WHEN** a short error or a long multiline error is shown at a supported window size
- **THEN** Dismiss is vertically centered and contained inside the notice padding while the message remains readable
- **AND** unchanged polling preserves Dismiss focus and activating it hides the notice without dispatching an action
