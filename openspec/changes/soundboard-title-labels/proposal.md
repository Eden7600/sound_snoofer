# Soundboard title labels
## Why
Clip labels are raw file stems such as `air_horn` or `sad-trombone`. Every other built-in label is now title case (`title-case-labels`).

## What Changes
A clip's label is its file stem in title case:
- Underscores and hyphens become spaces, and runs of spaces collapse.
- Words are capitalized, with minor words (a, an, the, and, or, nor, but, for, of, on, in, to, at, by, as, per, via, vs) lower case unless first.
- All-caps words and mixed-case words keep their case: `OMG`, `iPhone`, `DJ`.

For example, `air_horn` reads "Air Horn", `rick_and_morty` reads "Rick and Morty", and `OMG-wow` reads "OMG Wow".

The clip's ID, artwork matching (by file stem), normalization and playback are unchanged.

## Impact
- **Code:** `plugins/soundboard` (label), `docs/ui-contract.md`.
- **Unchanged:** IDs, so saved deck bindings keep working.
