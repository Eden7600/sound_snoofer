## ADDED Requirements

### Requirement: Audio ownership
The audio plugin SHALL retain serialized native ownership, deterministic routing, save-before-apply, verification, recording safety and recovery limits. VR SHALL depend on audio and own SteamVR observation.

#### Scenario: Audio without VR
- **WHEN** audio is enabled and VR is disabled
- **THEN** Normal audio works without SteamVR process observation or VR policy activation.

#### Scenario: Native recovery state
- **WHEN** audio reconnects or the application is renamed/restarted
- **THEN** uncertain-command markers and recovery budgets still constrain actions and recorder Start is not replayed.

### Requirement: Separate effective profiles
Known running SteamVR SHALL activate VR microphone and playback preferences over Normal. Known stopped SteamVR SHALL restore Normal preferences. Each profile SHALL have independent priority lists, saved overrides, processing mode and monitoring; mute, gain and recording preferences SHALL be shared.

#### Scenario: VR overrides Normal
- **WHEN** SteamVR starts while Normal has a manual source override
- **THEN** VR preferences take effect without overwriting Normal's saved override or clearing shared mute.

#### Scenario: SteamVR stops
- **WHEN** SteamVR stops after Normal preferences were edited during VR
- **THEN** the edited Normal choices become effective subject to current device availability.

#### Scenario: Detection failure
- **WHEN** SteamVR observation becomes unknown
- **THEN** the last known profile remains effective with a stale indication; before any known observation startup remains Normal with unknown VR status.

### Requirement: Explicit fallback and safe device matching
Microphone and playback SHALL resolve independently with actual presence, unique matching, debounce and verification. An optional final VR entry MAY fall back to Normal for that direction; no implicit cross-profile fallback SHALL occur.

#### Scenario: VR device disconnect
- **WHEN** the selected VR endpoint disappears
- **THEN** resolution follows that profile's saved override and fallback rules, uses Normal only if explicitly configured, and preserves saved preferences.

#### Scenario: Reconnect
- **WHEN** a preferred endpoint returns uniquely
- **THEN** the effective profile re-evaluates and verifies the required assignment without altering unrelated mute, gain or recording state.

#### Scenario: Ambiguous endpoint
- **WHEN** a candidate matches multiple indistinguishable connected endpoints
- **THEN** it is unavailable with an ambiguity reason and no arbitrary endpoint is selected.

#### Scenario: Normal fallback
- **WHEN** VR's final fallback entry resolves a Normal source
- **THEN** only that direction's source resolution delegates to Normal and VR processing/monitoring preferences remain applicable.

### Requirement: Explicit profile presentation
The audio view SHALL display separate Normal and VR microphone/playback sections. Overridden Normal sections SHALL be subdued, editable and accompanied by per-section override explanation; there SHALL be no combined selector.

#### Scenario: Editing an inactive profile
- **WHEN** a user edits Normal while VR is effective
- **THEN** the choice saves without immediately writing that inactive preference to the mixer.

#### Scenario: Effective microphone Off
- **WHEN** the effective profile selects microphone Off
- **THEN** managed microphone assignments and sends are disconnected under existing invariants without stopping playback or recorder transport.

#### Scenario: Disabling VR
- **WHEN** VR is disabled through application restart
- **THEN** shutdown does not change audio routes and the next audio startup reconciles Normal.
