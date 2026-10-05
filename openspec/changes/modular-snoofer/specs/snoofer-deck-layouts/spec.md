## ADDED Requirements

### Requirement: User-owned pages and navigation
Stream Deck SHALL provide stable user-owned pages for the current 36-key, 6-encoder device. The last physical dial SHALL be reserved globally: rotate wraps pages, press selects Home and its display identifies the page.

#### Scenario: Page boundaries
- **WHEN** the navigation dial rotates beyond either end of the ordered pages
- **THEN** navigation wraps to the opposite end.

#### Scenario: Delete Home
- **WHEN** Home is deleted while other pages exist
- **THEN** the first remaining page becomes Home; deleting the final page is rejected.

### Requirement: Shared and local bindings
Each binding SHALL target one compatible semantic control. Shared bindings SHALL reserve positions on every page; local collisions and navigation-dial assignments SHALL be rejected.

#### Scenario: Position collision
- **WHEN** a page attempts to bind a shared position or the reserved navigation dial
- **THEN** the editor explains the conflict without silently removing either binding.

### Requirement: Integrated live configurator
The TUI SHALL support page creation, rename, reorder, deletion, Home selection, key/dial assignment and clearing, shared bindings and effective-layout preview. Save SHALL validate and atomically apply live; Cancel SHALL discard draft changes.

#### Scenario: Offline editing
- **WHEN** the deck is disconnected
- **THEN** configuration remains editable and the current saved layout renders on reconnect.

#### Scenario: Save failure
- **WHEN** layout persistence fails
- **THEN** the existing active layout remains and the draft/error stays available.

### Requirement: Unavailable bindings and input generations
Missing controls SHALL retain their saved IDs and labels, render dim Unavailable and be inert. Page/layout/provider changes SHALL invalidate stale input rather than reinterpret it.

#### Scenario: Provider disabled and restored
- **WHEN** a bound plugin is disabled and subsequently enabled
- **THEN** its bindings remain saved, are inert while unavailable and restore only when their original control IDs return.

#### Scenario: Held key across page change
- **WHEN** a page changes with pending or held input
- **THEN** that input cannot trigger the new page's binding.

#### Scenario: Hardware reconnect
- **WHEN** the deck reconnects with a key already held
- **THEN** held-key suppression prevents accidental activation and the current page is redrawn without replaying queued commands.

#### Scenario: Ambiguous device mapping
- **WHEN** serial-profile configuration cannot identify one applicable layout unambiguously
- **THEN** validation reports the ambiguity rather than arbitrarily choosing a mapping.

### Requirement: Preserve useful initial presentation
Initial configuration SHALL preserve existing useful key/gain bindings and Studio icons with live state text. User edits SHALL not be overwritten by regenerated defaults.

#### Scenario: Restart after customization
- **WHEN** the user saves a custom layout and restarts Snoofer
- **THEN** that layout remains authoritative.
