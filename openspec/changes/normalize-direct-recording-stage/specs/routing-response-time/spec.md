## ADDED Requirements

### Requirement: Routing-only edits use a short debounce
The controller SHALL use a 100 ms debounce for plans without changed device assignments and retain the configured debounce for device assignment changes.

#### Scenario: Routing-only edit
- **WHEN** a live plan changes only sends or ASIO input patches
- **THEN** the controller schedules its next observation at the remaining 100 ms debounce deadline even if normal polling is slower

#### Scenario: Device change or mixed plan
- **WHEN** a plan assigns or clears any device
- **THEN** the configured device debounce applies to the complete plan

#### Scenario: Preview, idle or failure
- **WHEN** preview mode, converged routing or an application failure occurs
- **THEN** preview remains read-only, idle polling remains configured and failure backoff remains intact

#### Scenario: Inventory or desired state changes
- **WHEN** the pending plan changes before application
- **THEN** its stability deadline restarts and existing inventory/drift verification remains enforced
