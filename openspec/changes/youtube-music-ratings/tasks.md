# Tasks

Each numbered implementation block is a reviewable commit. Planning does not mark feature implementation complete.

## Design
- [x] 1. Review and commit proposal, requirements, design and task sequence with the stated focus default. Validate this change strictly.
- [ ] 2. With approved read-only browser access, verify current player identity, rating state and controls; record sanitized observations and refine the selector/identity contract before implementation.

## Implementation and automated checks
- [ ] 3. Extend the companion's bounded session/command contract and YouTube Music page support. Cover paused updates, rating toggles, same-title song changes, unknown/disabled state, stale routing, delayed artwork and no replay. Version helpers/events safely; explain the one-time tab reload and test upgrades plus existing transport after reload. Build and test both extension packages.
- [ ] 4. Extend Now Playing capture, validation, publication and rating pending/readback handling. Test explicit desired state, focused targeting, unsupported players, old extensions, song/focus/connection changes, shared rating reservation, independent transport, timeout and late observations.
- [ ] 5. Add the two hardware/editor icons and UI contract vocabulary; verify bindable controls, truthful state, media filtering and unchanged saved layouts. Inspect native-size renderer previews. Update installation/usage documentation.
- [ ] 6. Run Go tests, vet, race tests with the approved portable compiler, relevant GUI/type/lint/format checks, extension tests/builds and strict OpenSpec validation. Run the canonical application build and restore the prior launch configuration under existing permissions.

## Live acceptance
- [ ] 7. In an approved browser/desktop session, update the extension and bind Like/Dislike on the Plus and Plus XL without replacing user assignments. Verify inactive/active state, set/clear/switch, paused manual changes and truthful pending/failure display. Record and restore every tested track's initial rating, including tracks changed during task 8.
- [ ] 8. Verify browsing while music plays, song transitions, switching focused sessions, two YouTube Music tabs, page reload and bridge reconnect. Each first press works once, old actions never rate another track, and no rating is replayed. Record versions, results, cleanup and remaining website compatibility limits.
