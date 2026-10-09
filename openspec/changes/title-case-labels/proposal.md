# Title-case labels
## Why
The user prefers title case for the app's own labels ("Record Mic", not "Record mic"). "Mic processing" also names the wrong thing: the control picks the mic's signal path (Direct or Element), so it becomes "Mic Path".

## What Changes
- **Rename:** "Mic processing" becomes "Mic Path".
- **Title case for built-in labels, on the deck and in the GUI:**
  - control names and short labels;
  - selection option labels and collection labels;
  - group headings;
  - Diagnostics report titles and detail names;
  - the GUI's screen titles, panel headings, buttons and field labels.
- **Rule:** capitalize each word except articles, short conjunctions and short prepositions (a, an, the, and, or, nor, but, for, of, on, in, to, at, by, as, per, via, vs) unless they come first. Acronyms and mixed-case names keep their case (VR, PC, ASIO, A1, dB, LocalVQE). Each part of a hyphenated word is capitalized ("Echo-Only", "Third-Party").
- **Unchanged:**
  - Sentences: notes, hints, placeholders, errors and tooltips.
  - Status and value words. Some, such as "VR override", are compared in code; title-casing them would be a separate change.
  - Content names: songs, apps, scenes, clips, rooms and devices.

## Impact
- **Code:**
  - Label literals in `plugins/*` and `app/web/app.js`.
  - Tests and `scripts/check-gui.cjs` that name those labels.
  - `docs/ui-contract.md`, and the label mention in `internal/streamdeck/AGENTS.md`.
- **Behaviour:** none. Group names used as identifiers (Recording, Routing, Bindings, the Normal prefix) are renamed consistently wherever they are compared.
