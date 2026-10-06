# GUI on Tailwind and Lucide
## Why
The desktop GUI uses Unicode glyphs (≋ ▷ ☼ ▦ ◇ ⇄ ⌁ ◍ ♩ ◌) as icons, which render inconsistently and look poor, and a hand-written stylesheet of custom classes that is hard to extend.
## What Changes
- Replace every GUI glyph icon with Lucide icons (ISC). The SVGs are vendored into the repository; nothing is fetched at runtime. Stream Deck key icons stay code-drawn.
- Migrate the GUI to Tailwind CSS v4 utilities, compiled at build time by the Tailwind CLI. Colors come from `@theme` tokens; hand-written component classes are avoided.
- Keep the current look and layout without requiring pixel parity. Element hooks for automated checks move from styling classes to `data-*` attributes.
## Impact
Builds now need Node and the npm dev dependencies (`@tailwindcss/cli`, `tailwindcss`, `lucide-static`). The compiled CSS is generated and not committed. `scripts/build.ps1` and the GUI check scripts generate it. No Go behavior, control or Stream Deck rendering changes.
