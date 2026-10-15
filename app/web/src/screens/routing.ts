import { ui, el, button, panel, badge, withIcon, empty, iconButton, icon } from "../dom.ts";
import { root, updaters, c, appState, update } from "../state.ts";
import { request, press } from "../host.ts";
import { display } from "../model.ts";
import { commitSelect } from "../select.ts";
import { viewKey } from "../view-state.ts";
import type { PriorityView, PriorityEntry, PrioritySuggestion, MicView, OutputSlotView } from "./routing-types.ts";

// priorityEdit sends one change to a device priority list; Snoofer validates,
// saves and applies it.
// drawnConfig is the audio configuration hash (the edit controls' Value) the screen was built
// from. Index-based edits drawn from older lists send revision 0, which the host rejects as
// stale, instead of changing whatever now sits at that index.
let drawnConfig = "";
function routingKey<T extends HTMLElement>(node: T, ...parts: Array<string | number>) {
  return viewKey(node, "routing", drawnConfig, ...parts);
}
function listEdit(id: string, value: string) {
  const control = c(id);
  request(control, "set", value, 0, control?.Value === drawnConfig ? control.Revision : 0);
}
function priorityEdit(list: string, op: string, index = 0, value = "", field = "") {
  listEdit("audio.priority-edit", JSON.stringify({ list, op, index, value, field }));
}
function editable() {
  return !appState.pending && !!c("audio.priority-edit")?.Available;
}
// matchBadge states what an entry matches under the planner's rules.
function matchBadge(entry: PriorityEntry, missing = "No match") {
  const matches = entry.Matches || [];
  const b = entry.InUse
    ? badge("In use", "active")
    : matches.length > 1
      ? badge("Ambiguous · " + matches.length, "attention")
      : badge(matches.length ? "Ready" : missing);
  b.dataset.part = "match";
  if (matches.length) b.title = matches.join("\n");
  return b;
}
// patternField is a pattern input that applies on Enter or Apply.
function patternField(
  value: string | undefined,
  label: string,
  apply: (value: string) => void,
  key: Array<string | number>,
) {
  const wrap = el("div", "flex min-w-[min(100%,16rem)] flex-1 gap-[7px]"),
    input = el("input", "min-w-0 flex-1 font-mono text-xs"),
    b = button("Apply", () => apply(input.value), ui.small);
  input.type = "text";
  input.value = String(value);
  input.setAttribute("aria-label", label);
  input.spellcheck = false;
  routingKey(input, ...key, "field");
  routingKey(b, ...key, "apply");
  input.onkeydown = (e) => {
    if (e.key === "Enter") b.click();
    if (e.key === "Escape") {
      input.value = String(value);
      input.blur();
    }
  };
  const sync = () => {
    b.disabled = !editable() || input.value === value;
    input.disabled = !editable();
  };
  input.addEventListener("input", sync);
  updaters.push(sync);
  wrap.append(input, b);
  return wrap;
}
function entryButtons(list: string, index: number, count: number, name: string) {
  const box = el("div", "flex shrink-0 gap-1"),
    buttons = [
      iconButton("chevron-up", "Move " + name + " up", () => priorityEdit(list, "move", index, "up")),
      iconButton("chevron-down", "Move " + name + " down", () => priorityEdit(list, "move", index, "down")),
      iconButton("trash-2", "Remove " + name, () => priorityEdit(list, "remove", index)),
    ] as const;
  buttons[2].classList.add(ui.danger);
  for (const [i, action] of ["up", "down", "remove"].entries()) routingKey(buttons[i]!, list, index, action);
  box.append(...buttons);
  updaters.push(() => {
    for (const b of buttons) b.disabled = !editable();
    buttons[0].disabled ||= index === 0;
    buttons[1].disabled ||= index === count - 1;
    buttons[2].disabled ||= count === 1;
  });
  return box;
}
function entryRow(parent: HTMLElement, index: number) {
  const row = el(
    "div",
    "flex flex-wrap items-center gap-2.5 border-t border-[#25323e] py-3 first:border-t-0 first:pt-0",
  );
  row.dataset.part = "priority";
  row.append(el("span", "w-5 shrink-0 text-right text-muted tabular-nums", String(index + 1)));
  parent.append(row);
  return row;
}
// suggestionRows offers connected devices no entry matches, with patterns
// Snoofer generated.
function suggestionRows(
  parent: HTMLElement,
  list: string,
  suggestions: PrioritySuggestion[] | null | undefined,
  driverLabel = true,
) {
  if (!suggestions?.length) return;
  const box = el("div", "mt-4 border-t border-[#25323e] pt-3");
  box.dataset.part = "suggestions";
  box.append(el("h3", ui.stripTitle, "Suggestions"));
  parent.append(box);
  for (const s of suggestions) {
    const row = el("div", "flex flex-wrap items-center gap-2 py-1.5"),
      name = el("span", "min-w-0 flex-1 break-words", s.Name);
    row.append(name);
    if (driverLabel) row.append(badge(s.Driver.toUpperCase()));
    // Add saves the device by identity; patterns are the advanced forms.
    const add = (
      entry: { driver: string; id?: string; name?: string; pattern?: string },
      label: string,
      title: string,
      primary: boolean,
    ) => {
      const b = withIcon(
        button(
          label,
          () => priorityEdit(list, "add", 0, JSON.stringify(entry)),
          ui.small + (primary ? " " + ui.primary : ""),
        ),
        "plus",
      );
      b.title = title;
      routingKey(b, list, JSON.stringify(entry), "add");
      b.setAttribute("aria-label", label + " " + s.Name);
      row.append(b);
      updaters.push(() => {
        b.disabled = !editable();
      });
    };
    if (s.ID) add({ driver: s.Driver, id: s.ID, name: s.Name }, "Add", "This device, even if Windows renames it", true);
    add({ driver: s.Driver, pattern: s.Exact }, "Exact Pattern", s.Exact, !s.ID);
    if (s.Device) add({ driver: s.Driver, pattern: s.Device }, "Partial Pattern", s.Device, false);
    box.append(row);
  }
}
// deviceName shows an entry chosen by identity: the device's current name, or
// its stored label while disconnected.
function deviceName(label: string | undefined, matches: string[] | null | undefined) {
  const node = el("span", "flex min-w-[min(100%,16rem)] flex-1 items-center gap-2 break-words");
  node.dataset.part = "device-entry";
  node.append(
    icon("speaker", "size-4 shrink-0 text-muted"),
    document.createTextNode(matches?.length === 1 ? String(matches[0]) : String(label)),
  );
  if (!matches?.length) node.append(badge("Disconnected"));
  return node;
}
function candidateCard(parent: HTMLElement, title: string, list: string, view: Partial<PriorityView>) {
  const card = panel(title, parent);
  card.dataset.part = "list-" + list;
  candidateRows(card, title, list, view.Lists?.[list] || [], view.Suggestions?.[list], list === "playback");
}
// candidateRows lists device entries in priority order, then suggestions.
function candidateRows(
  card: HTMLElement,
  title: string,
  list: string,
  entries: PriorityEntry[],
  suggestions: PrioritySuggestion[] | null | undefined,
  driverLabel: boolean,
) {
  entries.forEach((entry, n) => {
    const row = entryRow(card, n);
    row.append(
      entry.DeviceID
        ? deviceName(entry.DeviceName, entry.Matches)
        : patternField(
            entry.Pattern,
            title + " " + (n + 1) + " pattern",
            (value) => priorityEdit(list, "set", n, value, "pattern"),
            [list, n, "pattern"],
          ),
    );
    if (list === "playback") row.append(badge((entry.Driver || "").toUpperCase()));
    row.append(matchBadge(entry), entryButtons(list, n, entries.length, title + " " + (n + 1)));
    // An identity row already names its device.
    const names = entry.Matches || [];
    if (names.length && !(entry.DeviceID && names.length === 1))
      row.append(el("small", "basis-full pl-7 text-[11px] break-words text-muted", names.join(" · ")));
  });
  if (!entries.length) card.append(el("p", "text-muted", "No entries."));
  suggestionRows(card, list, suggestions, driverLabel);
}
function micName(view: Partial<PriorityView>, id: string | undefined) {
  return id === "off" ? "Off" : (view.Mics || []).find((m) => m.ID === id)?.Name || id || "";
}
// textField is a plain text input that applies on Enter or Apply.
function textField(
  value: string,
  label: string,
  max: number,
  apply: (value: string) => void,
  key: Array<string | number>,
) {
  const wrap = el("div", "flex min-w-[min(100%,12rem)] flex-1 gap-[7px]"),
    input = el("input", "min-w-0 flex-1"),
    b = button("Apply", () => apply(input.value.trim()), ui.small);
  input.type = "text";
  input.value = value;
  input.maxLength = max;
  routingKey(input, ...key, "field");
  routingKey(b, ...key, "apply");
  input.setAttribute("aria-label", label);
  input.onkeydown = (e) => {
    if (e.key === "Enter") b.click();
    if (e.key === "Escape") {
      input.value = value;
      input.blur();
    }
  };
  const sync = () => {
    b.disabled = !editable() || !input.value.trim() || input.value.trim() === value;
    input.disabled = !editable();
  };
  input.addEventListener("input", sync);
  updaters.push(sync);
  wrap.append(input, b);
  return wrap;
}
// micsCard defines the microphones; the number is the input each one uses.
// Device microphones list their devices in priority order.
function micsCard(parent: HTMLElement, view: Partial<PriorityView>) {
  const card = panel("Microphones", parent, "col-span-full"),
    mics = view.Mics || [];
  card.dataset.part = "mics";
  mics.forEach((m, n) => {
    const row = entryRow(card, n);
    row.dataset.part = "mic";
    const state = m.InUse
      ? badge("In use", "active")
      : m.Silent
        ? badge("Silent", "attention")
        : badge(m.Option ? "Ready" : "Unavailable");
    state.dataset.part = "mic-state";
    row.append(
      textField(m.Name, "Microphone " + (n + 1) + " name", 16, (v) => priorityEdit("mics", "set", n, v, "name"), [
        "mics",
        m.ID,
        "name",
      ]),
      micSource(m, n, view),
      readyToggle(m, n),
      state,
      entryButtons("mics", n, mics.length, m.Name),
    );
    if (m.Device) {
      const box = el("div", "basis-full pl-7");
      candidateRows(
        box,
        m.Name,
        "mic-devices:" + m.ID,
        view.Lists?.["mic-devices:" + m.ID] || [],
        view.Suggestions?.["mic-devices:"],
        false,
      );
      row.append(box);
    }
  });
  if (mics.length < 5) {
    // A new microphone uses interface channels, or a connected input device.
    const add = el("div", "mt-4 flex flex-wrap items-center gap-2 border-t border-[#25323e] pt-3"),
      name = el("input", "w-40"),
      kind = el("select", "min-w-0 flex-1");
    add.dataset.part = "mic-add";
    name.type = "text";
    name.placeholder = "Microphone";
    name.maxLength = 16;
    name.setAttribute("aria-label", "New Microphone Name");
    kind.setAttribute("aria-label", "New Microphone Source");
    routingKey(name, "new-mic", "name");
    routingKey(kind, "new-mic", "source");
    kind.append(Object.assign(el("option", "", "Interface Channels"), { value: "" }));
    const devices = view.Suggestions?.["mic-devices:"] || [];
    devices.forEach((s, n) => {
      const o = el("option", "", s.Name);
      o.value = String(n);
      kind.append(o);
    });
    const added = () => {
      if (kind.value === "") return { name: name.value.trim() };
      const s = devices[Number(kind.value)];
      if (!s) return null;
      return {
        name: name.value.trim(),
        devices: [s.ID ? { driver: s.Driver, id: s.ID, name: s.Name } : { driver: s.Driver, pattern: s.Exact }],
      };
    };
    const b = withIcon(
      button(
        "Add Microphone",
        () => {
          const value = added();
          if (name.value.trim() && value) priorityEdit("mics", "add", 0, JSON.stringify(value));
        },
        ui.small,
      ),
      "plus",
    );
    name.onkeydown = (e) => {
      if (e.key === "Enter") b.click();
    };
    name.addEventListener("input", update);
    updaters.push(() => {
      name.disabled = kind.disabled = !editable();
      b.disabled = !editable() || !name.value.trim();
    });
    routingKey(b, "new-mic", "add");
    add.append(name, kind, b);
    card.append(add);
  }
}
// micSource switches a microphone between interface channels and a
// connected input device; a device microphone keeps its current devices.
function micSource(m: MicView, n: number, view: Partial<PriorityView>) {
  const select = el("select", "min-w-0 max-w-[16rem]"),
    devices = view.Suggestions?.["mic-devices:"] || [];
  select.setAttribute("aria-label", m.Name + " Source");
  routingKey(select, "mics", m.ID, "source");
  select.append(Object.assign(el("option", "", "Interface Channels"), { value: "interface" }));
  if (m.Device) select.append(Object.assign(el("option", "", "Device"), { value: "current" }));
  devices.forEach((s, i) => {
    const o = el("option", "", s.Name);
    o.value = String(i);
    select.append(o);
  });
  select.value = m.Device ? "current" : "interface";
  commitSelect(select, () => {
    if (select.value === "interface") {
      priorityEdit("mics", "set", n, "interface", "source");
      return;
    }
    const s = devices[Number(select.value)];
    if (!s) return;
    priorityEdit(
      "mics",
      "set",
      n,
      JSON.stringify(s.ID ? { driver: s.Driver, id: s.ID, name: s.Name } : { driver: s.Driver, pattern: s.Exact }),
      "source",
    );
  });
  updaters.push(() => {
    select.disabled = !editable();
  });
  return select;
}
// readyToggle asserts a microphone is usable whenever present: Auto never
// skips it for silence.
function readyToggle(m: MicView, n: number) {
  const b = button(
    "Always Ready",
    () => priorityEdit("mics", "set", n, String(!m.Ready), "ready"),
    ui.toggle + " " + ui.small,
  );
  b.setAttribute("aria-pressed", String(!!m.Ready));
  b.setAttribute("aria-label", m.Name + " always ready");
  b.title = "Auto never skips it for silence";
  routingKey(b, "mics", m.ID, "ready");
  updaters.push(() => {
    b.disabled = !editable();
  });
  return b;
}
// channelFields sets one interface microphone's channels: Off, Mono (one
// channel) or Stereo (left and right).
function channelFields(entry: PriorityEntry, n: number, m: MicView) {
  const field = "input:" + m.ID,
    current = entry.Inputs?.[m.ID] || [],
    wrap = el("div", "flex flex-wrap items-center gap-1.5");
  wrap.dataset.part = "channels";
  const mode = el("select", "w-24");
  mode.setAttribute("aria-label", m.Name + " on interface " + (n + 1));
  routingKey(mode, "interfaces", n, m.ID, "mode");
  for (const [value, label] of [
    ["off", "Off"],
    ["mono", "Mono"],
    ["stereo", "Stereo"],
  ] as const)
    mode.append(Object.assign(el("option", "", label), { value }));
  mode.value = current.length === 2 ? "stereo" : current.length ? "mono" : "off";
  const number = (value: number | undefined, label: string) => {
    const i = el("input", "w-16 tabular-nums");
    i.type = "number";
    i.min = "1";
    i.max = "64";
    i.value = String(value);
    i.setAttribute("aria-label", label);
    return i;
  };
  const left = current[0] || 1,
    name = m.Name,
    where = " for interface " + (n + 1);
  const inputs =
    mode.value === "stereo"
      ? [number(left, name + " left channel" + where), number(current[1], name + " right channel" + where)]
      : mode.value === "mono"
        ? [number(left, name + " channel" + where)]
        : [];
  const send = (value: string) => priorityEdit("interfaces", "set", n, value, field);
  commitSelect(mode, () =>
    send(mode.value === "off" ? "" : mode.value === "mono" ? String(left) : left + "," + (current[1] || left + 1)),
  );
  for (const i of inputs) i.onchange = () => send(inputs.map((x) => x.value).join(","));
  inputs.forEach((input, channel) => routingKey(input, "interfaces", n, m.ID, "channel", channel));
  updaters.push(() => {
    for (const x of [mode, ...inputs]) x.disabled = !editable();
  });
  wrap.append(el("span", "text-xs text-muted", name), mode, ...inputs);
  return wrap;
}
// activityCard edits metering: how many leading options are metered (Off
// removes metering), the silence threshold and the delay before Auto skips.
function activityCard(parent: HTMLElement, view: Partial<PriorityView>) {
  if (!view.Profiles) return;
  const a = view.Activity;
  const card = panel("Activity", parent),
    fields = el("div", "grid grid-cols-[max-content_minmax(0,1fr)] items-center gap-x-3 gap-y-2");
  card.dataset.part = "activity";
  const check = el("select", "min-w-0");
  check.setAttribute("aria-label", "Meter");
  routingKey(check, "activity", "check");
  check.append(Object.assign(el("option", "", "Off"), { value: "0" }));
  for (let k = 1; k <= 4; k++) check.append(Object.assign(el("option", "", "Top " + k), { value: String(k) }));
  check.value = String(a?.check ?? 0);
  commitSelect(check, () => priorityEdit("activity", "set", 0, check.value, "check"));
  const number = (value: number | undefined, label: string, field: string, min: number, max: number) => {
    const i = el("input", "w-24 tabular-nums");
    i.type = "number";
    i.min = String(min);
    i.max = String(max);
    i.value = a ? String(value) : "";
    i.setAttribute("aria-label", label);
    i.onchange = () => priorityEdit("activity", "set", 0, i.value, field);
    routingKey(i, "activity", field);
    return i;
  };
  const threshold = number(a?.silence_db, "Silence Below (dBFS)", "silence_db", -120, -20),
    delay = number(a?.silent_after_s, "Silent After (s)", "silent_after_s", 2, 600);
  fields.append(
    el("span", "text-xs text-muted", "Meter"),
    check,
    el("span", "text-xs text-muted", "Silence Below (dBFS)"),
    threshold,
    el("span", "text-xs text-muted", "Silent After (s)"),
    delay,
  );
  card.append(fields);
  updaters.push(() => {
    check.disabled = !editable();
    threshold.disabled = delay.disabled = !editable() || !a;
  });
}
function interfaceCard(parent: HTMLElement, view: Partial<PriorityView>) {
  const card = panel("Interfaces", parent, "col-span-full"),
    entries = view.Lists?.interfaces || [];
  card.dataset.part = "list-interfaces";
  entries.forEach((entry, n) => {
    const row = entryRow(card, n),
      fields = el("div", "grid min-w-0 flex-1 grid-cols-[max-content_minmax(0,1fr)] items-center gap-x-3 gap-y-2");
    fields.append(
      el("span", "text-xs text-muted", "Driver"),
      entry.ASIOID
        ? deviceName(entry.ASIOName, entry.Drivers)
        : patternField(
            entry.ASIOPattern,
            "Interface " + (n + 1) + " driver pattern",
            (v) => priorityEdit("interfaces", "set", n, v, "asio_pattern"),
            ["interfaces", n, "asio_pattern"],
          ),
    );
    fields.append(
      el("span", "text-xs text-muted", "Presence"),
      entry.PresenceID
        ? deviceName(entry.PresenceName, entry.Matches)
        : patternField(
            entry.PresencePattern,
            "Interface " + (n + 1) + " presence pattern",
            (v) => priorityEdit("interfaces", "set", n, v, "presence_pattern"),
            ["interfaces", n, "presence_pattern"],
          ),
    );
    const channels = el("div", "flex flex-wrap items-center gap-x-4 gap-y-2");
    for (const m of (view.Mics || []).filter((m) => !m.Device)) channels.append(channelFields(entry, n, m));
    fields.append(el("span", "text-xs text-muted", "Channels"), channels);
    row.append(fields, matchBadge(entry), entryButtons("interfaces", n, entries.length, "interface " + (n + 1)));
    const drivers = entry.Drivers || [];
    row.append(
      el(
        "small",
        "basis-full pl-7 text-[11px] break-words text-muted",
        (entry.Matches || []).join(" · ") + (drivers.length ? " · driver " + drivers.join(", ") : " · no driver"),
      ),
    );
  });
  // A new interface pairs an installed driver with the WDM input that proves
  // its hardware is connected.
  const drivers = view.Drivers,
    inputs = view.Inputs;
  if (drivers?.length && inputs?.length) {
    const add = el("div", "mt-4 flex flex-wrap items-center gap-2 border-t border-[#25323e] pt-3"),
      driver = el("select", "min-w-0 flex-1"),
      input = el("select", "min-w-0 flex-1");
    add.dataset.part = "interface-add";
    driver.setAttribute("aria-label", "New Interface Driver");
    input.setAttribute("aria-label", "New Interface Presence Input");
    routingKey(driver, "new-interface", "driver");
    routingKey(input, "new-interface", "presence");
    drivers.forEach((s, n) => {
      const o = el("option", "", s.Name);
      o.value = String(n);
      driver.append(o);
    });
    inputs.forEach((s, n) => {
      const o = el("option", "", s.Name);
      o.value = String(n);
      input.append(o);
    });
    // Both are saved by identity when Snoofer knows it, else as exact patterns.
    const added = () => {
      const d = drivers[Number(driver.value)],
        p = inputs[Number(input.value)];
      if (!d || !p) return null;
      return {
        ...(d.ID ? { asio_id: d.ID, asio_name: d.Name } : { asio_pattern: d.Exact }),
        ...(p.ID ? { presence_id: p.ID, presence_name: p.Name } : { presence_pattern: p.Exact }),
        inputs: {},
      };
    };
    const b = withIcon(
      button(
        "Add",
        () => {
          const value = added();
          if (value) priorityEdit("interfaces", "add", 0, JSON.stringify(value));
        },
        ui.small,
      ),
      "plus",
    );
    updaters.push(() => {
      b.disabled = driver.disabled = input.disabled = !editable();
    });
    routingKey(b, "new-interface", "add");
    add.append(el("h3", ui.stripTitle + " mb-0 basis-full", "Add Interface"), driver, input, b);
    card.append(add);
  }
}
function micPriorityCard(parent: HTMLElement, view: Partial<PriorityView>) {
  const card = panel("Mic Priority", parent),
    entries = view.Lists?.microphones || [];
  card.dataset.part = "list-microphones";
  entries.forEach((entry, n) => {
    const row = entryRow(card, n);
    row.append(el("span", "min-w-0 flex-1", micName(view, entry.ID)));
    row.append(
      entry.InUse ? badge("In use", "active") : badge(entry.Option ? "Ready" : "Unavailable"),
      entryButtons("microphones", n, entries.length, micName(view, entry.ID)),
    );
  });
  const missing = (view.Microphones || []).filter((id) => !entries.some((e) => e.ID === id));
  if (missing.length) {
    const add = el("div", "mt-4 flex gap-2 border-t border-[#25323e] pt-3"),
      select = el("select", "min-w-0 flex-1");
    select.setAttribute("aria-label", "Microphone to Add");
    routingKey(select, "microphones", "add-choice");
    for (const id of missing) {
      const o = el("option", "", micName(view, id));
      o.value = id;
      select.append(o);
    }
    const b = withIcon(
      button("Add", () => priorityEdit("microphones", "add", 0, select.value), ui.small),
      "plus",
    );
    updaters.push(() => {
      b.disabled = select.disabled = !editable();
    });
    routingKey(b, "microphones", "add");
    add.append(select, b);
    card.append(add);
  }
}
const sourceNames: Record<string, string> = {
  "virtual:1": "Computer",
  "virtual:2": "Virtual 2",
  "virtual:3": "Virtual 3",
  monitor: "Monitor",
  soundboard: "Soundboard",
  tape: "Tape",
};
const sourceIcons: Record<string, string> = {
  "virtual:1": "monitor",
  "virtual:2": "monitor",
  "virtual:3": "monitor",
  monitor: "headphones",
  soundboard: "music",
  tape: "cassette-tape",
};
const outputTones: Record<string, string> = { "In use": "active", "No output": "attention" };
// outputEdit adds, renames, re-assigns or removes an output slot by ID.
function outputEdit(op: string, id: string, value = "") {
  listEdit("audio.output-edit", JSON.stringify({ op, id, value }));
}
// routeCell is one source-to-destination switch; pressing it toggles (or
// cycles Monitor on Playback). Moving focus never dispatches.
function routeCell(id: string, label: string) {
  const b = button("", () => press(id), ui.toggle + " w-full min-w-0 px-2 py-1.5");
  b.dataset.part = "route";
  routingKey(b, id, "route");
  b.setAttribute("aria-label", label);
  updaters.push(() => {
    const item = c(id),
      on = !!item && item.Value !== "Off" && item.Value !== "off";
    b.textContent = item ? display(item) : "N/A";
    b.setAttribute("aria-pressed", String(on));
    b.disabled = !!appState.pending || !item?.Available;
  });
  return b;
}
function outputsCard(parent: HTMLElement, view: Partial<PriorityView>) {
  const card = panel("Outputs", parent, "col-span-full");
  card.dataset.part = "outputs";
  const slots = [1, 2, 3]
    .map((n) => "audio.slot-" + n)
    .filter((id) => {
      const slot = c(id);
      return slot && !slot.Hidden;
    });
  const editableOutputs = () => !appState.pending && !!c("audio.output-edit")?.Available;
  const playbackSources = [...appState.controls.keys()]
    .filter((id) => id.startsWith("audio.playback:"))
    .map((id) => id.slice("audio.playback:".length))
    .sort();
  const sources = [...playbackSources, "monitor"];
  for (const source of ["soundboard", "tape"])
    if (slots.some((id) => c(id + ":" + source)?.Available)) sources.push(source);
  const scroll = el("div", "overflow-x-auto"),
    table = el("table", "w-full min-w-[520px] border-separate border-spacing-x-2 border-spacing-y-1.5 text-[13px]");
  scroll.append(table);
  viewKey(scroll, "routing", "output-scroll");
  card.append(scroll);
  const head = el("tr"),
    thead = el("thead"),
    tbody = el("tbody");
  table.append(thead, tbody);
  thead.append(head);
  head.append(el("th", "w-[150px]"));
  const column = (title: string, device: () => string | undefined, stateID: string) => {
    const th = el("th", "min-w-[150px] pb-2 text-left align-top font-normal"),
      name = el("div", "flex items-center gap-1.5 font-semibold"),
      deviceText = el("div", "mt-0.5 text-[11px] break-words text-muted"),
      state = badge("");
    name.append(icon("speaker", "size-4 shrink-0 text-muted"), document.createTextNode(title));
    th.append(name, deviceText, state);
    head.append(th);
    updaters.push(() => {
      deviceText.textContent = device() || "";
      const v = stateID ? c(stateID)?.Value || "" : "";
      state.textContent = v;
      state.dataset.tone = outputTones[v] || "";
      state.hidden = !v;
    });
    return th;
  };
  column("Playback", () => c("audio.playback-device")?.Value, "").dataset.part = "output-playback";
  for (const id of slots) {
    const slot = c<OutputSlotView>(id);
    if (!slot) continue;
    const slotID = slot.ViewData?.ID || "",
      deviceName = slot.ViewData?.Device || "",
      th = column(slot.Label, () => c<OutputSlotView>(id)?.ViewData?.Device, id);
    th.dataset.part = "output-slot";
    // Slot management: device, rename, remove.
    const tools = el("div", "mt-2 flex flex-col gap-1.5"),
      device = el("select", "w-full min-w-0 text-xs"),
      rename = el("input", "min-w-0 flex-1 text-xs"),
      renameRow = el("div", "flex gap-1.5");
    device.setAttribute("aria-label", slot.Label + " device");
    routingKey(device, "output", slotID, "device");
    routingKey(rename, "output", slotID, "name");
    device.append(Object.assign(el("option", "", "No device"), { value: "" }));
    // The current device keeps its saved form; others are saved by identity.
    if (deviceName) {
      const o = el("option", "", deviceName);
      o.value = "current";
      device.append(o);
    }
    const choices = (view.OutputDevices || []).filter((d) => d.Name !== deviceName);
    choices.forEach((d, n) => {
      const o = el("option", "", d.Name);
      o.value = String(n);
      device.append(o);
    });
    device.value = deviceName ? "current" : "";
    commitSelect(device, () => {
      if (device.value === "") {
        outputEdit("device", slotID, "");
        return;
      }
      const d = choices[Number(device.value)];
      if (!d) return;
      outputEdit("device", slotID, d.ID ? JSON.stringify({ ID: d.ID, Name: d.Name }) : d.Name);
    });
    rename.type = "text";
    rename.value = slot.Label;
    rename.maxLength = 16;
    rename.setAttribute("aria-label", "Rename " + slot.Label);
    const apply = button("Rename", () => outputEdit("rename", slotID, rename.value.trim()), ui.small),
      remove = iconButton("trash-2", "Remove " + slot.Label, () => outputEdit("remove", slotID));
    remove.classList.add(ui.danger);
    routingKey(apply, "output", slotID, "rename");
    routingKey(remove, "output", slotID, "remove");
    rename.onkeydown = (e) => {
      if (e.key === "Enter") apply.click();
    };
    renameRow.append(rename, apply, remove);
    tools.append(device, renameRow);
    th.append(tools);
    updaters.push(() => {
      for (const n of [device, rename, remove]) n.disabled = !editableOutputs();
      apply.disabled = !editableOutputs() || !rename.value.trim() || rename.value.trim() === slot.Label;
    });
  }
  for (const source of sources) {
    const tr = el("tr"),
      th = el("th", "text-left font-normal text-muted"),
      label = el("span", "inline-flex items-center gap-2");
    label.append(
      icon(sourceIcons[source] || "audio-lines", "size-4 shrink-0"),
      document.createTextNode(sourceNames[source] || source),
    );
    th.append(label);
    tr.append(th);
    const playback = el("td");
    playback.dataset.part = "cell";
    if (source === "monitor") playback.append(routeCell("audio.monitor", "Playback Monitor"));
    else if (source === "tape") playback.append(el("span", "text-xs text-muted", "Always"));
    else if (c("audio.playback:" + source))
      playback.append(routeCell("audio.playback:" + source, "Playback " + (sourceNames[source] || source)));
    tr.append(playback);
    for (const id of slots) {
      const td = el("td");
      td.dataset.part = "cell";
      if (c(id + ":" + source)?.Available)
        td.append(routeCell(id + ":" + source, (c(id)?.Label || "") + " " + (sourceNames[source] || source)));
      tr.append(td);
    }
    tbody.append(tr);
  }
  // A new output may start without a device; one can be assigned later from
  // connected devices that are neither Playback nor another slot.
  if (slots.length < 3) {
    const add = el("div", "mt-4 flex flex-wrap items-center gap-2 border-t border-[#25323e] pt-3"),
      name = el("input", "w-40"),
      device = el("select", "min-w-0 flex-1");
    add.dataset.part = "output-add";
    name.type = "text";
    name.placeholder = "Monitor output";
    name.maxLength = 16;
    name.setAttribute("aria-label", "New Output Name");
    device.setAttribute("aria-label", "New Output Device");
    routingKey(name, "new-output", "name");
    routingKey(device, "new-output", "device");
    device.append(Object.assign(el("option", "", "No device"), { value: "" }));
    const devices = view.OutputDevices || [];
    devices.forEach((d, n) => {
      const o = el("option", "", d.Name);
      o.value = String(n);
      device.append(o);
    });
    const added = () => {
      const d = devices[Number(device.value)];
      return d
        ? { name: name.value.trim(), device: d.Name, ...(d.ID ? { device_id: d.ID } : {}) }
        : { name: name.value.trim() };
    };
    const b = withIcon(
      button(
        "Add Output",
        () => {
          if (name.value.trim()) outputEdit("add", "", JSON.stringify(added()));
        },
        ui.small,
      ),
      "plus",
    );
    name.onkeydown = (e) => {
      if (e.key === "Enter") b.click();
    };
    updaters.push(() => {
      name.disabled = device.disabled = !editableOutputs();
      b.disabled = !editableOutputs() || !name.value.trim();
    });
    name.addEventListener("input", update);
    routingKey(b, "new-output", "add");
    add.append(name, device, b);
    card.append(add);
  }
}
export function buildRouting() {
  drawnConfig = c("audio.priority-edit")?.Value || "";
  const status = c<PriorityView>("audio.priorities");
  if (!status) {
    empty(root, "Audio is disabled. Enable it in Plugins.");
    return;
  }
  const view: Partial<PriorityView> = status.ViewData || {},
    error = el("p", "mb-[18px] text-critical");
  error.dataset.part = "priority-error";
  root.append(error);
  updaters.push(() => {
    error.textContent = c("audio.priority-edit")?.Status || "";
    error.hidden = !error.textContent;
  });
  const grid = el("div", ui.grid);
  root.append(grid);
  outputsCard(grid, view);
  micsCard(grid, view);
  interfaceCard(grid, view);
  candidateCard(grid, "Playback", "playback", view);
  micPriorityCard(grid, view);
  activityCard(grid, view);
}
