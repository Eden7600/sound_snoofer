## ADDED Requirements
### Requirement: Dark and inert while locked or displays off
While the Windows session is locked or the monitors are off, the Stream Deck SHALL show nothing and SHALL ignore every key, dial and touch. When unlocked with monitors on, it SHALL show its current page again.

#### Scenario: Lock
- **WHEN** the user locks Windows
- **THEN** every key and the touch strip go black, the backlight turns off where supported, and presses dispatch nothing

#### Scenario: Monitors sleep while unlocked
- **WHEN** the monitors turn off while the session is unlocked
- **THEN** the deck is dark and inert until the monitors are on again

#### Scenario: Unlock
- **WHEN** the session is unlocked and the monitors are on
- **THEN** the deck redraws the page it showed, at the configured brightness

#### Scenario: Press while dark
- **WHEN** a key is held while dark and still held after unlock
- **THEN** no press or hold action fires for it

#### Scenario: Reconnect while dark
- **WHEN** the deck reconnects while the session is locked
- **THEN** it stays dark

#### Scenario: State unknown
- **WHEN** Windows does not report lock or display state
- **THEN** the deck stays lit and usable

#### Scenario: Brightness unsupported
- **WHEN** the brightness feature report fails
- **THEN** the deck still shows black and ignores input, and the failure is a diagnostic only
