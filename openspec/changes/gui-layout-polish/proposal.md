# GUI layout and diagnostics polish

## Why
GUI audit items 10–14 identify cramped Routing editors, stretched Audio cards, Deck controls covered by the save row, crowded meter labels and an uninformative Diagnostics notice list. The current typed screens retain these defects; content corrections and state-preservation work do not solve them.

## What Changes
- Keep Audio cards at their content height, pair live switches with their labels and place Interface beside Engine above the gain strips.
- Give Routing section links, compact priority rows and a separate output-device editor below the route matrix. Show microphone-name Apply only for a changed name, retaining Enter/Escape and stale-list rules.
- Align Deck Device/Page selectors, keep Save/Discard in normal document flow beside the top editing controls, and state the scope of shared editing.
- Keep meter labels readable at narrow widths and report an empty Soundboard search explicitly.
- Group actionable Diagnostics notices by registered plugin name, collapse repeated Normal-profile VR override into one contextual notice, reuse clipboard handling, and keep the required Engine row plus restart commands without duplicating ordinary Audio settings.

## Impact
Frontend presentation and focused tests only. Reuse native controls, Tailwind, existing copy behavior and provider IDs/payloads. No dependencies, routing/persistence changes or new action protocol. Sidebar attention and disabled-plugin journeys remain a separate change; action outcomes and edit-state preservation have their own owners.
