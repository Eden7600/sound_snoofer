# Stream Deck graphics
- Expert UI: terse labels, state words, no instructional sentences. Let icons carry context; Mic mute becomes Mute, Active monitoring becomes Monitor, Active processing becomes Mic processing, and enablement becomes Mic stack.
- No perimeter borders on keys. Keep the existing dark Studio palette and clear state badge.
- Icons must communicate the operation and distinguish relevant states through shape, not color alone. Use the existing code-drawn primitives; preview the actual renderer at native size.
- Preserve pending, error, unknown, fallback and mute distinctions. Detailed diagnostics belong in controls, not on a 112px key.
- Compact labels come from provider ShortLabel; retain Label for contexts without icons. Do not infer actions or state from shortened display text.
- Preserve blank keys, user layouts, rotation, meter readings and changed-image caching.

Read ../../docs/ui-contract.md before edits. Reuse palette.go colors and keyAccent precedence; no inline semantic color variants. Routine builds preserve this baseline.
