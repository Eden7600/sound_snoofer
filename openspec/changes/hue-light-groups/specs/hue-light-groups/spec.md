## ADDED Requirements
### Requirement: Staged light-group dials
The Hue plugin SHALL offer Decor, Mixed and Saturation dials for the selected room. Turning SHALL only stage a target; a press SHALL apply it with paced per-light writes. With nothing staged, a press SHALL toggle the Decor or Mixed lights and do nothing on Saturation.

#### Scenario: Stage and apply decorative brightness
- **WHEN** the Decor dial is turned to 40% and pressed
- **THEN** nothing is sent while turning, and on the press each decorative light in the room is set on at 40%

#### Scenario: Toggle without a stage
- **WHEN** the Mixed dial is pressed with nothing staged and a mixed light is lit
- **THEN** every mixed light in the room is turned off

#### Scenario: Stage expires
- **WHEN** a staged target is not applied within 10 s of the last turn
- **THEN** it is dropped, and the dial shows the observed state again

#### Scenario: Rate limit
- **WHEN** an apply covers many lights, or the bridge throttles a write
- **THEN** writes go out one at a time at least 100 ms apart, and a throttled write is retried after a longer gap

#### Scenario: Not confirmed
- **WHEN** some written lights do not report the requested state within 3 s
- **THEN** the dial reports how many were not applied, and nothing is retried

#### Scenario: No matching lights
- **WHEN** the selected room has no decorative lights
- **THEN** the Decor dial is blank

#### Scenario: Hue Sync streaming
- **WHEN** Hue Sync is streaming to the lights
- **THEN** the Decor, Mixed and Saturation dials are unavailable

### Requirement: Saturation keeps each light's hue
Applying a saturation SHALL move each colour-mode light's colour along its own line from white, keeping its hue, and SHALL leave white-only lamps, lights in white-temperature mode, and near-white colours unchanged.

#### Scenario: Desaturate
- **WHEN** Saturation is staged at 30% and applied in a room with red and blue lights
- **THEN** both lights move to 30% of the way from white to their gamut edge, staying red and blue

#### Scenario: Gradient strip
- **WHEN** a gradient strip shows five colours
- **THEN** all five points change saturation

#### Scenario: Only white lights
- **WHEN** every colour light in the room shows a white temperature
- **THEN** Saturation shows White and presses do nothing
