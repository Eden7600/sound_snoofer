# UI contract

This is the baseline for every build, not a redesign prompt. Change it only when a requested UX change requires it. Preserve everything outside that change.

## Goals and tone
Fast recognition, truthful state, predictable actions. Users know Snoofer. Use short nouns and state words, sentence case in source, no tutorial copy, enthusiasm, redundant qualifiers or implementation terminology. Keep consequential warnings (restart/recording interruption) and actionable error details.

## Content by surface
| Surface | Content |
| --- | --- |
| Deck key | Short label, recognizable icon, current state or local status. One or two words where possible; max 16 ASCII characters for built-ins. No explanatory sentences, action IDs or profile suffix on active-profile controls. |
| Deck dial | Target, gain, live meter; page dial shows previous/current/next page names on three lines, with the current name larger and cyan; neighboring names are smaller and neutral. Press still returns Home. No repeated gain label when dB is visible. |
| TUI | Explicit labels where no icon supplies context, profile sections, diagnostics and compact keyboard hints. Do not duplicate the current state in prose. |
| Tray | Open controls, lifecycle actions, concise state. No tutorials. |

Provider Label identifies an action without an icon. Optional ShortLabel is for compact icon-bearing surfaces. Shortening presentation must never change IDs, layout bindings, command semantics or saved settings. Fixed-profile bindings retain Normal/VR qualifiers. Custom plugin labels remain intact.

## Vocabulary
| Meaning | Deck label/state |
| --- | --- |
| Mic mute / speaker mute | Mute (different mic/speaker icons) |
| Monitoring | Monitor; Off / Pre / Post |
| Processing | Mic processing; Direct / Element |
| Stack enablement | Mic stack; On / Off |
| Input selection | Mic target; Auto / selected device |
| Recording capture | Record mic / Record PC; On / Off |
| Recording tap | Mic stage; Pre / Post |
| Recorder transport | Record / Stop rec; Ready / Rec |
| Media transport | Prev / Play / Next / Stop; no fictitious playback state |
| Soundboard | Filename / Stop; Ready / Playing (cyan), Wait during normalization; failures stay local to the clip |
| Gain | A1 / A2 / Mic; numeric dB and meter |
| Soundboard gain | Volume; numeric dB; press resets to 0 dB |
| Pending | Wait |
| Unavailable or unknown | N/A; meter LEVEL N/A |
| Failed | Error; detailed diagnostic in TUI |
| Overridden normal profile | VR; TUI VR override |

Status takes priority only on the affected control. Keep uncertainty distinct from Off or zero. Meter expiry must not show silence. Ready is not recording; enabled capture is not recorder transport.

## Color semantics
Colors supplement shape and words; they never carry state alone. Key backgrounds stay dark; no perimeter borders. Icon and state badge follow the same local semantic accent.
| Meaning | Deck color | Use |
| --- | --- | --- |
| Normal text | #E3EDF3 | Labels and neutral symbols |
| Neutral | #8797A3 | Off, Ready, generic commands |
| Active | #3AC6E1 | On/live, active processing mode, enabled monitor |
| Attention | #EEB74C | Wait, fallback, confirmation |
| Critical / audio inhibition | #FF6978 | Error, muted mic/output, recorder currently recording |
| Unavailable | #8797A3 | N/A; never danger merely because a device is absent |
| Background | #0E141B | Key and dial background |
| Meter signal | #2DD28C / #F5BE3C / #FF5A64 | -60..0 dBFS; amber from the -12 dB region, red near -3 dB; not application status |

Precedence: unavailable > failure > pending/fallback > muted/recording > active > neutral. Ordinary Off is neutral, including disabled mic stack. Recording's circle/square and Ready/Rec distinguish transport state. No arbitrary colors per control/category. Color definitions and precedence live in internal/streamdeck/palette.go.

TUI displays one existing control group at a time. Left/right or brackets cycle groups; up/down select controls. At 80 columns and above, show a section rail; narrower views show the section position. Editors use the full form width; forms are capped at 112 cells and shrink to content with a six-line minimum. Show row diagnostics once in the selected detail. Keep all groups reachable, preserve selection across snapshots, and exclude surface-only actions.

TUI retains blue focus/section styling, neutral text/details and subdued overridden rows. Amber is for consequential confirmations, not every footer. Honor NO_COLOR; preserve keyboard navigation, terminal bounds and diagnostics. Do not force pixel-exact deck colors into terminal themes.

## Visual and behavioral invariants
- Studio line icons, dark background, label above symbol, state badge below; no key outline.
- Signal path is left-to-right. Direct bypasses FX; Element passes through FX. Pre/Post taps branch before/after FX.
- Mic stack is power, mute is a mic with a slash when muted.
- Keep native rotation, blank unused keys, user-owned pages, dial pagination and meter placement.
- Do not change bindings, page positions, shortcuts, profile ownership, save/apply behavior or confirmations as a side effect of copy/style work.
- A UI change must update this contract when needed, add/update focused presentation checks, and inspect actual native-size renderer output. No fresh visual direction or synonyms on routine builds.


## Visual baseline
![Native-size key reference](design/streamdeck-controls.png)

Top: normal labels and signal-path states. Bottom: stack Off, muted, recorder Ready/Rec, Wait, Error, N/A, and fallback. The last example shows the amber fallback accent and * suffix. A supplied fallback never relies on color alone.


## Soundboard pages
Opt-in auto_controls prefixes fill unbound keys and add runtime overflow pages. Manual and shared bindings win. File names are labels; Play/Stop symbols identify transport. Matching PNG/JPEG/WebP/GIF artwork fits proportionally within a square and may replace the clip symbol in a 64px area, preserving its filename and state badge. Artwork colors are user content; badges retain semantic colors. GIF artwork uses its first frame as a static icon. Invalid/missing artwork falls back to Play. Stop remains at the configured position on every generated page. The configurator edits the saved template and previews automatic bindings; clearing the prefix disables automatic filling.

Soundboard key reference (Ready, Playing, Stop, and sample artwork in Ready/Playing):

![Soundboard keys](design/soundboard.png)

Page dial reference:

![Page dial](design/page-dial.png)

## Personal Home layout
Open controls occupies key 31 (zero-based; bottom row, fifth column). The rightmost four columns are empty across all four rows. This is a user-owned layout choice; do not relocate it during builds or overwrite other users' layouts.
