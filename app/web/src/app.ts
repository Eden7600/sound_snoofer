import { controlsByID, gridMove } from "./model.ts";
import { $, ui, icon } from "./dom.ts";
import { appState, root, widgets, updaters, c, update, showError, setReconnecting } from "./state.ts";
import { viewVersion, hydrateState } from "./host.ts";
import { initializeConfirmation, closeLocalConfirmation } from "./confirmation.ts";
import { captureView, restoreView } from "./view-state.ts";
import type { ViewState, ViewUpdate } from "./types.ts";
import type { MediaView } from "./screens/media-types.ts";
import type { AppAudioView } from "./screens/appaudio-types.ts";
import type { DeckView } from "./screens/deck-types.ts";
import { buildAudio } from "./screens/audio.ts";
import { buildRouting } from "./screens/routing.ts";
import { buildSoundboard } from "./screens/soundboard.ts";
import { buildLights } from "./screens/lights.ts";
import { buildAppAudio } from "./screens/appaudio.ts";
import { buildMeetings } from "./screens/meetings.ts";
import { buildMedia } from "./screens/media.ts";
import { buildDeck, clearDeckRange, deckGeometry } from "./screens/deck.ts";
import { buildPlugins } from "./screens/plugins.ts";
import { buildApps } from "./screens/apps.ts";
import { buildDiagnostics } from "./screens/diagnostics.ts";
import { buildPluginGate, pluginGateState, pluginStates } from "./plugin-gate.ts";
let signature = "";
let lifecycle: ReturnType<typeof pluginGateState>;
const titles: Record<string, [string, string]> = {
  audio: ["AUDIO", "Audio"],
  routing: ["AUDIO", "Routing"],
  soundboard: ["LIBRARY", "Soundboard"],
  lights: ["LIGHTING", "Lights"],
  appaudio: ["MIXER", "App Audio"],
  meetings: ["CALLS", "Meetings"],
  media: ["NOW PLAYING", "Media"],
  deck: ["CONTROL SURFACE", "Stream Deck"],
  plugins: ["SYSTEM", "Plugins"],
  apps: ["SYSTEM", "Third-Party Apps"],
  diagnostics: ["SYSTEM", "Diagnostics"],
};

function layoutKey() {
  // Values and telemetry are updated in place. Only structure/context rebuilds a screen.
  // A configuration change rebuilds too, so drawnConfig always names the drawn lists.
  const list: unknown[] = [...appState.controls.values()].map((v) => [
    v.ID,
    v.Label,
    v.Kind,
    v.Group,
    v.Options,
    v.OptionLabels,
  ]);
  list.push(c("audio.priority-edit")?.Value);
  return JSON.stringify([
    appState.screen,
    pluginGateState(appState.screen),
    appState.screen === "plugins" || pluginGateState(appState.screen) === "disabled"
      ? appState.state.ConfigRevision
      : null,
    appState.screen === "plugins" || appState.screen === "apps" ? pluginStates() : null,
    list,
    Object.keys(appState.state.Plugins || {}),
    appState.state.PluginLabels,
    appState.screen === "lights"
      ? [
          appState.state.Enabled?.hue,
          c("hue.group")?.Value,
          c("hue.rooms")?.ViewData,
          [...appState.controls.values()].filter((v) => v.ID.startsWith("hue.scene-")).map((v) => v.Order),
        ]
      : null,
    appState.screen === "meetings" ? [appState.state.Enabled?.discord] : null,
    appState.screen === "media"
      ? [
          appState.state.Enabled?.nowplaying,
          (c<MediaView>("nowplaying.status")?.ViewData?.Sessions || []).map((s) => [s.ID, s.CanSeek, s.CanMute]),
          (c<MediaView>("nowplaying.status")?.ViewData?.Browsers || []).map((b) => b.Name),
        ]
      : null,
    appState.screen === "appaudio"
      ? [
          appState.state.Enabled?.appaudio,
          (c<AppAudioView>("appaudio.status")?.ViewData?.Apps || []).map((a) => [
            a.ID,
            a.Name,
            a.Hidden,
            a.Picked,
            a.Open,
            a.Rule,
            a.Naming,
          ]),
          c<AppAudioView>("appaudio.status")?.ViewData?.Exclude,
        ]
      : null,
    appState.screen === "routing"
      ? [
          c("audio.priorities")?.ViewData,
          [1, 2, 3].map((n) => [
            c("audio.slot-" + n)?.Hidden,
            c("audio.slot-" + n)?.Status,
            c("audio.slot-" + n)?.ViewData,
          ]),
        ]
      : null,
    appState.screen === "deck"
      ? [
          deckGeometry(),
          c<DeckView>("streamdeck.preview")?.ViewData?.Selected,
          c("streamdeck.page")?.Value,
          c("streamdeck.profile")?.Value,
          c("streamdeck.shared")?.Value,
          c<DeckView>("streamdeck.preview")?.ViewData?.Regions,
          c<DeckView>("streamdeck.preview")?.ViewData?.Collections,
          c("streamdeck.deck")?.Options,
          c("streamdeck.deck")?.Value,
        ]
      : null,
  ]);
}
function build() {
  widgets.length = 0;
  updaters.length = 0;
  root.replaceChildren();
  for (const b of document.querySelectorAll<HTMLButtonElement>("[data-screen]")) {
    if (b.dataset.screen === appState.screen) b.setAttribute("aria-current", "page");
    else b.removeAttribute("aria-current");
  }
  const title = titles[appState.screen];
  if (!title) throw new Error("Unknown screen: " + appState.screen);
  $("#eyebrow").textContent = title[0];
  $("#title").textContent = title[1];
  const builders: Record<string, () => void> = {
    audio: buildAudio,
    routing: buildRouting,
    soundboard: buildSoundboard,
    lights: buildLights,
    appaudio: buildAppAudio,
    meetings: buildMeetings,
    media: buildMedia,
    deck: buildDeck,
    plugins: buildPlugins,
    apps: buildApps,
    diagnostics: buildDiagnostics,
  };
  if (!buildPluginGate(appState.screen)) builders[appState.screen]?.();
}
function receive(next: ViewState | null) {
  appState.state = next || appState.state;
  appState.controls = controlsByID(appState.state);
  const nextKey = layoutKey(),
    nextLifecycle = pluginGateState(appState.screen);
  const lifecycleChanged = lifecycle !== nextLifecycle;
  if (nextKey !== signature) {
    // Preserve edits until blur if a provider changes its available choices mid-edit.
    const focused = document.activeElement;
    if (!lifecycleChanged && focused && root.contains(focused) && ["INPUT", "SELECT"].includes(focused.tagName)) {
      update();
      return;
    }
    const saved = signature ? captureView(root) : undefined;
    if (lifecycleChanged) saved?.nodes.clear();
    lifecycle = nextLifecycle;
    signature = nextKey;
    build();
    update();
    if (saved) restoreView(root, saved);
    return;
  }
  update();
}
const navIcons: Record<string, string> = {
  audio: "audio-waveform",
  routing: "route",
  soundboard: "music",
  lights: "lightbulb",
  appaudio: "app-window",
  meetings: "video",
  media: "disc-3",
  deck: "layout-grid",
  plugins: "puzzle",
  apps: "plug",
  diagnostics: "activity",
};
for (const b of document.querySelectorAll<HTMLButtonElement>("[data-screen]")) {
  const target = b.dataset.screen;
  if (!target || !titles[target]) throw new Error("Unknown navigation screen");
  b.className = ui.nav;
  b.prepend(icon(navIcons[target] || "circle", "size-5 shrink-0"));
  b.onclick = () => {
    closeLocalConfirmation();
    clearDeckRange();
    appState.screen = target;
    signature = "";
    receive(appState.state);
  };
  b.onkeydown = (e) => {
    if (!["ArrowUp", "ArrowDown", "Home", "End"].includes(e.key)) return;
    e.preventDefault();
    const items = [...document.querySelectorAll<HTMLButtonElement>("[data-screen]")];
    const i = items.indexOf(b);
    const next = e.key === "Home" ? 0 : e.key === "End" ? items.length - 1 : gridMove(i, e.key, 1, items.length);
    items[next]?.focus();
    items[next]?.click();
  };
}
initializeConfirmation();
let connectionFailures = 0;
async function poll() {
  try {
    let next: ViewUpdate | null;
    try {
      if (!window.go?.app?.Desktop) throw new Error("Desktop connection unavailable");
      next = await window.go.app.Desktop.State(viewVersion);
    } catch {
      connectionFailures = Math.min(connectionFailures + 1, 6);
      setReconnecting(true);
      update();
      return;
    }
    connectionFailures = 0;
    setReconnecting(false);
    receive(hydrateState(next));
  } catch (error) {
    showError(error);
  } finally {
    // A render failure must not stop polling; disconnected polls back off.
    const cadence = ["audio", "soundboard", "appaudio", "media"].includes(appState.screen) ? 100 : 200;
    setTimeout(poll, connectionFailures ? Math.min(5000, 100 * 2 ** connectionFailures) : cadence);
  }
}
poll();
