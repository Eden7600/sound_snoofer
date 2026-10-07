# Echo cancellation probe

Run from the repository root on Windows x64 after building the DLL:

```powershell
./tools/aec-probe/run.ps1                                   # tests bin/snoofer-aec.dll
./tools/aec-probe/run.ps1 -Dll .local/aec-out/snoofer-aec.dll
```

Requires the Visual Studio C++ tools and a Windows SDK. The executable and objects stay in `.local/aec-probe`. Nothing touches Voicemeeter or an audio device.

The probe loads the DLL and calls its insert stages exactly as the monitor callback does, input insert before output insert in each cycle, on an 8-channel buffer with a stereo mic at channels 2–3 and the speaker reference at 0–1. A synthetic room plays speech-like noise (syllables, 1.8 s phrases, 0.7 s pauses) and returns it to the mic after 40 ms through a 50 ms decaying response; a local 440 Hz voice speaks in the pauses.

Checks, over the final 5 s of a 20 s run at 256- and 512-sample buffers (48 kHz):
- echo energy falls by at least 20 dB while only the speakers play;
- the local voice stays within 6 dB once the echo tail has decayed;
- every non-mic channel is bit-identical;
- every 10 ms frame is processed.

At 44.1 kHz and in bypass, the engine reports inactive and every channel is bit-identical.
