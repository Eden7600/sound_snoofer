import type { ViewState } from "./types.ts";
import type { PriorityView } from "./screens/routing-types.ts";
import { connectionTone, stale } from "./model.ts";
import { lifecycleStates, pluginScreens, providerGateState } from "./plugin-lifecycle.ts";
import { icon } from "./dom.ts";

type Severity = "attention" | "critical";
const sources: Record<string, [plugin: string, screen: string]> = {
  "audio.health": ["audio", "audio"],
  "aec.status": ["aec", "audio"],
  "audio.slot-1": ["audio", "routing"],
  "audio.slot-2": ["audio", "routing"],
  "audio.slot-3": ["audio", "routing"],
  "soundboard.status": ["soundboard", "soundboard"],
  "appaudio.status": ["appaudio", "appaudio"],
  "nowplaying.status": ["nowplaying", "media"],
  "streamdeck.status": ["streamdeck", "deck"],
};
const connectionScreens: Record<string, string> = {
  audio: "audio",
  aec: "audio",
  hue: "lights",
  discord: "meetings",
  googlemeet: "meetings",
  soundboard: "soundboard",
};

// Only sources with a current issue or recovery path at the destination count.
export function screenAttention(state: ViewState, now = Date.now()) {
  const attention: Record<string, Severity> = {},
    lifecycle = lifecycleStates(state);
  const mark = (screen: string, severity: Severity) => {
    if (attention[screen] !== "critical") attention[screen] = severity;
  };
  const ready = (screen: string, plugin?: string) =>
    pluginScreens[screen]?.some(
      (gate) => (!plugin || gate.plugin === plugin) && providerGateState(state, gate) === "running",
    );
  for (const [plugin, status] of Object.entries(lifecycle)) {
    if (status !== "failed") continue;
    mark("plugins", "critical");
    for (const [screen, gates] of Object.entries(pluginScreens))
      if (gates.some((gate) => gate.plugin === plugin)) mark(screen, "critical");
  }
  for (const control of state.Controls || []) {
    const source = sources[control.ID];
    if (
      source &&
      !control.Hidden &&
      lifecycle[source[0]] === "running" &&
      ready(source[1]) &&
      (control.Attention === "attention" || control.Attention === "critical")
    )
      mark(source[1], control.Attention);
    const conn = control.Connection;
    if (control.Kind !== "connection" || !conn) continue;
    const plugin = control.ID.split(".")[0] || "";
    if (lifecycle[plugin] && lifecycle[plugin] !== "running") continue;
    const tone = connectionTone(conn);
    let severity: Severity | undefined;
    if (conn.State !== "connecting" && (tone === "attention" || tone === "critical")) severity = tone;
    else if (stale(conn, now)) severity = "attention";
    if (
      conn.State === "unconfigured" &&
      lifecycle[plugin] === "running" &&
      (control.ID === "hue.app-bridge" || control.ID === "discord.app-client")
    )
      severity = "attention";
    if (!severity) continue;
    mark("apps", severity);
    const owner = connectionScreens[plugin];
    if (owner && ready(owner, owner === "meetings" ? plugin : undefined)) mark(owner, severity);
  }
  if (ready("routing")) {
    const view = state.Controls?.find((control) => control.ID === "audio.priorities")?.ViewData as
      PriorityView | undefined;
    if (
      Object.values(view?.Lists || {}).some((list) =>
        list?.some((entry) => !entry.InUse && (entry.Matches?.length || 0) > 1),
      )
    )
      mark("routing", "attention");
  }
  return attention;
}

export function updateNavigationAttention(state: ViewState) {
  const attention = screenAttention(state);
  for (const button of document.querySelectorAll<HTMLButtonElement>("[data-screen]")) {
    const severity = attention[button.dataset.screen || ""];
    const existing = button.querySelector<SVGSVGElement>('[data-part="nav-attention"]');
    const normal = button.querySelector<SVGSVGElement>('svg:not([data-part="nav-attention"])');
    normal?.classList.toggle("hidden", !!severity);
    if (!severity) {
      existing?.remove();
      delete button.dataset.attention;
      button.removeAttribute("aria-description");
      button.removeAttribute("title");
      continue;
    }
    const marker = existing || icon("triangle-alert");
    marker.dataset.part = "nav-attention";
    marker.setAttribute(
      "class",
      severity === "critical" ? "size-5 shrink-0 text-critical" : "size-5 shrink-0 text-attention",
    );
    if (!existing) button.prepend(marker);
    const description = severity === "critical" ? "Problem" : "Needs attention";
    button.dataset.attention = severity;
    button.setAttribute("aria-description", description);
    button.title = description;
  }
}
