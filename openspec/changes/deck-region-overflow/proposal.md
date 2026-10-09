# Deck region overflow editing
## Why
- **Clipping is not editable.** `Region.Clip` keeps a region to the keys it covers, so it never adds overflow sets. The default Home strips are clipped, but the editor cannot show or set clipping, and regions it adds are never clipped. After the user redrew the Home strips in the editor, Home grew to three sets (Home, Home 2, Home 3). Home has no Up/Down keys, so the page dial visited each one.
- **Audit of the Stream Deck screen** found more gaps:
  - **Clipped stacked regions still overflow.** A region whose rows are shared by several sources ignores Clip.
  - **Overflow is invisible.** The editor previews only the first set. Nothing says that a page has several sets, or that the page dial visits each when the page has no Up/Down keys.
  - **Shared-row sources are hidden.** A region's source picker shows only its first source, so the other sources sharing its rows are invisible.
  - **Make Home on Home** marks the draft unsaved without changing anything.

## What Changes
- **Overflow per region:** each region row in the editor has an Overflow toggle. Off saves `clip: true`. New regions keep today's default: Overflow on.
- **Clipped stacks:** a clipped region whose rows are shared by several sources shows only its first set.
- **Sets note:** the Regions panel shows how many sets the edited page has, and how they are reached: Up/Down, or the page dial when the page has no scroll key.
- **Shared-row sources:** a region row names the other sources that share its rows.
- **Make Home:** disabled on the Home page; it changes nothing there.

## Impact
- **Code:** `plugins/streamdeck` (edit operation, expansion, editor view, Make Home), `app/web/app.js`, `scripts/check-gui.cjs`, `docs/ui-contract.md`.
- **Data:** no format change. `clip` already exists on saved regions.
- **Personal layout:** the user's Home strips are set back to clipped, as requested.
- **Unchanged:** audio and every other plugin.
