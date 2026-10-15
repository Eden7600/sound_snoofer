## ADDED Requirements

### Requirement: Browsing a live dropdown does not write settings
Native dropdowns that directly set provider configuration SHALL stage keyboard browsing locally.
Arrow keys, Home/End, PageUp/PageDown and typeahead SHALL NOT dispatch an action before explicit
commitment. Enter SHALL commit the selected choice once. Escape or leaving the field after
closed-select browsing without commitment SHALL cancel the local choice without dispatch.

#### Scenario: Browse Processing and Monitor
- **WHEN** the user moves through a closed Processing or Monitor dropdown with the keyboard
- **THEN** no semantic action is sent until Enter confirms the choice
- **AND** the confirmed choice is sent exactly once

#### Scenario: Cancel a keyboard choice
- **WHEN** the user presses Escape or tabs away after browsing a different choice with the popup closed
- **THEN** the observed provider choice is restored and no action is sent

### Requirement: Native popup selection retains explicit commitment
A pointer choice or explicit confirmation in a native popup SHALL commit once. Cancelling the
popup with Escape, Alt+Up or an outside click SHALL NOT dispatch. Tab inside an open native popup
SHALL retain native confirmation semantics and commit once. Existing forms with Add/Apply and Stream Deck draft/navigation controls
SHALL retain their existing separate commitment semantics.

#### Scenario: Pointer or keyboard popup choice
- **WHEN** the user confirms an option in a native popup opened by pointer or keyboard
- **THEN** the GUI sends one request for that option
- **AND** dismissing the popup without confirming sends none

#### Scenario: Tab confirms inside the native popup
- **WHEN** the user opens the native popup, highlights a different option and presses Tab
- **THEN** that explicit native confirmation sends one request
- **AND** no delay or key-release timing heuristic is used to distinguish it from pointer confirmation

#### Scenario: Popup dismissal leaves subsequent browsing safe
- **WHEN** the user dismisses the native popup with Escape, Alt+Up or an outside click
- **THEN** the observed choice is restored without a request
- **AND** subsequent closed-select browsing still waits for Enter

### Requirement: Committed choices preserve action identity
Dropdown commitment SHALL retain focus-time revisions and existing stale-list checks. A later
snapshot SHALL NOT silently retarget a staged choice or substitute a fresh revision.

#### Scenario: Provider changes during browsing
- **WHEN** the provider publishes a new revision after the user starts browsing
- **THEN** commitment retains the captured revision or existing stale-list sentinel
- **AND** provider rejection remains visible through the existing error path
