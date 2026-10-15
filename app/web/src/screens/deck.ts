import { ui, el, button, panel, empty, icon } from "../dom.ts";
import { appState, root, widgets, updaters, c, update } from "../state.ts";
import { request, press } from "../host.ts";
import { control, command } from "../widgets.ts";
import { compatible, gridMove } from "../model.ts";
import type { DeckView } from "./deck-types.ts";

interface DeckRange {
  page?: string;
  anchor: number;
  end: number;
  dials?: boolean;
}

// deckRange is the GUI-local key rectangle, or dial span (dials: true, slot
// indexes after the keys), for creating regions; selecting it never
// dispatches anything. It resets when the edited page changes.
let deckRange: DeckRange | null = null;
// deckShape is the edited layout's grid: key count, columns and dial count.
function deckShape() {
  const v = c<DeckView>("streamdeck.preview")?.ViewData;
  return {
    keys: v?.Keys.length ?? 0,
    columns: v?.Columns ?? 1,
    dials: v?.Dials.length ?? 0,
  };
}
function rangeKeys(range: DeckRange | null): number[] {
  if (!range) return [];
  if (range.dials) {
    const out = [];
    for (let i = Math.min(range.anchor, range.end); i <= Math.max(range.anchor, range.end); i++) out.push(i);
    return out;
  }
  const n = deckShape().columns;
  const top = Math.min(Math.floor(range.anchor / n), Math.floor(range.end / n)),
    bottom = Math.max(Math.floor(range.anchor / n), Math.floor(range.end / n));
  const left = Math.min(range.anchor % n, range.end % n),
    right = Math.max(range.anchor % n, range.end % n);
  const out = [];
  for (let row = top; row <= bottom; row++) for (let col = left; col <= right; col++) out.push(row * n + col);
  return out;
}
function regionDialsLabel(first: number, last: number) {
  const a = Math.min(first, last) + 1,
    b = Math.max(first, last) + 1;
  return a === b ? "Dial " + a : "Dials " + a + "–" + b;
}
function regionKeysLabel(first: number, last: number) {
  const keys = rangeKeys({ anchor: first, end: last }),
    start = keys[0],
    end = keys.at(-1);
  if (start === undefined || end === undefined) return "";
  return keys.length === 1 ? "Key " + (start + 1) : "Keys " + (start + 1) + "–" + (end + 1);
}
export function buildDeck() {
  const preview = c<DeckView>("streamdeck.preview");
  if (!preview?.ViewData) {
    empty(root, "Stream Deck is disabled or its editor is not ready.");
    return;
  }
  const toolbar = el("div", ui.toolbar);
  const selectors = el("div", ui.grid + " col-span-full flex-1");
  toolbar.append(selectors);
  control("streamdeck.profile", selectors, "Device");
  control("streamdeck.page", selectors, "Page");
  root.append(toolbar);
  const shape = preview.ViewData,
    K = shape.Keys.length,
    D = shape.Dials.length,
    columns = shape.Columns;
  const workspace = el(
      "div",
      "grid grid-cols-[minmax(0,1fr)_280px] items-start gap-[18px] min-[1500px]:grid-cols-[minmax(0,1fr)_330px] max-[1120px]:grid-cols-1",
    ),
    left = el("div"),
    inspector = el("section", ui.panel + " sticky top-5 max-[1120px]:static");
  inspector.dataset.part = "inspector";
  workspace.append(left, inspector);
  root.append(workspace);
  const frame = el("div", "mb-5 rounded-[18px] border border-[#2e414f] bg-deck p-[19px]"),
    keys = el("div", "grid gap-[7px]"),
    dials = el("div", "mt-[22px] grid gap-2");
  frame.append(keys, dials);
  left.append(frame);
  // The grid follows the model; narrow decks keep keys near their + XL size.
  keys.style.gridTemplateColumns = "repeat(" + columns + ",minmax(0,1fr))";
  frame.style.maxWidth = columns < 9 ? columns * 88 + 38 + "px" : "";
  dials.style.gridTemplateColumns = "repeat(" + (D + (shape.NavDial ? 1 : 0)) + ",minmax(0,1fr))";
  dials.hidden = !D && !shape.NavDial;
  keys.setAttribute("aria-label", "Deck Keys");
  dials.setAttribute("aria-label", "Deck Dials");
  const keyClass =
    "relative flex aspect-square min-w-0 flex-col items-center justify-center overflow-hidden rounded-[7px] bg-key px-[3px] py-[5px] text-[9px] data-[empty=true]:border-[#1c2a36] data-[empty=true]:bg-key-empty data-[region=true]:bg-active-bg data-[range=true]:outline-2 data-[range=true]:outline-dashed data-[range=true]:outline-active aria-pressed:border-2 aria-pressed:border-active aria-pressed:shadow-[0_0_0_2px_#183a46] min-[1500px]:text-xs max-[1120px]:text-[11px]";
  const dialClass =
    "relative data-[region=true]:bg-active-bg data-[range=true]:outline-2 data-[range=true]:outline-dashed data-[range=true]:outline-active flex min-h-[62px] min-w-0 flex-col items-center justify-center bg-key px-1 py-[7px] text-[10px] before:mb-[5px] before:size-[18px] before:rounded-full before:border-2 before:border-[#60798a] before:content-[''] aria-pressed:border-2 aria-pressed:border-active aria-pressed:shadow-[0_0_0_2px_#183a46] *:max-w-full *:truncate";
  function slotButton(index: number, dial: boolean) {
    const b = button(
      "",
      (e) => {
        // Shift extends the region selection without selecting a binding.
        if (e.shiftKey && deckRange && !!deckRange.dials === dial) {
          deckRange.end = index;
          update();
          return;
        }
        deckRange = {
          page: c("streamdeck.page")?.Value,
          anchor: index,
          end: index,
          dials: dial,
        };
        request(c("streamdeck.slot"), "set", (dial ? "Dial " : "Key ") + (dial ? index - K + 1 : index + 1));
      },
      dial ? dialClass : keyClass,
    );
    b.dataset.part = dial ? "dial" : "deck-key";
    const number = el(
        "small",
        "absolute top-0.5 left-1 text-[8px] text-muted",
        String(dial ? index - K + 1 : index + 1),
      ),
      glyph = el("span", "leading-tight text-active"),
      name = el("span", "max-w-full truncate"),
      source = el("span", "text-[8px] text-attention"),
      regionTag = el("small", "absolute right-1 bottom-0.5 text-[8px] text-active");
    if (!dial) b.append(number, glyph);
    b.append(regionTag);
    b.append(name, source);
    (dial ? dials : keys).append(b);
    b.onkeydown = (e) => {
      if (!e.key.startsWith("Arrow")) return;
      e.preventDefault();
      if (e.shiftKey) {
        if (!deckRange || !!deckRange.dials !== dial)
          deckRange = {
            page: c("streamdeck.page")?.Value,
            anchor: index,
            end: index,
            dials: dial,
          };
        deckRange.end = dial
          ? K + gridMove(deckRange.end - K, e.key, D, D)
          : gridMove(deckRange.end, e.key, columns, K);
        update();
        const focused = (dial ? dials : keys).children[dial ? deckRange.end - K : deckRange.end];
        if (focused instanceof HTMLElement) focused.focus();
        return;
      }
      const list = [...(dial ? dials : keys).querySelectorAll<HTMLButtonElement>("button:not(:disabled)")];
      const next = gridMove(list.indexOf(b), e.key, dial ? D : columns, list.length);
      list[next]?.focus();
    };
    updaters.push(() => {
      const view = c<DeckView>("streamdeck.preview")?.ViewData;
      if (!view) return;
      const slot = dial ? view.Dials[index - K] : view.Keys[index];
      if (!slot) return;
      const item = c(slot.Control);
      b.setAttribute("aria-pressed", String(view.Selected === index));
      const region = slot.Region ?? -1,
        inRange = rangeKeys(deckRange).includes(index);
      b.dataset.region = String(region >= 0);
      b.dataset.range = String(inRange);
      regionTag.textContent = region >= 0 ? "R" + (region + 1) : "";
      b.setAttribute(
        "aria-label",
        (dial ? "Dial " + (index - K + 1) : "Key " + (index + 1)) +
          ": " +
          (item?.Label || slot.Label || "Empty") +
          (slot.Source ? " · " + slot.Source : "") +
          (region >= 0 ? " · Region " + (region + 1) : "") +
          (inRange ? " · In selection" : ""),
      );
      b.title = b.getAttribute("aria-label") || "";
      b.disabled = !!appState.pending;
      b.dataset.empty = String(!slot.Control);
      name.textContent = item?.ShortLabel || item?.Label || slot.Label || "";
      source.textContent = slot.Source || "";
      const art = item?.Artwork || "";
      if (glyph.dataset.art !== art || glyph.dataset.control !== slot.Control) {
        glyph.dataset.art = art;
        glyph.dataset.control = slot.Control;
        glyph.replaceChildren();
        if (art) {
          const image = el("img", "size-[30px] object-contain min-[1500px]:size-12");
          image.src = "data:image/png;base64," + art;
          image.alt = "";
          glyph.append(image);
        } else if (slot.Control) glyph.append(symbolIcon(slot.Control));
      }
    });
  }
  // reserved is the page control: the Pages key of decks without dials, or the last dial.
  const reserved = (base: string, parent: HTMLElement) => {
    const r = el(
      "div",
      base.replace(/(aria-pressed|data-\[\w+=true\]):\S+ ?/g, "") + " rounded-[7px] border border-line",
    );
    r.dataset.part = "reserved";
    r.append(el("span", "", "Pages"), el("small", "text-muted", "Reserved"));
    parent.append(r);
  };
  for (let i = 0; i < K; i++) {
    if (i === shape.Reserved) reserved(keyClass, keys);
    else slotButton(i, false);
  }
  for (let i = K; i < K + D; i++) slotButton(i, true);
  if (shape.NavDial) reserved(dialClass, dials);
  const tools = panel("Page", left, "grid gap-3");
  control("streamdeck.name", tools, "Name");
  const commands = el("div", ui.actions + " justify-end");
  tools.append(commands);
  command("streamdeck.earlier", "Earlier", commands, "", "chevron-left");
  command("streamdeck.later", "Later", commands, "", "chevron-right");
  const makeHome = command("streamdeck.home", "Make Home", commands, "", "house");
  if (makeHome)
    updaters.push(() => {
      if (c<DeckView>("streamdeck.preview")?.ViewData?.Home) makeHome.disabled = true;
    });
  const more = el("details", "mt-4"),
    summary = el("summary", "", "Page Options");
  more.append(summary);
  tools.append(more);
  control("streamdeck.add", more, "New Page");
  regionsPanel(left);
  decksPanel(left);
  const deletion = el("div", ui.actions + " justify-end");
  more.append(deletion);
  const del = command("streamdeck.delete", "Delete Page", deletion, ui.danger);
  if (del)
    del.onclick = () => {
      if (window.confirm("Delete this page from the draft? Save applies the deletion.")) press("streamdeck.delete");
    };
  const selected = preview.ViewData.Selected,
    dial = selected >= K;
  const eyebrow = el(
    "div",
    "text-[10px] font-semibold tracking-[1.7px] text-muted",
    dial ? "DIAL " + (selected - K + 1) : "KEY " + (selected + 1),
  );
  eyebrow.dataset.part = "eyebrow";
  inspector.append(eyebrow, el("h2", ui.h2, "Binding"));
  const ownership = el("p", ui.note + " mt-1");
  inspector.append(ownership);
  updaters.push(() => {
    const v = c<DeckView>("streamdeck.preview")?.ViewData;
    if (!v) return;
    const slot = dial ? v.Dials[selected - K] : v.Keys[selected];
    if (!slot) return;
    ownership.className = slot.Source ? ui.note + " mt-1" : "mt-1 mb-[15px] text-xs text-muted";
    ownership.textContent =
      slot.Source === "Auto"
        ? "Auto-filled · assigning a binding pins this position"
        : slot.Source === "Shared"
          ? "Shared Across Pages"
          : "Page binding";
  });
  control("streamdeck.shared", inspector, "Edit Shared Binding", {
    layout: "stack",
  });
  const search = el("input", "w-full");
  search.type = "search";
  search.placeholder = "Find an action";
  search.setAttribute("aria-label", "Find a Binding");
  const select = el(
    "select",
    "my-2.5 min-h-[220px] max-h-[300px] w-full text-xs max-[1120px]:min-h-[150px] *:p-[7px] *:whitespace-normal",
  );
  select.size = 9;
  select.setAttribute("aria-label", "Binding");
  inspector.append(search, select);
  function populate() {
    const value = c("streamdeck.binding")?.Value || "";
    select.replaceChildren();
    const clear = el("option", "", "Clear Binding");
    clear.value = "";
    select.append(clear);
    for (const item of appState.controls.values()) {
      // Go-to and scroll keys are the only Stream Deck controls that can be bound.
      if (
        (item.ID.startsWith("streamdeck.") &&
          !item.ID.startsWith("streamdeck.goto-") &&
          !item.ID.startsWith("streamdeck.scroll-") &&
          item.ID !== "streamdeck.reset-focus") ||
        !compatible(item, dial)
      )
        continue;
      const pluginID = item.ID.split(".")[0] || item.ID;
      const label = item.Label + " · " + (appState.state.PluginLabels?.[pluginID] || pluginID);
      if (!(label + " " + pluginID).toLocaleLowerCase().includes(search.value.toLocaleLowerCase()) && item.ID !== value)
        continue;
      const option = el("option", "", label);
      option.value = item.ID;
      option.title = item.ID;
      select.append(option);
    }
    if (value && ![...select.options].some((o) => o.value === value)) {
      const option = el("option", "", value + " · unavailable");
      option.value = value;
      select.append(option);
    }
    select.value = value;
  }
  search.oninput = populate;
  populate();
  let bindingRevision: number | undefined;
  select.onfocus = () => {
    bindingRevision = c("streamdeck.binding")?.Revision;
  };
  select.onchange = () => {
    request(c("streamdeck.binding"), "set", select.value, 0, bindingRevision);
    select.blur();
  };
  updaters.push(() => {
    select.disabled = !!appState.pending || !c("streamdeck.binding")?.Available;
    if (document.activeElement !== select) select.value = c("streamdeck.binding")?.Value || "";
  });
  const bar = el(
      "div",
      "sticky bottom-3 mt-[18px] flex items-center justify-between gap-3 rounded-[10px] border border-[#345365] bg-[#192a36] px-[18px] py-3.5",
    ),
    status = el("p", "text-xs"),
    actions = el("div", ui.actions);
  bar.append(status, actions);
  root.append(bar);
  const discard = command("streamdeck.cancel", "Discard", actions),
    save = command("streamdeck.save", "Save Layout", actions, ui.primary);
  updaters.push(() => {
    const v = c<DeckView>("streamdeck.preview")?.ViewData;
    status.textContent = c("streamdeck.status")?.Value || "";
    status.className = "text-xs " + (v?.Dirty ? "text-attention" : "text-muted");
    if (discard) discard.disabled = !!appState.pending || !v?.Dirty;
    if (save) save.disabled = !!appState.pending || !v?.Dirty;
  });
}
// decksPanel holds Paging and the remembered decks: name, own layout and
// Forget. Dropping an own layout and forgetting a deck are confirmed.
function decksPanel(parent: HTMLElement) {
  if (!c("streamdeck.paging")) return;
  const card = panel("Decks", parent, "mt-5 grid gap-3");
  card.dataset.part = "decks";
  control("streamdeck.paging", card, "Paging");
  if (!(c("streamdeck.deck")?.Options || []).length) return;
  control("streamdeck.deck", card, "Deck");
  control("streamdeck.deck-name", card, "Deck Name");
  control("streamdeck.deck-own", card, "Own Layout");
  const own = widgets.at(-1);
  if (own?.id === "streamdeck.deck-own")
    own.input.onclick = () => {
      if (
        c("streamdeck.deck-own")?.Value !== "On" ||
        window.confirm("Use the model layout on this deck? Its own layout is deleted.")
      )
        press("streamdeck.deck-own");
    };
  const actions = el("div", ui.actions + " justify-end");
  card.append(actions);
  const forget = command("streamdeck.deck-forget", "Forget Deck", actions, ui.danger);
  if (forget)
    forget.onclick = () => {
      if (window.confirm("Forget this deck and its own layout?")) press("streamdeck.deck-forget");
    };
}
function regionsPanel(parent: HTMLElement) {
  const view = c<DeckView>("streamdeck.preview")?.ViewData;
  if (!view || !c("streamdeck.region-add")) return;
  const card = panel("Regions", parent, "mt-5 grid gap-3");
  card.dataset.part = "regions";
  const collections = view.Collections || [];
  const sourceSelect = (value: string, label: string) => {
    const select = el("select", "min-w-0 flex-1");
    for (const item of collections) {
      const option = el("option", "", item.Label);
      option.value = item.ID;
      select.append(option);
    }
    if (value && !collections.some((item) => item.ID === value)) {
      const option = el("option", "", label || value);
      option.value = value;
      select.append(option);
    }
    select.value = value || collections[0]?.ID || "";
    return select;
  };
  const sets = el("p", "text-xs");
  sets.dataset.part = "page-sets";
  card.append(sets);
  updaters.push(() => {
    const v = c<DeckView>("streamdeck.preview")?.ViewData,
      count = v?.Sets || 1;
    sets.hidden = count <= 1;
    sets.textContent = count + " sets · " + (v?.Scrolls ? "Up/Down" : "page dial visits each");
    sets.className = "text-xs " + (v?.Scrolls ? "text-muted" : "text-attention");
  });
  const regions = view.Regions || [];
  if (!regions.length)
    card.append(el("p", ui.note, "No regions. Select keys or dials (Shift+click or Shift+arrows) and add one."));
  regions.forEach((region, i) => {
    const row = el("div", "flex flex-wrap items-center gap-2.5");
    row.dataset.part = "region";
    const name = el("strong", "w-7 text-xs text-active", "R" + (i + 1)),
      select = sourceSelect(region.Source, region.Label),
      keysLabel = el(
        "small",
        "text-muted",
        region.Legacy
          ? "Whole page · prefix"
          : region.Dials
            ? regionDialsLabel(region.First, region.Last)
            : regionKeysLabel(region.First, region.Last),
      );
    select.setAttribute("aria-label", "Region " + (i + 1) + " Source");
    let editRevision: number | undefined;
    select.onfocus = () => {
      editRevision = c("streamdeck.region-source")?.Revision;
    };
    select.onchange = () => {
      request(c("streamdeck.region-source"), "set", i + "," + select.value, 0, editRevision);
      select.blur();
    };
    const stacked = region.Stacked ? el("small", "text-muted", "+ " + region.Stacked) : null;
    // Overflow adds sets; off saves the region clipped to the keys it covers.
    const overflow = el("label", "flex items-center gap-1.5 text-xs"),
      box = el("input", "size-4 accent-active");
    box.type = "checkbox";
    box.checked = !region.Clip;
    box.setAttribute("aria-label", "Region " + (i + 1) + " Overflow");
    overflow.append(box, document.createTextNode("Overflow"));
    overflow.dataset.part = "region-overflow";
    overflow.title = "Adds sets when the region has more than fits";
    box.onchange = () => request(c("streamdeck.region-clip"), "set", i + "," + (box.checked ? "off" : "on"));
    const remove = button("Remove", () => request(c("streamdeck.region-remove"), "set", String(i)), ui.danger);
    remove.setAttribute("aria-label", "Remove Region " + (i + 1));
    row.append(name, select);
    if (stacked) row.append(stacked);
    row.append(keysLabel);
    if (c("streamdeck.region-clip")) row.append(overflow);
    row.append(remove);
    card.append(row);
    updaters.push(() => {
      select.disabled = remove.disabled = box.disabled = !!appState.pending;
      const saved = (c<DeckView>("streamdeck.preview")?.ViewData?.Regions || [])[i];
      if (saved && !appState.pending) box.checked = !saved.Clip;
    });
  });
  const add = el("div", "flex flex-wrap items-center gap-2.5 border-t border-[#25323e] pt-3"),
    source = sourceSelect("", ""),
    create = button(
      "",
      () => {
        if (!deckRange) return;
        request(c("streamdeck.region-add"), "set", deckRange.anchor + "," + deckRange.end + "," + source.value);
        deckRange = null;
      },
      ui.primary,
    );
  source.setAttribute("aria-label", "New Region Source");
  create.dataset.part = "add-region";
  add.append(source, create);
  card.append(add);
  updaters.push(() => {
    if (deckRange && deckRange.page !== c("streamdeck.page")?.Value) deckRange = null;
    const keysInRange = rangeKeys(deckRange),
      current = c<DeckView>("streamdeck.preview")?.ViewData;
    const K = deckShape().keys;
    const overlaps = keysInRange.some((k) => ((k >= K ? current?.Dials[k - K] : current?.Keys[k])?.Region ?? -1) >= 0);
    create.textContent = deckRange
      ? "Add Region · " +
        (deckRange.dials
          ? regionDialsLabel(deckRange.anchor - K, deckRange.end - K)
          : regionKeysLabel(deckRange.anchor, deckRange.end))
      : "Add Region";
    create.disabled = !!appState.pending || !deckRange || overlaps || !collections.length;
    create.title = overlaps ? "The selection overlaps a region" : !deckRange ? "Select keys or dials first" : "";
    source.disabled = !collections.length;
  });
}
// Deck key previews map a control's deck icon to the closest Lucide icon.
const deckIcons: Record<string, string> = {
  "mic-mute": "mic",
  call: "phone",
  "call-leave": "phone-off",
  "discord-deafen": "headphones",
  "discord-deafen-muted": "headphone-off",
  "discord-video": "video",
  "discord-video-off": "video-off",
  "screen-share": "screen-share",
  "discord-connect": "link",
  "record-mic": "mic",
  "vr-mic": "mic",
  "speaker-mute": "volume-2",
  "vr-playback": "volume-2",
  "record-computer": "monitor",
  monitor: "headphones",
  "mic-stack": "power",
  "mode-direct": "audio-lines",
  "mode-element": "audio-lines",
  "tap-pre": "audio-lines",
  "tap-post": "audio-lines",
  "record-toggle": "circle-dot",
  "record-start": "circle-dot",
  "record-stop": "square",
  "soundboard-play": "play",
  "soundboard-stop": "square",
  "soundboard-overlap": "layers",
  "media-prev": "skip-back",
  "media-next": "skip-forward",
  "media-play": "play",
  "open-controls": "sliders-horizontal",
  defaults: "sliders-horizontal",
  "engine-restart": "refresh-cw",
  "hue-scene": "lightbulb",
  "hue-brightness": "sun",
  "hue-pair": "link",
  "huesync-sync": "monitor",
  "huesync-mode": "layers",
  "huesync-intensity": "waves",
  "hue-motion": "radar",
  "hue-motion-off": "radar",
  "hue-sort": "sort",
  "deck-page": "folder",
  "app-audio": "app-window",
  "focus-reset": "crosshair",
  echo: "audio-lines",
  "tape-play": "play",
  "tape-pause": "pause",
  "tape-stop": "square",
  "tape-rew": "rewind",
  "tape-ff": "fast-forward",
  "deck-up": "chevron-up",
  "deck-down": "chevron-down",
};
function symbolIcon(id: string) {
  const mapped = deckIcons[c(id)?.Icon || ""];
  const name =
    mapped ||
    (/mute/.test(id)
      ? "volume-2"
      : /mic|source|mode/.test(id)
        ? "mic"
        : /play|clip/.test(id)
          ? "play"
          : /stop/.test(id)
            ? "square"
            : /record/.test(id)
              ? "circle-dot"
              : /prev/.test(id)
                ? "skip-back"
                : /next/.test(id)
                  ? "skip-forward"
                  : /scene/.test(id)
                    ? "lightbulb"
                    : "circle");
  return icon(name, "size-6 min-[1500px]:size-8");
}
