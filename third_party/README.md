# Third-party sources

Vendored, unmodified, and built only by `scripts/build-aec.ps1` into `bin/snoofer-aec.dll` for echo cancellation (see `openspec/changes/echo-cancellation`).

| Directory | Upstream | Version | Licence |
|---|---|---|---|
| `webrtc-audio-processing/` | https://gitlab.freedesktop.org/pulseaudio/webrtc-audio-processing | tag `v2.1`, commit `846fe90a289f58b7c9303a635142aa2c7caa93e5`; the `webrtc/` tree, `COPYING` and `README.md` only | BSD-3-Clause (`webrtc/LICENSE`, `webrtc/PATENTS`, `COPYING`) |
| `abseil-cpp/` | https://github.com/abseil/abseil-cpp | release `20240722.0`, archive SHA-256 `f50e5ac311a81382da7fa75b97310e4b9006474f9560ac46f54a9967f07d4ae3` | Apache-2.0 (`LICENSE`) |

## Abseil subset
Only `absl/{algorithm,base,functional,memory,meta,numeric,strings,types,utility}` are kept: what the webrtc sources include, plus those headers' own dependencies. Tests, benchmarks, test helpers and build files (`BUILD.bazel`, `CMakeLists.txt`) are removed. `scripts/build-aec.ps1` compiles the few `.cc` files the linker needs.

## Updating
1. Replace a directory with a newer upstream copy, pruned the same way.
2. Update the table above.
3. Update the source lists in `scripts/build-aec.ps1` from upstream's `meson.build` files.
4. Rebuild and rerun the offline echo cancellation probe.

## LocalVQE (optional neural AEC)

scripts/build-neural-aec.ps1 fetches LocalVQE f53063c9eb2a85f96479867d1dd911dc3bf6319b
and its GGML submodule c044a8eeae2591faa0950c8b5e514cbc4bbfc4ca to .local/neural-aec.
Source: https://github.com/localai-org/LocalVQE (Apache-2.0); GGML is MIT.
Upstream applies its tracked GRU patch. The build substitutes MSVC /fp:strict for
GCC -ffp-contract=off in the adaptive front end. No model algorithms are changed.
The script downloads the v1.4-AEC echo-only and v1.3 joint cleanup F32 GGUFs from
https://huggingface.co/LocalAI-io/LocalVQE, verifies pinned SHA256 hashes and retains
source licenses under bin/licenses/localvqe. Upstream additionally verifies its
model allowlist before parsing. Portable CMake 3.31.6 is hash-pinned in the cache;
MSVC/Windows SDK remain installed prerequisites. No model data goes to a server.
