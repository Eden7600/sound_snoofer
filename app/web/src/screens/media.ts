import { isBusy, actionFeedback } from "../actions.ts";
import { ui, el, button, panel, empty, iconButton, icon } from "../dom.ts";
import { root, updaters, c, appState } from "../state.ts";
import { request, press } from "../host.ts";
import { setArt } from "../widgets.ts";
import { progressAt } from "../model.ts";
import type { MediaView, MediaSession } from "./media-types.ts";
import { viewKey } from "../view-state.ts";

// mediaCard shows one session: artwork, title, source, progress with a seek
// slider and play/pause, which also focuses it; the focused card adds Previous,
// Next and Mute.
function mediaCard(id: string, parent: HTMLElement) {
  const info = (): Partial<MediaSession> =>
    (c<MediaView>("nowplaying.status")?.ViewData?.Sessions || []).find((s) => s.ID === id) || {};
  const card = el("article", ui.strip + " flex flex-col gap-2 data-[focused=true]:border-active"),
    head = el("div", "flex items-center gap-3"),
    art = el("div", "grid size-16 shrink-0 place-items-center text-active"),
    text = el("div", "min-w-0 flex-1");
  const title = el("h3", "truncate text-sm font-semibold"),
    artist = el("div", "truncate text-[13px]"),
    source = el("div", "truncate text-[11px] text-muted");
  card.dataset.part = "media-session";
  text.append(title, artist, source);
  head.append(art, text);
  const seek = el("input", "w-full accent-active");
  seek.type = "range";
  seek.min = "0";
  seek.step = "1000";
  seek.dataset.part = "seek";
  viewKey(seek, id, "seek");
  const times = el("div", "flex justify-between text-[11px] text-muted tabular-nums"),
    elapsed = el("span"),
    length = el("span");
  times.append(elapsed, length);
  // While dragged the slider keeps the pointer's position; afterwards it follows
  // the session again (focus alone would freeze it after one drag).
  let dragging = false;
  seek.onpointerdown = () => {
    dragging = true;
  };
  seek.onpointerup = seek.onpointercancel = () => {
    dragging = false;
  };
  seek.onchange = () => {
    dragging = false;
    request(c(id), "set", seek.value);
  };
  const actions = el("div", ui.actions),
    status = el("small", "text-[11px]");
  const prev = iconButton("skip-back", "Previous", () => press("nowplaying.prev", id)),
    play = iconButton("play", "Play or Pause", () => press(id)),
    next = iconButton("skip-forward", "Next", () => press("nowplaying.next", id));
  const mute = button("Mute", () => press("nowplaying.mute", id), ui.toggle + " " + ui.small);
  play.dataset.part = "play";
  mute.dataset.part = "mute";
  viewKey(prev, id, "previous");
  viewKey(play, id, "play");
  viewKey(next, id, "next");
  viewKey(mute, id, "mute");
  actions.append(prev, play, next, mute, status);
  card.append(head, seek, times, actions);
  parent.append(card);
  actionFeedback(id, card);
  for (const key of ["nowplaying.prev", "nowplaying.next", "nowplaying.mute"]) actionFeedback(key, card, id);
  const clock = (ms?: number) => {
    const s = Math.max(0, Math.floor((ms || 0) / 1000));
    return s >= 3600
      ? Math.floor(s / 3600) +
          ":" +
          String(Math.floor(s / 60) % 60).padStart(2, "0") +
          ":" +
          String(s % 60).padStart(2, "0")
      : Math.floor(s / 60) + ":" + String(s % 60).padStart(2, "0");
  };
  updaters.push(() => {
    const item = c(id),
      s = info();
    if (!item) return;
    setArt(art, item.Artwork, () => icon("disc-3", "size-9"), "size-16");
    title.textContent = s.Title || s.App || item.Label;
    artist.textContent = [s.Artist, s.Album].filter(Boolean).join(" · ");
    artist.hidden = !artist.textContent;
    source.textContent = s.App || "";
    card.dataset.focused = String(!!s.Focused);
    play.replaceChildren(icon(s.Status === "Playing" ? "pause" : "play", "size-4 shrink-0"));
    play.setAttribute("aria-label", (s.Status === "Playing" ? "Pause " : "Play ") + (s.Title || s.App || ""));
    seek.hidden = times.hidden = !s.DurationMs;
    seek.max = String(s.DurationMs || 0);
    seek.setAttribute("aria-label", "Seek " + (s.Title || s.App || ""));
    const position = dragging ? Number(seek.value) : progressAt(s);
    if (!dragging) seek.value = String(position);
    elapsed.textContent = clock(position);
    length.textContent = clock(s.DurationMs);
    prev.hidden = !s.Focused || !s.CanPrev;
    next.hidden = !s.Focused || !s.CanNext;
    mute.hidden = !s.Focused || !s.CanMute;
    mute.textContent = s.Muted ? "Muted" : "Mute";
    mute.setAttribute("aria-pressed", String(!!s.Muted));
    mute.dataset.tone = s.Muted ? "critical" : "";
    status.textContent = s.Pending ? "Wait" : s.Failure || "";
    status.className = "text-[11px] " + (s.Failure ? "text-critical" : "text-attention");
    seek.disabled = isBusy(id) || !item.Available || !s.CanSeek;
    play.disabled = isBusy(id) || !item.Available || !s.CanToggle;
    prev.disabled = isBusy("nowplaying.prev") || !c("nowplaying.prev")?.Available;
    next.disabled = isBusy("nowplaying.next") || !c("nowplaying.next")?.Available;
    mute.disabled = isBusy("nowplaying.mute") || !c("nowplaying.mute")?.Available;
  });
}
// extensionCard explains the Snoofer Media extension, which reports each
// browser tab, and shows which browsers are connected.
function extensionCard(parent: HTMLElement) {
  const card = panel("Browser Extension", parent, "mt-[18px]");
  card.dataset.part = "extension";
  const intro = el(
    "p",
    "-mt-2 text-[13px] text-muted",
    "Browsers show Windows only one session for all their tabs. The Snoofer Media extension reports each tab separately, so every video or song can be paused and sought on its own. It connects by itself; there is nothing to pair.",
  );
  const browsers = el("div", "mt-3 grid gap-1 text-[13px]"),
    problem = el("p", "mt-2 text-[13px] text-critical");
  const steps = el("ol", "mt-3 list-decimal space-y-1 pl-5 text-[13px] text-muted");
  for (const step of [
    "Install Snoofer Media from the Chrome Web Store (Chrome, Brave, Edge) or Firefox Add-ons.",
    "In Firefox, click the extension's toolbar button and choose Allow on all sites.",
    "Play something in a tab; it appears above.",
  ])
    steps.append(el("li", "", step));
  const dev = el(
    "p",
    "mt-2 text-[12px] text-muted",
    "Developing: run npm run build in the repository's extension folder, then load extension/dist/chrome unpacked (Developer mode) or extension/dist/firefox as a temporary add-on.",
  );
  card.append(intro, browsers, problem, steps, dev);
  updaters.push(() => {
    const v: Partial<MediaView> = c<MediaView>("nowplaying.status")?.ViewData || {};
    browsers.replaceChildren();
    for (const b of v.Browsers || [])
      browsers.append(
        el(
          "div",
          "text-active",
          b.Name + " connected · " + b.Sessions + " with media" + (b.Version ? " · extension " + b.Version : ""),
        ),
      );
    if (!(v.Browsers || []).length) browsers.append(el("div", "text-muted", "No browser connected."));
    problem.textContent = [
      v.Bridge?.Error,
      v.Bridge?.Refused,
      v.WindowsError ? "Windows media sessions: " + v.WindowsError : "",
    ]
      .filter(Boolean)
      .join(" · ");
    problem.hidden = !problem.textContent;
  });
}
export function buildMedia() {
  const sessions = [...appState.controls.values()]
    .filter((v) => v.Collection === "nowplaying.sessions")
    .sort((a, b) => (a.Order || 0) - (b.Order || 0));
  if (!sessions.length) empty(root, "Nothing is playing. Start music or a video in any app or browser tab.");
  else {
    const grid = el("div", "grid grid-cols-[repeat(auto-fill,minmax(300px,1fr))] gap-[18px]");
    root.append(grid);
    for (const item of sessions) mediaCard(item.ID, grid);
  }
  extensionCard(root);
}
