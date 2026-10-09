# Design
`clipLabel(stem)` in `plugins/soundboard` builds the display label:
1. Splits the stem on underscores, hyphens and spaces, dropping empty parts.
2. Keeps a word unchanged when any letter after the first is upper case, or when the whole word is upper case.
3. Otherwise lower-cases minor words (except the first word) and capitalizes the first letter of the rest, leaving the rest of the word as written.

The catalog keeps the file stem for artwork matching and stores `clipLabel(stem)` as `Label`. Clips sort by label, as today.
