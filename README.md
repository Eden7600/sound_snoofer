<p align="center">
  <img src="docs/assets/sound-snoofer-painterly.png" alt="Sound Snoofer: golden protogen mascot with headphones and a yellow digital visor" width="360">
</p>

# Snoofer

A Windows tray application with optional compiled Audio, VR, Stream Deck, Soundboard and Windows media plugins. Core owns the tray, desktop controls, plugin lifecycle and semantic controls. Audio retains Sound Snoofer's Voicemeeter routing, recording and recovery behavior.

## Run

Launch `bin/snoofer.exe`; choose **Open controls** from its tray icon. Closing controls leaves Snoofer running. **Quit Snoofer** releases resources without resetting mixer routing or stopping recording.

Configuration defaults to `snoofer.json` beside the executable. Plugin dependency changes are confirmed together and restart the application. A new configuration starts with every plugin disabled. Personal settings were manually converted in `bin/snoofer.json`, with operational journals at their original paths.

The controls window has Audio, Soundboard, Stream Deck, Plugins and Diagnostics screens. Click or use Tab to focus controls; sidebar Up/Down follows its vertical order. Text fields apply with Enter and cancel with Escape. Normal audio remains editable under VR overrides. The deck editor shows physical positions with a binding inspector and explicit Save/Discard.

## Build and validate

The GUI uses the installed Microsoft WebView2 runtime. HTML/CSS/JavaScript assets are embedded in snoofer.exe; no frontend bundler or separate server is required. Use the Go version in go.mod. Default audio builds require installed MSVC x64 tools and the Windows SDK for the callback companion; core-only builds do not. Do not redistribute Voicemeeter's vendor DLL.

```powershell
.\scripts\build.ps1
.\scripts\build.ps1 -Tags core
.\scripts\build.ps1 -Tags no_audio
.\scripts\check.ps1
.\scripts\check.ps1 -GUI
.\bin\snoofer.exe --check --config .\bin\snoofer.json
```

Optional `-GUI` checks use the development Playwright dependency and installed Chrome, plus Windows UI Automation for the packaged WebView2 window. Run `npm ci` once for development dependencies. No browser download is performed by these scripts.

Keep `snoofer-audio-monitor.dll` beside audio-enabled executables and `snoofer-soundboard.dll` beside soundboard-enabled executables. Build scripts use the Windows GUI subsystem to avoid a blank terminal. They do not stop an in-use application.

For interactive development preview, copy the envelope and audio sidecars into a repository-local scratch directory, update state_path, and use `--dry-run --config <copied-envelope>`. Preview avoids audio writes but can save UI settings; do not interactively smoke-test against personal state.

## Guides

- [Application and configuration](docs/application-guide.md)
- [Plugin authoring and composition](docs/plugins.md)
- [Troubleshooting](docs/troubleshooting.md)
- [Recording setup](docs/recording.md)
- [Implementation and validation](openspec/changes/modular-snoofer/implementation.md)
- [Remaining hardware acceptance](openspec/changes/modular-snoofer/tasks.md)

Audio maintenance commands use `snoofer --config <envelope> audio <command>`. These commands run audio alone, without VR process policy; use the tray for the complete application.

Callback progress proves processing, not audible output. Physical Volt/VR/Stream Deck acceptance is tracked separately from automated tests.

## License

[GNU AGPL version 3](LICENSE), AGPL-3.0-only.

## Build

Run ./scripts/build.ps1 from PowerShell (Go, MSVC and Windows SDK required for the default audio build). Exit Snoofer first. The only app output is bin/snoofer.exe. Run ./scripts/check.ps1 for tests, vet, OpenSpec validation and that same build; no duplicate build commands.
