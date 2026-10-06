# ASIO-first playback

## Why
Hardware bus names leak into everyday controls. Interface selection must determine which ASIO inputs and outputs are eligible, independently of microphone and playback preferences.

## What Changes
- Replace the single ASIO matcher and special playback fallback with an ordered interface list and ordinary playback candidates.
- Only the selected interface supplies ASIO inputs/outputs; lower interfaces remain unavailable even when connected.
- Publish one Playback gain/meter and persisted authoritative playback mute. Do not set, copy, remember or reset gains during routing.
- Manually convert this installation's config and deck bindings; no automatic migration layer.
