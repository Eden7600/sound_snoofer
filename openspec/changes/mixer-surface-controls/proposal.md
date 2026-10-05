## Why
Stream Deck requires gain and mute actions that share authoritative state with the TUI. Mic mute must use Voicemeeter's native mute without switching source or interrupting device assignments.
## What Changes
Introduce shared typed control actions and observations, native A1/A2/active-mic gain, independent microphone and playback mute, and multi-client acknowledgments.
## Impact
Worker ownership boundary, native parameter validation, control IPC, intent persistence and Controls/Graph status. No new diagnostic tabs. This foundation precedes direct-stream-deck.
