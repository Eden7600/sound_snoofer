import { ui, tc, el, button, panel, empty, iconButton, icon } from "../dom.ts";
import { root, updaters, c, appState, update } from "../state.ts";
import { request, press } from "../host.ts";
import { meterBar, setArt } from "../widgets.ts";
import { display, tone, numericValue, relativeTime } from "../model.ts";
import type { AppAudioView, AppAudioApp } from "./appaudio-types.ts";
import { viewKey } from "../view-state.ts";
import { registerDraft } from "../draft.ts";

// appEdit sends a pick or rule change for an app.
function appEdit(op: string, app: string, value = "", revision?: number) {
  request(c("appaudio.edit"), "set", JSON.stringify({ op, app, value }), 0, revision);
}
function appStrip(id: string, parent: HTMLElement) {
  const initial = c(id);
  if (!initial) return;
  const node = el("div", ui.strip + " flex flex-col"),
    head = el("div", "mb-3 flex items-center gap-3"),
    art = el("div", "grid size-10 shrink-0 place-items-center text-active"),
    name = el("h3", "min-w-0 flex-1 truncate text-sm font-semibold", initial.Label);
  node.dataset.part = "app";
  parent.append(node);
  const info = (): Partial<AppAudioApp> =>
    (c<AppAudioView>("appaudio.status")?.ViewData?.Apps || []).find((a) => a.ID === id) || {};
  const pin = iconButton("pin", "Pin " + initial.Label, () =>
    appEdit(info().Picked ? "unpick" : "pick", initial.Label),
  );
  pin.dataset.part = "pin";
  pin.className += " " + ui.toggle + " min-w-0 px-2";
  const earlier = iconButton("chevron-left", "Move " + initial.Label + " earlier", () =>
      appEdit("move", initial.Label, "up"),
    ),
    later = iconButton("chevron-right", "Move " + initial.Label + " later", () =>
      appEdit("move", initial.Label, "down"),
    );
  head.append(art, name, earlier, later, pin);
  const value = el("div", ui.readout),
    status = el("div", ui.meterLabel);
  const slider = el("input", "mt-3 w-full accent-active");
  slider.type = "range";
  slider.min = "0";
  slider.max = "100";
  slider.step = "1";
  slider.setAttribute("aria-label", initial.Label + " volume");
  slider.dataset.part = "volume";
  slider.onchange = () => request(c(id), "set", slider.value);
  const meter = meterBar(),
    mute = button("Mute", () => press(id), ui.toggle + " mt-3 self-start");
  mute.dataset.part = "mute";
  const more = el("details", "mt-3 text-[13px]"),
    summary = el("summary", "cursor-pointer text-muted", "More");
  more.append(summary);
  viewKey(more, id, initial.Label, "more");
  viewKey(summary, id, initial.Label, "more-summary");
  viewKey(mute, id, "mute");
  viewKey(slider, id, "volume");
  viewKey(pin, id, initial.Label, "pin");
  viewKey(earlier, id, initial.Label, "earlier");
  viewKey(later, id, initial.Label, "later");
  const rename = el("input", "min-w-0 flex-1");
  rename.type = "text";
  rename.placeholder = "New name";
  rename.setAttribute("aria-label", "Rename " + initial.Label);
  const renameDraft = registerDraft(rename, { revision: () => c("appaudio.edit")?.Revision, refresh: update });
  const renameRow = el("div", "mt-2 flex gap-2"),
    renameApply = button(
      "Rename",
      () => {
        if (rename.value.trim()) appEdit("rename", initial.Label, rename.value.trim(), renameDraft.revision);
      },
      ui.small,
    );
  renameRow.append(rename, renameApply);
  viewKey(rename, id, initial.Label, "rename");
  viewKey(renameApply, id, initial.Label, "rename-apply");
  const combine = el("select", "min-w-0 flex-1");
  combine.setAttribute("aria-label", "Combine " + initial.Label + " into");
  const combineDraft = registerDraft(combine, { revision: () => c("appaudio.edit")?.Revision, refresh: update });
  const combineRow = el("div", "mt-2 flex gap-2"),
    combineApply = button(
      "Combine",
      () => {
        if (combine.value) appEdit("combine", initial.Label, combine.value, combineDraft.revision);
      },
      ui.small,
    );
  combineRow.append(combine, combineApply);
  viewKey(combine, id, initial.Label, "combine");
  viewKey(combineApply, id, initial.Label, "combine-apply");
  for (const other of (c<AppAudioView>("appaudio.status")?.ViewData?.Apps || []).filter(
    (a) => !a.Hidden && a.ID !== id,
  )) {
    const option = el("option", "", "Into " + other.Name);
    option.value = other.Name;
    combine.append(option);
  }
  const hide = button("Hide", () => appEdit("hide", initial.Label), ui.small + " " + ui.danger);
  hide.dataset.part = "hide";
  viewKey(hide, id, initial.Label, "hide");
  const details = el("dl", "mt-3 grid grid-cols-[max-content_minmax(0,1fr)] gap-x-3 gap-y-1 text-[12px] text-muted");
  more.append(renameRow, combineRow, el("div", ui.actions + " mt-2"), details);
  more.children[3]?.append(hide);
  // Rules naming programs into this app (its rename, combined programs);
  // Separate undoes one.
  const naming = info().Naming || [],
    separates: HTMLButtonElement[] = [];
  if (naming.length) {
    const list = el("div", "mt-3 border-t border-[#25323e] pt-2");
    list.dataset.part = "naming";
    list.append(el("h4", "text-[11px] font-bold text-muted", "Named Here"));
    for (const rule of naming) {
      const row = el("div", "flex items-center justify-between gap-2 py-1"),
        programs = (rule.Programs || []).join(", ");
      const b = button("Separate", () => appEdit("separate", initial.Label, rule.Match), ui.small);
      viewKey(b, id, initial.Label, "separate", rule.Match);
      b.setAttribute("aria-label", "Separate " + programs);
      separates.push(b);
      row.append(el("span", "min-w-0 break-all text-ink", programs), b);
      list.append(row);
    }
    more.insertBefore(list, details);
  }
  node.append(head, value, status, slider, ...meter.nodes, mute, more);
  updaters.push(() => {
    renameDraft.sync();
    combineDraft.sync();
    const item = c(id);
    if (!item) return;
    const a = info(),
      closed = item.Value === "Closed",
      percent = numericValue(item.Value);
    setArt(art, item.Artwork, () => icon("app-window", "size-7"), "size-10");
    value.textContent = display(item);
    status.textContent = item.Status || (closed ? "Not running" : "");
    status.className = ui.meterLabel + " " + tc(tone(item));
    if (document.activeElement !== slider && percent !== null) slider.value = String(percent);
    meter.update(item);
    const muted = item.Value === "Muted";
    mute.textContent = muted ? "Muted" : "Mute";
    mute.setAttribute("aria-pressed", String(muted));
    mute.dataset.tone = muted ? "critical" : "";
    pin.setAttribute("aria-pressed", String(!!a.Picked));
    pin.title = a.Picked ? "Unpin" : "Pin";
    earlier.hidden = later.hidden = !a.Picked;
    for (const b of [slider, mute]) b.disabled = !!appState.pending || !item.Available || closed;
    for (const n of [slider, mute, ...meter.nodes]) n.hidden = closed;
    for (const b of [pin, earlier, later, renameApply, combineApply, hide, rename, combine, ...separates])
      b.disabled = !!appState.pending || !c("appaudio.edit")?.Available;
    for (const b of [renameApply, combineApply, hide, rename, combine]) b.disabled ||= closed;
    renameApply.disabled ||= !rename.value.trim();
    combineApply.disabled ||= !combine.value;
    details.replaceChildren();
    for (const [label, text] of [
      ["Programs", (a.Executables || []).join("\n")],
      ["Devices", (a.Devices || []).join("\n")],
      ["Processes", (a.PIDs || []).join(", ")],
      ["Sessions", String(a.Sessions || 0)],
      ["Rule", a.Rule || "None"],
      ["Heard", a.LastHeard ? relativeTime(a.LastHeard) : "Not yet"],
    ]) {
      details.append(el("dt", "", label), el("dd", "whitespace-pre-wrap break-all text-ink", text || "—"));
    }
  });
}
export function buildAppAudio() {
  // The recent window is edited inline; it saves on Enter or when focus leaves.
  const intro = el("div", "mb-2 flex flex-wrap items-center gap-2 text-[13px] text-muted"),
    minutes = el("input", "w-20 tabular-nums");
  intro.dataset.part = "appaudio-note";
  minutes.type = "number";
  minutes.min = "1";
  minutes.max = "1440";
  minutes.step = "1";
  minutes.setAttribute("aria-label", "Recent Window in Minutes");
  minutes.dataset.part = "recent";
  viewKey(minutes, "appaudio", "recent");
  minutes.onchange = () => {
    const value = Number(minutes.value),
      current = c<AppAudioView>("appaudio.status")?.ViewData?.RecentMinutes;
    if (Number.isInteger(value) && value >= 1 && value <= 1440 && value !== current)
      appEdit("recent", "recent", String(value));
    else minutes.value = String(current || 5);
  };
  minutes.onkeydown = (e) => {
    if (e.key === "Enter") minutes.blur();
  };
  intro.append(el("span", "", "Pinned apps first, then apps heard in the last"), minutes, el("span", "", "minutes."));
  const note = el("p", "mb-[18px] text-[13px]");
  root.append(intro, note);
  updaters.push(() => {
    const s = c<AppAudioView>("appaudio.status");
    if (document.activeElement !== minutes) minutes.value = String(s?.ViewData?.RecentMinutes || 5);
    minutes.disabled = !!appState.pending || !c("appaudio.edit")?.Available;
    note.textContent = s?.Status || (s?.Value === "Preview" ? "Preview: volumes are read, never changed." : "");
    note.hidden = !note.textContent;
    note.className = "mb-[18px] text-[13px] " + (s?.Status ? "text-critical" : "text-muted");
  });
  const apps = [...appState.controls.values()]
    .filter((v) => v.Collection === "appaudio.apps")
    .sort((a, b) => (a.Order || 0) - (b.Order || 0));
  if (!apps.length) empty(root, "No app has played sound recently. Play something, or pin an app to keep it here.");
  else {
    const grid = el("div", "grid grid-cols-[repeat(auto-fill,minmax(250px,1fr))] gap-[18px]");
    root.append(grid);
    for (const item of apps) appStrip(item.ID, grid);
  }
  excludedCard(c<AppAudioView>("appaudio.status")?.ViewData || {});
}
// excludedCard lists excluded program patterns with the running apps each
// hides, adds patterns by file name, and lists apps hidden by JSON rules.
function excludedCard(view: Partial<AppAudioView>) {
  const card = panel("Excluded", root, "mt-[18px]");
  card.dataset.part = "excluded";
  card.append(
    el(
      "p",
      "-mt-2 mb-3 text-[13px] text-muted",
      "Programs never shown here or on the deck. Use the file name; * matches anything.",
    ),
  );
  const apps = view.Apps || [],
    editable = () => !appState.pending && !!c("appaudio.edit")?.Available,
    buttons: HTMLButtonElement[] = [];
  const row = (
    title: string,
    detail: string,
    action: (event: MouseEvent) => void,
    label: string,
    aria: string,
    key: string[],
  ) => {
    const node = el("div", "flex items-center justify-between gap-3 border-t border-[#25323e] py-2.5"),
      text = el("div", "min-w-0");
    text.append(el("div", "break-all", title));
    if (detail) text.append(el("small", "block break-all text-[11px] text-muted", detail));
    const b = button(label, action, ui.small);
    b.setAttribute("aria-label", aria);
    viewKey(b, "appaudio", ...key);
    buttons.push(b);
    node.append(text, b);
    card.append(node);
  };
  for (const pattern of view.Exclude || []) {
    const hiding = apps.filter((a) => a.Hidden && a.Open && a.Rule === "Excluded (" + pattern + ")").map((a) => a.Name);
    row(
      pattern,
      hiding.length ? "Hiding " + hiding.join(", ") : "",
      () => appEdit("include", pattern, pattern),
      "Remove",
      "Remove " + pattern,
      ["include", pattern],
    );
  }
  if (!(view.Exclude || []).length) card.append(el("p", "text-[13px] text-muted", "Nothing is excluded."));
  for (const a of apps.filter((a) => a.Hidden && a.Open && !(a.Rule || "").startsWith("Excluded"))) {
    row(a.Name, a.Rule, () => appEdit("unhide", a.Name), "Unhide", "Unhide " + a.Name, [a.ID, a.Name, "unhide"]);
  }
  const add = el("div", "mt-3 flex gap-2 border-t border-[#25323e] pt-3"),
    input = el("input", "min-w-0 flex-1");
  input.type = "text";
  input.placeholder = "game.exe";
  input.setAttribute("aria-label", "Program to Exclude");
  const draft = registerDraft(input, {
    revision: () => c("appaudio.edit")?.Revision,
    refresh: update,
    completed: (value) => !!c<AppAudioView>("appaudio.status")?.ViewData?.Exclude?.includes(value.trim()),
  });
  const apply = button(
    "Exclude",
    () => {
      const v = input.value.trim();
      if (v) {
        draft.submit();
        appEdit("exclude", v, v, draft.revision);
      }
    },
    ui.small,
  );
  apply.dataset.part = "exclude";
  viewKey(input, "appaudio", "exclude");
  viewKey(apply, "appaudio", "exclude-apply");
  input.onkeydown = (e) => {
    if (e.key === "Enter") apply.click();
  };
  add.append(input, apply);
  card.append(add);
  updaters.push(() => {
    draft.sync();
    for (const b of buttons) b.disabled = !editable();
    apply.disabled = !editable() || !input.value.trim();
  });
  input.addEventListener("input", update);
}
