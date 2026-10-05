## Supersession

Historical requirements below are superseded by element-process-fallback: Post remains saved in Direct; effective Pre is temporary and returning to Element restores Post. Do not implement the historical normalization scenarios.

## ADDED Requirements

### Requirement: Direct recording uses Pre
The system SHALL normalize Post mic recording to Pre when processing mode is Direct.

#### Scenario: Switch from Element to Direct
- **WHEN** mode changes to Direct while mic recording stage is Post
- **THEN** mode and Pre stage are saved together before routing is applied
- **AND** recording inclusion flags and recorder transport are unchanged

#### Scenario: Load legacy choices
- **WHEN** saved or programmatic intent contains Direct and Post
- **THEN** effective intent uses Pre and the next explicit save persists Pre

#### Scenario: Return to Element
- **WHEN** mode changes back to Element
- **THEN** stage remains Pre until explicitly changed to Post

#### Scenario: Select stage in Direct
- **WHEN** Recording Mic Stage is activated in Direct mode
- **THEN** Pre remains selected and no Post command is queued

#### Scenario: Save fails
- **WHEN** saving a mode change fails
- **THEN** the prior effective mode and stage remain active and no dependent routing is applied

#### Scenario: Invalid stage spelling
- **WHEN** a stage is neither Pre nor Post
- **THEN** validation rejects it instead of silently correcting it
