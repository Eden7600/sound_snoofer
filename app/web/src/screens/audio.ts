import { ui, el, panel, empty, iconButton, icon } from "../dom.ts";
import { root, updaters, c, appState } from "../state.ts";
import { press } from "../host.ts";
import { control, command, mixer } from "../widgets.ts";

export function buildAudio() {
  if (![...appState.controls.keys()].some((id) => id.startsWith("audio."))) {
    empty(root, "Audio is disabled. Enable it in Plugins.");
    return;
  }
  const grid = el("div", ui.grid);
  root.append(grid);
  const live = panel("Live Controls", grid, "col-span-full"),
    top = el("div", "mb-5 grid grid-cols-3 gap-5"),
    strips = el("div", "grid grid-cols-2 gap-5 max-[850px]:gap-[9px]");
  control("audio.health", live, "Engine");
  live.append(top);
  control("audio.mic-stack", top, "Mic Stack", { layout: "bare" });
  control("audio.mic-mute", top, "Mic Mute", { layout: "bare" });
  control("audio.speaker-mute", top, "Playback Mute", { layout: "bare" });
  live.append(strips);
  mixer("audio.gain-mic", strips, "MICROPHONE");
  mixer("audio.gain-playback", strips, "PLAYBACK");
  control("audio.interface", live, "Interface");
  const used = new Set([
    "audio.health",
    "audio.mic-stack",
    "audio.mic-mute",
    "audio.speaker-mute",
    "audio.gain-mic",
    "audio.gain-playback",
    "audio.interface",
    "audio.playback-device",
  ]);
  for (const [title, prefix] of [
    ["Normal", "audio.normal-"],
    ["VR", "audio.vr-profile-"],
  ] as const) {
    if (!c(prefix + "source")) continue;
    const card = panel(title, grid),
      note = el("p", ui.note);
    card.append(note);
    updaters.push(() => {
      note.textContent = c(prefix + "source")?.Subdued ? "VR override · Normal remains editable" : "";
      note.hidden = !note.textContent;
    });
    for (const [key, label] of [
      ["source", "Microphone"],
      ["mode", "Processing"],
      ["monitor", "Monitor"],
      ["output", "Playback"],
    ] as const) {
      control(prefix + key, card, label);
      used.add(prefix + key);
    }
  }
  const recording = [...appState.controls.values()].filter((v) => v.Group === "Recording" && !v.SurfaceOnly);
  if (recording.length) {
    const card = panel("Recording", grid);
    for (const item of recording) {
      control(item.ID, card);
      used.add(item.ID);
    }
    if (c("audio.tape-play")) transport(card);
  }
  if (c("aec.mode")) {
    const card = panel("Echo Cancellation", grid);
    card.dataset.part = "echo";
    control("aec.status", card, "Status");
    control("aec.mode", card, "Mode");
    control("aec.engine", card, "Engine");
    control("aec.strength", card, "Strength");
    control("aec.timing", card, "Timing");
    const actions = el("div", ui.actions + " mt-3");
    card.append(actions);
    command("aec.retry", "Retry", actions);
  }
  const other = [...appState.controls.values()].filter(
    (v) => v.ID.startsWith("audio.") && !v.SurfaceOnly && !used.has(v.ID) && v.Group !== "Routing",
  );
  if (other.length || c("audio.engine-restart")) {
    const card = panel("Routing & Recovery", grid);
    for (const item of other) control(item.ID, card);
    const engine = el("div", ui.actions + " mt-3");
    card.append(engine);
    command("audio.engine-restart", "Restart Audio Engine", engine, "", "refresh-cw");
  }
}
// transport is the Recording card's recorder row: Record, then playback of
// the loaded file. Labels follow the recorder state; availability comes from
// the controls, so nothing here can act on a recording.
function transport(card: HTMLElement) {
  const row = el("div", ui.actions + " mt-3");
  row.dataset.part = "tape";
  card.append(row);
  const record = command("audio.record-toggle", "Record", row, "", "circle-dot"),
    play = command("audio.tape-play", "Play", row, "", "play");
  // Transport keys with nothing to do in the recorder's state are inactive:
  // disabled here as on the deck.
  const stop = command("audio.tape-stop", "Stop", row, "", "square");
  if (stop)
    updaters.push(() => {
      stop.disabled = !!appState.pending || !c("audio.tape-stop")?.Available || !!c("audio.tape-stop")?.Inactive;
    });
  for (const [id, name, label] of [
    ["audio.tape-rew", "rewind", "Rewind"],
    ["audio.tape-ff", "fast-forward", "Fast-forward"],
  ] as const) {
    if (!c(id)) continue;
    const b = iconButton(name, label, () => press(id));
    row.append(b);
    updaters.push(() => {
      b.disabled = !!appState.pending || !c(id)?.Available || !!c(id)?.Inactive;
    });
  }
  const relabel = (b: HTMLButtonElement | null, text: string, name: string) => {
    if (b && b.dataset.label !== text) {
      b.dataset.label = text;
      b.replaceChildren(icon(name, "size-4 shrink-0"), text);
    }
  };
  updaters.push(() => {
    // Labels follow the keys' icons: Stop Rec while recording or paused, and
    // Pause while the tape plays or a recording runs.
    const recording = c("audio.record-toggle")?.Icon === "record-stop",
      pausing = c("audio.tape-play")?.Icon === "tape-pause";
    relabel(record, recording ? "Stop Rec" : "Record", recording ? "square" : "circle-dot");
    relabel(play, pausing ? "Pause" : "Play", pausing ? "pause" : "play");
  });
}
