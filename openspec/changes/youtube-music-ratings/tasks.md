# Tasks

Each numbered implementation block is a reviewable commit. Planning does not mark feature implementation complete.

## Design
- [x] 1. Review and commit proposal, requirements, design and task sequence with the stated focus default. Validate this change strictly.
- [x] 2. Complete the approved read-only identity/selector probe and record its sanitized [observations](observations.md), observed-layout contract and limits before implementation. Verified in Brave on 2026-10-15 without playback or account changes. Rating mutations, the complete unrated/disliked mappings, paused mutation updates, and other-browser acceptance remain unchecked work in tasks 3, 7 and 8.

## Implementation and automated checks
- [x] 3. Extend the companion's bounded session/command contract and YouTube Music page support. Cover paused updates, rating toggles, same-title song changes, unknown/disabled state, stale routing, delayed artwork and no replay. Version helpers/events safely; explain the one-time tab reload and test upgrades plus existing transport after reload. Build and test both extension packages.
- [x] 4. Extend Now Playing capture, validation, publication and rating pending/readback handling. Test explicit desired state, focused targeting, unsupported players, old extensions, song/focus/connection changes, shared rating reservation, independent transport, timeout and late observations.
- [x] 5. Add the two hardware/editor icons and UI contract vocabulary; verify bindable controls, truthful state, media filtering and unchanged saved layouts. Inspect native-size renderer previews. Update installation/usage documentation.
- [x] 6. Run Go tests, vet, race tests with the approved portable compiler, relevant GUI/type/lint/format checks, extension tests/builds and strict OpenSpec validation. Run the canonical application build and restore the prior launch configuration under existing permissions.

## Live acceptance
- [ ] 7. In an approved browser/desktop session, update the extension and bind Like/Dislike on the Plus and Plus XL without replacing user assignments. Verify inactive/active state, set/clear/switch, paused manual changes and truthful pending/failure display. Record and restore every tested track's initial rating, including tracks changed during task 8.
- [ ] 8. Verify browsing while music plays, song transitions, switching focused sessions, two YouTube Music tabs, page reload and bridge reconnect. Each first press works once, old actions never rate another track, and no rating is replayed. Record versions, results, cleanup and remaining website compatibility limits.

Automated implementation checkpoint: [validation.md](validation.md). Live tasks 7–8 remain unchecked.

## Requested personal placement — 2026-10-15

The user deferred desktop acceptance and requested Like at Home's top-right key,
with Dislike immediately below. On the Plus XL these are r1c9/r2c9 (zero-based
indices 8/17); both are currently empty in the inspected saved layout. Apply this
to the personal layout during the next approved live session, preserving all
other assignments and shipped defaults. This supersedes the earlier lack of a
placement request; it does not authorize unrelated layout migration.

On the smaller Plus, the corresponding r1c4/r2c4 slots currently hold Record and
Go to Meetings. Whether to replace these or limit the request to Plus XL is
awaiting the user's answer. No live layout has been changed.

Placement resolved: apply only to Plus XL. Smaller-deck layouts remain unchanged
for a later redesign; no Record or Meetings shortcut is replaced there.
