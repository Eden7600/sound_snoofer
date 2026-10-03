# Hardware acceptance

## Observed on 2026-10-03

Read-only `devices --json` connected to the installed **Voicemeeter Banana (edition 2)** using registry DLL discovery. It listed these WDM endpoints:

- Input: `INPUT 1/2 (Volt 2)`.
- Input: `Microphone (2- Insta360 Link 2)`.
- Input: `Microphone (3- SteelSeries Arena 9)`.
- Input: `CABLE Output (VB-Audio Virtual Cable)`.
- Output: `MONITOR L/R (Volt 2)`.
- Output: `Speakers (3- SteelSeries Arena 9)`.
- Output: the regular and 16-channel VB-Audio Cable inputs.

AirPods were not present as a WDM endpoint in this snapshot. Hardware input assignments and A2/A3 were empty. A1 reported `Speakers (3- SteelSeries Arena `, matching the enumerated truncated MME name. A WDM configuration would propose a change to the full WDM name. No live assignments were changed during development.

## Manual checklist (pending)

1. Confirm the microphone input strip and output bus to manage, plus the Volt 2/Element topology. Use narrow name regexes in a personal config; inspect a dry-run and save the current inventory/assignments.
2. While dry-run watch is running, power-cycle Volt 2 and connect/disconnect AirPods. Confirm the WDM inventory refreshes, the desired microphone switches between Volt and Insta360, and output switches between AirPods and Arena 9. Confirm transient Bluetooth churn does not cause a premature decision.
3. If the Remote API holds stale WDM entries, implement/test Core Audio active-endpoint availability before signing off automatic switching. Do not mask staleness with engine restarts.
4. Run one-shot apply and listen/record to verify actual routing. An accepted setter/readback alone is not a listening test. Record any actual restart requirements.
5. Run live watch through disconnect/reconnect and all-candidates-absent cases. Confirm idle polling does not keep resetting assignments.
6. Restart Voicemeeter during watch. Confirm the watcher reconnects and revalidates configuration. Compare Element patch/insert, gain, mute, and strip-to-bus settings before and after.
7. Test on Potato when available, including A4/A5 or inputs 4/5, and rejection of those targets after returning to Banana.

These checks require the physical devices and human listening. They are separate from the automated fake-backend tests. Keep the OpenSpec change active until required acceptance is complete.

## Personal configuration update

The user confirmed Hardware Input 1 for microphones. `config.local.json` now manages `input:1` with `(?i)volt.*2` ahead of `(?i)insta360.*link.*2`. Playback is not included pending output-bus confirmation. This configuration has only been previewed; physical switching and listening checks remain pending.

## Confirmed studio model (supersedes WDM-only microphone setup)

Volt uses ASIO A1; channels 1/1 feed strip 1 and channels 2/2 feed strip 2. Playback uses the lowest free bus and moves its sends. Primary VAIO is always routed to playback through the semantic virtual:1 rule. The personal config now uses studio mode. The latest read-only preview observed Volt already on A1 and SteelSeries on A2; it proposed patch values [1,1,2,2] and VAIO A2 enabled/A1 disabled. These were previewed, not applied.

Remaining manual acceptance: verify channel isolation with the lav on Volt input 2, playback audibility through both ASIO-presence states, VAIO-send drift repair, physical presence freshness, and unchanged Element inserts/unmanaged routing.
