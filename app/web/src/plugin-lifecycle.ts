import type { PluginState, ViewState } from "./types.ts";

// Each screen waits for the publication it actually needs.
export interface PluginGate {
  plugin: string;
  publication: string;
  view?: true;
}
export const pluginScreens: Record<string, PluginGate[]> = {
  audio: [{ plugin: "audio", publication: "audio.health" }],
  routing: [{ plugin: "audio", publication: "audio.priorities", view: true }],
  soundboard: [{ plugin: "soundboard", publication: "soundboard.status" }],
  lights: [{ plugin: "hue", publication: "hue.status" }],
  appaudio: [{ plugin: "appaudio", publication: "appaudio.status", view: true }],
  meetings: [
    { plugin: "discord", publication: "discord.channel" },
    { plugin: "googlemeet", publication: "googlemeet.call" },
  ],
  media: [{ plugin: "nowplaying", publication: "nowplaying.status", view: true }],
  deck: [{ plugin: "streamdeck", publication: "streamdeck.preview", view: true }],
};

function published(state: ViewState, gate: PluginGate) {
  const control = state.Controls?.find((control) => control.ID === gate.publication);
  return !!control && (!gate.view || !!control.ViewData);
}

export function lifecycleStates(state: ViewState): Record<string, PluginState> {
  if (state.PluginStates != null) return state.PluginStates;
  // Only legacy unversioned fixtures omit the whole map. Never parse their prose.
  const states: Record<string, PluginState> = {};
  for (const id of new Set([...Object.keys(state.Plugins || {}), ...Object.keys(state.Enabled || {})]))
    states[id] = state.Plugins && id in state.Plugins ? (state.Enabled?.[id] ? "starting" : "disabled") : "missing";
  for (const gate of Object.values(pluginScreens).flat()) if (published(state, gate)) states[gate.plugin] = "running";
  return states;
}

export function gateState(state: ViewState, screen: string) {
  const gates = pluginScreens[screen];
  if (gates?.length !== 1) return;
  return providerGateState(state, gates[0]!);
}

export function providerGateState(state: ViewState, gate: PluginGate) {
  const lifecycle = lifecycleStates(state)[gate.plugin] || "missing";
  return lifecycle === "running" && !published(state, gate) ? "starting" : lifecycle;
}
