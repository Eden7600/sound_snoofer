import { isBusy, actionFeedback } from "../actions.ts";
import { ui, tc, el, button, panel, iconButton, icon } from "../dom.ts";
import { root, updaters, c, appState } from "../state.ts";
import { request, press } from "../host.ts";
import { control, setArt, clipState } from "../widgets.ts";
import { display, tone, numericValue, sceneRoom } from "../model.ts";
import type { Control } from "../types.ts";
import type { HueStatusView, HueRoom } from "./lights-types.ts";
import { viewKey } from "../view-state.ts";

function lightsSetup(parent: HTMLElement) {
  const card = el("section", ui.panel + " mb-[18px] border-[#345365]"),
    title = el("h2", ui.h2),
    text = el("p", "mt-2 mb-3.5 text-muted"),
    actions = el("div", ui.actions);
  card.dataset.part = "setup";
  const pair = button("Pair", () => press("hue.pair"), ui.primary);
  viewKey(pair, "hue.pair", "press");
  actions.append(pair);
  card.append(title, text, actions);
  actionFeedback("hue.pair", card);
  parent.append(card);
  updaters.push(() => {
    const status = c<HueStatusView>("hue.status"),
      p = c("hue.pair"),
      info = status?.ViewData || {},
      value = status?.Value || "";
    let heading = "",
      body = "",
      accent = "",
      canPair = false;
    if (p?.Value === "Press button") {
      heading = "Press the link button";
      body =
        "Press the round button on top of the Hue Bridge" +
        (info.bridge ? " at " + info.bridge : "") +
        " within 30 seconds.";
      accent = "attention";
    } else if (p?.Status) {
      heading = "Pairing failed";
      body = p.Status;
      accent = "critical";
      canPair = true;
    } else if (value === "Not paired") {
      heading = info.bridge ? "Bridge found at " + info.bridge : "Bridge found";
      body = "Pair once so Snoofer can control your lights.";
      canPair = true;
    } else if (value === "No bridge") {
      heading = "Searching for the Hue Bridge";
      body =
        "No bridge has answered yet. If it is on another network, set hue.address in snoofer.json to its IP address.";
    } else if (value === "Multiple bridges") {
      heading = "Multiple bridges found";
      body = status?.Status || "";
    } else if (value === "Error") {
      heading = "Bridge needs pairing";
      body = status?.Status || "";
      accent = "critical";
      canPair = true;
    } else if (value === "Disconnected") {
      heading = "Connecting to the Hue Bridge";
      body = status?.Status || "";
    }
    card.hidden = !heading;
    title.textContent = heading;
    title.className = ui.h2 + " " + tc(accent);
    text.textContent = body;
    pair.hidden = !canPair;
    pair.textContent = value === "Error" ? "Pair Again" : "Pair";
    pair.disabled = isBusy("hue.pair") || !p?.Available;
  });
}
function lightDial(
  id: string,
  parent: HTMLElement,
  title: string,
  pressLabel: string,
  step: number,
  min: number,
  max: number,
) {
  if (!c(id)) return;
  const node = el("div", ui.strip),
    value = el("div", ui.readout),
    slider = el("input", "mt-3 w-full border-0 bg-transparent p-0 accent-active"),
    note = el("div", ui.meterLabel),
    actions = el("div", ui.gainButtons);
  node.dataset.part = "strip";
  slider.type = "range";
  slider.min = String(min);
  slider.max = String(max);
  slider.step = "any";
  slider.setAttribute("aria-label", title);
  viewKey(slider, id, "slider");
  node.append(el("h3", ui.stripTitle, title), value, slider, note, actions);
  parent.append(node);
  actionFeedback(id, node);
  actions.append(
    viewKey(
      iconButton("minus", title + " down", () => request(c(id), "adjust", "", -1)),
      id,
      "decrease",
    ),
    viewKey(
      button(pressLabel, () => press(id)),
      id,
      "press",
    ),
    viewKey(
      iconButton("plus", title + " up", () => request(c(id), "adjust", "", 1)),
      id,
      "increase",
    ),
  );
  // The plugin owns absolute values; the slider sends the equivalent relative ticks on release.
  slider.onchange = () => {
    const current = numericValue(c(id)?.Value);
    if (current === null) return;
    const ticks = Math.round((Number(slider.value) - current) / step);
    if (ticks) request(c(id), "adjust", "", ticks);
  };
  updaters.push(() => {
    const control = c(id),
      number = numericValue(control?.Value);
    value.textContent = control ? display(control) : "Unavailable";
    value.className = ui.readout + (control?.Subdued ? " text-attention" : "");
    if (document.activeElement !== slider && number !== null) slider.value = String(number);
    slider.disabled = isBusy(id) || !control?.Available || number === null;
    note.textContent = control?.Status || "";
    note.className = ui.meterLabel + " " + tc(tone(control));
    for (const b of actions.querySelectorAll("button")) b.disabled = isBusy(id) || !control?.Available;
  });
}
function segmented(id: string, title: string, parent: HTMLElement) {
  const initial = c(id);
  if (!initial) return;
  const row = el("div", "mb-4"),
    group = el("div", "flex");
  group.setAttribute("role", "group");
  group.setAttribute("aria-label", title);
  row.append(el("h3", ui.stripTitle, title), group);
  parent.append(row);
  actionFeedback(id, row);
  updaters.push(() => {
    row.hidden = !!c(id)?.Hidden;
  });
  for (const option of initial.Options || []) {
    const b = button(
      initial.OptionLabels?.[option] ?? option,
      () => request(c(id), "set", option),
      "-ml-px min-w-0 flex-1 rounded-none px-1 py-2 first:ml-0 first:rounded-l-[7px] last:rounded-r-[7px] aria-pressed:relative aria-pressed:border-active-edge aria-pressed:bg-active-bg aria-pressed:text-active",
    );
    group.append(b);
    viewKey(b, id, "option", option);
    updaters.push(() => {
      const item = c(id);
      b.setAttribute("aria-pressed", String(item?.Value === option));
      b.disabled = isBusy(id) || !item?.Available;
    });
  }
}
function sceneCard(id: string, parent: HTMLElement) {
  const b = button("", () => press(id), ui.scene),
    art = el("div", ui.sceneArt),
    name = el("strong", "text-[13px] break-words"),
    status = el("small", "mt-[5px] text-[11px] text-muted");
  b.dataset.part = "scene";
  viewKey(b, id, "scene");
  b.append(art, name, status);
  const wrap = el("div", "min-w-0");
  b.classList.add("w-full");
  wrap.append(b);
  actionFeedback(id, wrap);
  parent.append(wrap);
  updaters.push(() => {
    const item = c(id);
    if (!item) return;
    name.textContent = item.ShortLabel || item.Label;
    b.title = item.Label;
    b.disabled = isBusy(id) || !item.Available;
    b.dataset.state = clipState(item, "Active");
    status.textContent = item.Status === "Pending" ? "Wait" : item.Status || display(item);
    status.className = "mt-[5px] text-[11px] " + (tc(tone(item)) || "text-muted");
    setArt(art, item.Artwork, () => icon("lightbulb", "size-7"), "size-10");
  });
}
// roomsCard chooses the rooms and zones Snoofer controls, here and on the deck.
function roomsCard(parent: HTMLElement) {
  const item = c<HueRoom[] | null>("hue.rooms");
  if (!item?.ViewData?.length) return;
  const card = panel("Rooms", parent, "col-span-full");
  card.dataset.part = "rooms";
  card.append(el("p", "text-[13px] text-muted", "Only checked rooms appear here and on the deck."));
  const list = el("div", "mt-3 grid grid-cols-[repeat(auto-fill,minmax(180px,1fr))] gap-2"),
    error = el("p", "mt-2 text-critical");
  card.append(list, error);
  actionFeedback("hue.rooms", card);
  const boxes: HTMLInputElement[] = [];
  for (const room of item.ViewData) {
    const label = el("label", "flex items-center gap-2.5"),
      box = el("input", "size-4 accent-active");
    box.type = "checkbox";
    box.value = room.ID;
    viewKey(box, "hue.rooms", room.ID);
    label.append(box, el("span", "", room.Name + (room.Kind === "zone" ? " (zone)" : "")));
    list.append(label);
    boxes.push(box);
    box.onchange = () =>
      request(
        c<HueRoom[] | null>("hue.rooms"),
        "set",
        boxes
          .filter((b) => b.checked)
          .map((b) => b.value)
          .join(","),
      );
  }
  updaters.push(() => {
    const current = c<HueRoom[] | null>("hue.rooms"),
      chosen = new Map((current?.ViewData || []).map((r) => [r.ID, r.Chosen]));
    for (const b of boxes) b.checked = !!chosen.get(b.value);
    const count = boxes.filter((b) => b.checked).length;
    // The last chosen room stays chosen.
    for (const b of boxes) b.disabled = isBusy("hue.rooms") || !current?.Available || (b.checked && count === 1);
    error.textContent = current?.Status || "";
    error.hidden = !error.textContent;
  });
}
export function buildLights() {
  lightsSetup(root);
  const grid = el(
    "div",
    "grid grid-cols-[minmax(0,1.5fr)_minmax(0,1fr)] items-start gap-[18px] max-[1120px]:grid-cols-1",
  );
  grid.dataset.part = "lights";
  root.append(grid);
  const room = panel("Room", grid),
    roomNote = el("p", ui.note);
  room.dataset.part = "room";
  room.append(roomNote);
  const roomHead = room.querySelector<HTMLElement>("[data-part=panel-head]");
  if (roomHead) control("hue.group", roomHead, "Room", { layout: "bare", hideLabel: true });
  const dials = el("div", "mt-4 grid grid-cols-1 gap-5");
  room.append(dials);
  lightDial("hue.brightness", dials, "BRIGHTNESS", "On/Off", 2, 1, 100);
  // Motion sensors appear only while the room has some.
  const motion = el("div", "mt-4");
  motion.dataset.part = "motion";
  room.append(motion);
  control("hue.motion", motion, "Motion Sensors");
  updaters.push(() => {
    const item = c("hue.motion");
    motion.hidden = !item || !!item.Hidden;
  });
  const sortRow = el("div", "mt-4");
  sortRow.dataset.part = "sort";
  room.append(sortRow);
  segmented("hue.sort", "Sort", sortRow);
  updaters.push(() => {
    roomNote.textContent =
      c("hue.sync")?.Value === "On" ? "Hue Sync is driving the lights · Brightness adjusts the sync" : "";
    roomNote.hidden = !roomNote.textContent;
  });
  const groupControl = c("hue.group"),
    roomName = (groupControl?.OptionLabels?.[groupControl.Value] || "").replace(/ \(zone\)$/, "");
  const scenes = [...appState.controls.values()].filter((v) => v.ID.startsWith("hue.scene-"));
  // Scenes follow the host's chosen sort (Order within each room), then name.
  const byOrder = (a: Control, b: Control) =>
    (a.Order || 0) - (b.Order || 0) || a.ShortLabel.localeCompare(b.ShortLabel);
  const own = scenes.filter((v) => roomName && sceneRoom(v) === roomName).sort(byOrder);
  const sceneGrid = el("div", ui.sceneGrid);
  sceneGrid.dataset.part = "room-scenes";
  room.append(sceneGrid);
  if (own.length) for (const scene of own) sceneCard(scene.ID, sceneGrid);
  else
    sceneGrid.append(el("p", "text-muted", roomName ? "No scenes in this room." : "Choose a room to see its scenes."));
  const others = scenes.filter((v) => !own.includes(v));
  if (others.length) {
    const details = el("details", "mt-4"),
      summary = el("summary", "", "Other Rooms");
    details.append(summary);
    viewKey(details, "hue", groupControl?.Value || "", "other-rooms");
    viewKey(summary, "hue", groupControl?.Value || "", "other-rooms-summary");
    room.append(details);
    for (const name of [...new Set(others.map(sceneRoom))].sort()) {
      const heading = el("h3", "mt-4 text-xs font-bold text-muted", name || "Other");
      heading.dataset.part = "room-heading";
      details.append(heading);
      const g = el("div", ui.sceneGrid);
      details.append(g);
      for (const scene of others.filter((v) => sceneRoom(v) === name).sort(byOrder)) sceneCard(scene.ID, g);
    }
  }
  const sync = panel("Hue Sync", grid),
    syncState = el("p", "-mt-1.5 mb-3 text-[13px]"),
    toggle = button("", () => press("hue.sync"), ui.toggle + " mb-[18px] w-full p-3.5 text-base"),
    help = el("p", "text-muted");
  sync.append(syncState, toggle);
  actionFeedback("hue.sync", sync);
  viewKey(toggle, "hue.sync", "toggle");
  segmented("hue.sync-mode", "Mode", sync);
  segmented("hue.sync-intensity", "Intensity", sync);
  sync.append(help);
  updaters.push(() => {
    const status = c("hue.sync-status"),
      item = c("hue.sync"),
      on = item?.Value === "On",
      failed = !!item?.Status && item.Status !== "Pending";
    syncState.textContent = status?.Value === "N/A" ? "Not connected" : status?.Value || "";
    syncState.className = "-mt-1.5 mb-3 text-[13px] " + (on ? "text-active" : tc(tone(status)));
    toggle.textContent = item?.Status === "Pending" ? "Wait" : on ? "Stop Sync" : "Start Sync";
    toggle.setAttribute("aria-pressed", String(on));
    toggle.dataset.tone = failed ? "critical" : "";
    toggle.disabled = isBusy("hue.sync") || !item?.Available;
    help.textContent =
      status?.Value === "N/A"
        ? "Open Hue Sync and turn on Settings → Third-party control."
        : status?.Value === "No bridge"
          ? "Hue Sync has no bridge connection."
          : failed
            ? item?.Status || ""
            : "";
    help.className = failed ? "text-critical" : "text-muted";
    help.hidden = !help.textContent;
  });
  roomsCard(grid);
}
