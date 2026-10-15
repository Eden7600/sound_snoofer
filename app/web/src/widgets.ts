import type { Control, WidgetInput } from "./types.ts";
import type { MeterState } from "./model.ts";
import { display, tone, meterValue, meterBallistics, gainPosition } from "./model.ts";
import { el, button, iconButton, withIcon, ui } from "./dom.ts";
import { appState, c, updaters, widgets } from "./state.ts";
import { request, press } from "./host.ts";
export function command(id: string, label: string, parent: HTMLElement, className = "", iconName = "") {
  if (!c(id)) return null;
  const b = button(label, () => press(id), className);
  if (iconName) withIcon(b, iconName);
  parent.append(b);
  updaters.push(() => {
    b.disabled = !!appState.pending || !c(id)?.Available;
  });
  return b;
}
// control renders one semantic control. layout "row" puts the label beside the
// input, "stack" above it (narrow panels), and "bare" drops the row separators;
// hideLabel keeps the label for assistive technology only.
export function control(id: string, parent: HTMLElement, label?: string, { layout = "row", hideLabel = false } = {}) {
  const initial = c(id);
  if (!initial) return;
  const rowClass =
    layout === "stack"
      ? "block py-3 border-t border-[#25323e] first:border-t-0 first:pt-0"
      : layout === "bare"
        ? "flex items-center justify-between gap-[18px]"
        : "flex items-center justify-between gap-[18px] border-t border-[#25323e] py-3 first:border-t-0 first:pt-0 last:pb-0 max-[850px]:gap-2.5";
  const row = el("div", rowClass),
    caption = el("div", layout === "stack" ? "mb-[7px]" : "min-w-0 flex-1"),
    title = el("label", hideLabel ? "sr-only" : layout === "stack" ? "block" : "", label || initial.Label),
    note = el("small", "mt-1 block text-[11px] break-words");
  row.dataset.part = "row";
  caption.append(title, note);
  row.append(caption);
  const wide = layout === "stack";
  let input: WidgetInput;
  let editRevision: number | undefined;
  if (initial.Kind === "toggle") {
    input = button("", () => request(c(id), "press"), ui.toggle);
    input.setAttribute("aria-label", label || initial.Label);
  } else if (initial.Kind === "selection") {
    input = el("select", wide ? "w-full" : hideLabel ? "min-w-40" : "max-w-[58%] min-w-[110px]");
    input.id = "field-" + id;
    title.htmlFor = input.id;
    for (const value of initial.Options || []) {
      const option = el("option", "", initial.OptionLabels?.[value] ?? (value || "None"));
      option.value = value;
      input.append(option);
    }
    input.onfocus = () => {
      editRevision = c(id)?.Revision;
    };
    input.onchange = () => {
      request(c(id), "set", "value" in input ? input.value : "", 0, editRevision);
      input.blur();
    };
  } else if (initial.Kind === "text") {
    const field = el("input", "min-w-0 flex-1");
    field.type = "text";
    input = field;
    input.id = "field-" + id;
    title.htmlFor = input.id;
    input.onfocus = () => {
      editRevision = c(id)?.Revision;
    };
    input.onkeydown = (e) => {
      if (e.key === "Enter") {
        request(c(id), "set", "value" in input ? input.value : "", 0, editRevision);
        input.blur();
      }
      if (e.key === "Escape") {
        if ("value" in input) input.value = c(id)?.Value || "";
        input.blur();
      }
    };
    input.title = "Enter to apply";
  } else if (initial.Kind === "command") {
    input = button(label || initial.Label, () => press(id));
    title.textContent = initial.Value || "";
  } else if (initial.Kind === "numeric") {
    input = el("div", ui.actions);
    const value = el("span", "max-w-[65%] text-right break-words");
    input.append(
      iconButton("minus", "Decrease " + (label || initial.Label), () => request(c(id), "adjust", "", -1)),
      value,
      iconButton("plus", "Increase " + (label || initial.Label), () => request(c(id), "adjust", "", 1)),
    );
    updaters.push(() => {
      value.textContent = display(c(id) || initial);
      for (const b of input.querySelectorAll("button")) b.disabled = !!appState.pending || !c(id)?.Available;
    });
  } else {
    input = el("span");
  }
  if (initial.Kind === "text") {
    const editor = el("div", wide ? "flex min-w-0 gap-[7px]" : "flex min-w-0 max-w-[65%] gap-[7px]"),
      apply = button(
        id === "streamdeck.add" ? "Add" : "Apply",
        () => {
          request(c(id), "set", "value" in input ? input.value : "", 0, editRevision ?? c(id)?.Revision);
          input.blur();
        },
        ui.small,
      );
    editor.append(input, apply);
    row.append(editor);
    const enable = () => {
      apply.disabled =
        !!appState.pending || !c(id)?.Available || ("value" in input ? input.value : "") === (c(id)?.Value || "");
    };
    input.addEventListener("input", enable);
    updaters.push(enable);
  } else row.append(input);
  parent.append(row);
  widgets.push({ id, row, title, note, input, initial });
}
// meterBar is a level meter: a dimmed full gradient as the unlit track, the
// lit level clipped over it with a short transition, a held peak marker and a
// dBFS scale. update returns the reading, or null when unknown.
export function meterBar() {
  const bar = el("div", "relative mt-3 h-2.5 rounded-[3px] bg-[#283542]"),
    unlit = el("div", ui.meterGradient + " rounded-[3px] opacity-20"),
    signal = el(
      "div",
      ui.meterGradient +
        " rounded-[3px] transition-[clip-path] duration-[120ms] ease-out [clip-path:inset(0_100%_0_0)]",
    ),
    peakMark = el(
      "div",
      "absolute -top-0.5 -bottom-0.5 w-0.5 -translate-x-1/2 bg-ink data-[tone=critical]:bg-meter-red",
    );
  bar.dataset.part = "meter";
  peakMark.dataset.part = "peak";
  bar.append(unlit, signal, peakMark);
  const scale = el("div", "relative mt-1 h-3 text-[10px] text-muted tabular-nums");
  for (const db of [-60, -30, -12, -3, 0]) {
    const mark = el(
      "span",
      "absolute " + (db === -60 ? "" : db === 0 ? "-translate-x-full" : "-translate-x-1/2"),
      String(db),
    );
    mark.style.left = ((db + 60) / 60) * 100 + "%";
    scale.append(mark);
  }
  let meterState: MeterState | null = null;
  return {
    nodes: [bar, scale] as const,
    update(control?: Control) {
      const db = meterValue(control?.Meter);
      meterState = meterBallistics(meterState, db, Date.now());
      const level = meterState.level;
      signal.style.clipPath = "inset(0 " + (level === null ? 100 : -(level / 60) * 100) + "% 0 0)";
      peakMark.hidden = meterState.peak === null || meterState.peak <= -60;
      if (meterState.peak !== null) peakMark.style.left = ((meterState.peak + 60) / 60) * 100 + "%";
      peakMark.dataset.tone = meterState.peak !== null && meterState.peak >= -3 ? "critical" : "";
      return db;
    },
  };
}
export function mixer(id: string, parent: HTMLElement, title: string) {
  if (!c(id)) return;
  const node = el("div", ui.strip),
    value = el("div", ui.readout),
    text = el("div", ui.meterLabel),
    actions = el("div", ui.gainButtons);
  const device = el("div", ui.meterLabel);
  // Position track: where the gain sits in its range, with the 0 dB mark.
  const track = el("div", "relative mt-2 h-1 rounded-full bg-[#283542]"),
    fillTrack = el("div", "h-full rounded-full bg-active transition-[width] duration-[120ms]"),
    zero = el("div", "absolute -top-1 h-3 w-0.5 -translate-x-1/2 bg-ink");
  track.dataset.part = "position";
  track.append(fillTrack, zero);
  const meter = meterBar(),
    [bar, scale] = meter.nodes;
  node.dataset.part = "strip";
  node.append(el("h3", ui.stripTitle, title), device, value, track, bar, scale, text, actions);
  const mute = button("0 dB", () => press(id), ui.toggle);
  actions.append(
    iconButton("minus", title + " down", () => request(c(id), "adjust", "", -1)),
    mute,
    iconButton("plus", title + " up", () => request(c(id), "adjust", "", 1)),
  );
  // Only soundboard's press means reset; audio gain press retains its existing mute behavior.
  if (id !== "soundboard.volume") {
    mute.textContent = "Mute";
    mute.title = "Toggle mute";
  }
  parent.append(node);
  updaters.push(() => {
    const control = c(id);
    device.textContent =
      id === "audio.gain-playback"
        ? c("audio.playback-device")?.Value || ""
        : id === "audio.gain-mic"
          ? c("audio.mic-device")?.Value || ""
          : "";
    device.hidden = !device.textContent;
    value.textContent = control ? display(control) : "Unavailable";
    const db = meter.update(control);
    text.textContent = db === null ? "LEVEL N/A" : db.toFixed(1) + " dBFS";
    const position = control?.Available ? gainPosition(control.Value) : null;
    track.hidden = !position;
    if (position) {
      fillTrack.style.width = position.position * 100 + "%";
      zero.hidden = position.zero === null;
      zero.style.left = (position.zero || 0) * 100 + "%";
    }
    const muteIDs: Record<string, string> = {
      "audio.gain-mic": "audio.mic-mute",
      "audio.gain-playback": "audio.speaker-mute",
    };
    const muteID = muteIDs[id];
    if (muteID) {
      const muted = c(muteID)?.Value === "On";
      mute.dataset.tone = muted ? "critical" : "";
      mute.setAttribute("aria-pressed", String(muted));
    }
    for (const b of actions.querySelectorAll("button")) b.disabled = !!appState.pending || !control?.Available;
  });
}
export function setArt(
  art: HTMLElement,
  artwork: string | undefined,
  fallback: () => Node,
  size: string,
  animated?: string,
) {
  if (art.dataset.art === (artwork || "")) return;
  art.dataset.art = artwork || "";
  art.replaceChildren();
  if (artwork) {
    const image = el("img", size + " object-contain");
    image.src = "data:image/png;base64," + artwork;
    image.alt = "";
    art.append(image);
    if (animated) animate(art, image, animated, artwork);
  } else art.append(fallback());
}
// animate plays a control's artwork frames, fetched once per artwork; frames
// for older artwork are ignored, and playback stops when the image is replaced.
async function animate(art: HTMLElement, image: HTMLImageElement, id: string, artwork: string) {
  const desktop = window.go?.app?.Desktop;
  if (!desktop?.Animation) return;
  let frames;
  try {
    frames = await desktop.Animation(id);
  } catch {
    return;
  }
  if (!frames || frames.length < 2 || frames[0]?.Artwork !== artwork) return;
  let n = 0;
  const step = () => {
    if (!image.isConnected || art.dataset.art !== artwork) return;
    n = (n + 1) % frames.length;
    const frame = frames[n];
    if (!frame) return;
    image.src = "data:image/png;base64," + frame.Artwork;
    setTimeout(step, frame.Delay / 1e6);
  };
  setTimeout(step, frames[0].Delay / 1e6);
}
export function clipState(item: Control, active: string) {
  return item.Value === active || item.Value === "Loop"
    ? "playing"
    : item.Value === "Wait" || item.Status === "Pending"
      ? "wait"
      : tone(item) === "critical"
        ? "error"
        : "";
}
