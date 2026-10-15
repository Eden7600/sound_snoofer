# Output-slot presentation detail

## Why
Repository audit finding 33 found that healthy Music destination keys show Error. The audio provider puts the device name in Status, while the deck correctly treats an unrecognised nonempty status as an error. The existing renderer test bypasses provider-to-tile conversion.

## What Changes
- Publish the output-slot device name as ViewData.Device, beside its existing stable ID and bus, leaving Status for actual status/error semantics.
- Read the structured device detail on Routing without changing device choice, rename, source switch or stale-edit behavior.
- Test the actual generated provider control through bindingTile and the native-size renderer, preserving generic pending/unavailable/error behavior.

## Impact
Audio output publication, its generated fixtures and tests, Routing types/presentation and focused deck/GUI regressions. No persistence format, control ID, binding, routing or device-write changes. No new dependency or host interaction.
