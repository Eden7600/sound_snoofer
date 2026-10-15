import { ui, tc, el, button, panel, empty, icon } from "../dom.ts";
import { root, updaters, c, appState, update } from "../state.ts";
import { request, press } from "../host.ts";
import { command, control, mixer, setArt, clipState } from "../widgets.ts";
import { display, tone } from "../model.ts";

export function buildSoundboard() {
  const clips = [...appState.controls.values()].filter((v) => v.ID.startsWith("soundboard.clip-"));
  const toolbar = el("div", ui.toolbar),
    search = el("input", "min-w-[170px] flex-1");
  search.type = "search";
  search.placeholder = "Search Clips";
  search.setAttribute("aria-label", "Search Clips");
  toolbar.append(icon("search", "size-5 shrink-0 text-muted"), search);
  command("soundboard.stop", "Stop", toolbar, "", "square");
  root.append(toolbar);
  if (c("soundboard.volume")) {
    const card = panel("Playback", root, "mb-[18px]");
    control("soundboard.status", card, "Library");
    control("soundboard.overlap", card, "Overlap Clips");
    mixer("soundboard.volume", card, "VOLUME");
  }
  if (!clips.length) {
    empty(root, "No clips available. Check the soundboard plugin and library folder.");
    return;
  }
  const grid = el("div", ui.clipGrid);
  root.append(grid);
  for (const initial of clips) {
    // A mouse or touch hold of 500 ms loops the clip and swallows the pointer
    // click that follows; a shorter click or Enter (detail 0) plays it once.
    let holdTimer = 0,
      held = false;
    const b = button(
        "",
        (e) => {
          const swallow = held && e.detail > 0;
          held = false;
          if (!swallow) press(initial.ID);
        },
        ui.clip,
      ),
      art = el("div", ui.clipArt),
      name = el("strong", "text-[13px] break-words", initial.Label),
      status = el("small", "mt-[5px] text-[11px] text-muted");
    b.dataset.part = "clip";
    b.append(art, name, status);
    b.title = initial.Label;
    grid.append(b);
    b.onpointerdown = (e) => {
      if (e.button !== 0) return;
      clearTimeout(holdTimer);
      held = false;
      holdTimer = window.setTimeout(() => {
        held = true;
        request(c(initial.ID), "hold");
      }, 500);
    };
    b.onpointerup = b.onpointerleave = b.onpointercancel = () => clearTimeout(holdTimer);
    b.oncontextmenu = (e) => e.preventDefault();
    updaters.push(() => {
      const item = c(initial.ID);
      if (!item) return;
      b.disabled = !!appState.pending || !item.Available;
      b.dataset.state = clipState(item, "Playing");
      status.textContent = item.Status || display(item);
      status.className = "mt-[5px] text-[11px] " + (tc(tone(item)) || "text-muted");
      setArt(art, item.Artwork, () => icon("play", "size-9"), "size-16", item.ID);
      b.hidden = !item.Label.toLocaleLowerCase().includes(search.value.toLocaleLowerCase());
    });
  }
  search.oninput = update;
}
