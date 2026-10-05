# Soundboard peak normalization
## Why
Make clip peak levels consistent without modifying the source MP3s or changing the volume dial.
## What Changes
Normalize decoded samples to -1 dBFS in cached WAV copies before playback. Keep silent clips silent and leave dynamics unchanged. Preparation is cancellable and never blocks Stop or replacement.
## Impact
The existing Windows companion uses Media Foundation for offline decoding. No new runtime executable or third-party decoder is required.
