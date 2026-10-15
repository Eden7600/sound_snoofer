## ADDED Requirements

### Requirement: Connection recovery is distinct from action feedback
The GUI SHALL show a global, nonblocking Reconnecting banner below the shared header while host state polling fails. It SHALL retain the last received screen and existing action errors. The banner SHALL clear only after a successful state response, including an unchanged response.

#### Scenario: Host disconnects while an action error is visible
- **WHEN** state polling fails after an action error was displayed
- **THEN** the global connection banner appears without replacing that error or clearing the screen
- **AND** navigation remains available and shows the same global recovery state
- **AND** successful recovery clears only the connection banner

### Requirement: Failed polling backs off and recovers
The GUI SHALL increase the retry delay after consecutive transport failures, bounded at five seconds, and restore the existing screen polling cadence after success. Render failures SHALL remain distinct from transport failures and SHALL NOT prevent another poll.

#### Scenario: Connection remains unavailable
- **WHEN** consecutive state requests fail
- **THEN** retry delays grow from 200 ms to a maximum of 5000 ms
- **AND** the GUI continues attempting recovery without duplicating error notices

#### Scenario: A recovered snapshot cannot render
- **WHEN** the state request succeeds but rendering throws
- **THEN** the connection banner is cleared and the render error is reported
- **AND** the next state request is still scheduled at the normal cadence

### Requirement: Recovery never replays actions
The GUI SHALL explicitly refuse new actions while disconnected. It SHALL NOT queue refused actions or automatically retry failed or uncertain one-shot commands when the connection returns. Recovery SHALL NOT itself be treated as an action result.

#### Scenario: Recording command fails before connection loss
- **WHEN** sending a one-shot recording command fails and subsequent state polling disconnects then recovers
- **THEN** the original failure remains visible
- **AND** that command is sent only once

#### Scenario: User acts during recovery
- **WHEN** a user requests an action while the recovery banner is visible
- **THEN** the GUI reports that the action was not sent
- **AND** no Send call occurs during disconnection or automatically after recovery
