import type { PluginState, ViewState } from "./types.ts";

// Each screen waits for the publication it actually needs.
export const pluginScreens: Record<string, { plugin: string; publication: string; view?: true }> = {
  audio: { plugin: "audio", publication: "audio.health" },
  routing: { plugin: "audio", publication: "audio.priorities", view: true },
  soundboard: { plugin: "soundboard", publication: "soundboard.status" },
  lights: { plugin: "hue", publication: "hue.status" },
  appaudio: { plugin: "appaudio", publication: "appaudio.status", view: true },
  meetings: { plugin: "discord", publication: "discord.channel" },
  media: { plugin: "nowplaying", publication: "nowplaying.status", view: true },
  deck: { plugin: "streamdeck", publication: "streamdeck.preview", view: true },
};

function published(state: ViewState, gate: (typeof pluginScreens)[string]) {
  const control = state.Controls?.find((control) => control.ID === gate.publication);
  return !!control && (!gate.view || !!control.ViewData);
}

export function lifecycleStates(state: ViewState): Record<string, PluginState> {
  if (state.PluginStates != null) return state.PluginStates;
  // Only legacy unversioned fixtures omit the whole map. Never parse their prose.
  const states: Record<string, PluginState> = {};
  for (const id of new Set([...Object.keys(state.Plugins || {}), ...Object.keys(state.Enabled || {})]))
    states[id] = state.Plugins && id in state.Plugins ? (state.Enabled?.[id] ? "starting" : "disabled") : "missing";
  for (const gate of Object.values(pluginScreens)) if (published(state, gate)) states[gate.plugin] = "running";
  return states;
}

export function gateState(state: ViewState, screen: string) {
  const gate = pluginScreens[screen];
  if (!gate) return;
  const lifecycle = lifecycleStates(state)[gate.plugin] || "missing";
  return lifecycle === "running" && !published(state, gate) ? "starting" : lifecycle;
}
