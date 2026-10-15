import { button, el, panel, ui } from "./dom.ts";
import { appState, root, updaters } from "./state.ts";
import { send } from "./host.ts";
import { isBusy } from "./actions.ts";
import { viewKey } from "./view-state.ts";

import { gateState, lifecycleStates, pluginScreens, providerGateState, type PluginGate } from "./plugin-lifecycle.ts";

export function pluginStates() {
  return lifecycleStates(appState.state);
}
export function pluginGateState(screen: string) {
  return gateState(appState.state, screen);
}

export function pluginLifecycleKey(screen: string) {
  return JSON.stringify(pluginScreens[screen]?.map((gate) => providerGateState(appState.state, gate)));
}

// Selection, retry and confirmation share one lifecycle action group.
export function lifecycleUnavailable() {
  return appState.reconnecting || isBusy("lifecycle");
}

export function buildPluginGate(screen: string) {
  const gates = pluginScreens[screen];
  return gates?.length === 1 ? buildProviderGate(gates[0]!, root) : false;
}

export function buildProviderGate(gate: PluginGate, parent: HTMLElement) {
  const state = providerGateState(appState.state, gate);
  if (state === "running") return false;
  const id = gate.plugin,
    label = appState.state.PluginLabels?.[id] || id;
  const card = panel(label, parent),
    message = el("p", state === "failed" ? "text-critical" : "text-muted");
  card.dataset.part = "plugin-gate";
  card.dataset.plugin = id;
  message.textContent = {
    missing: "Not part of this build",
    disabled: "Off",
    starting: "Starting",
    failed: "Could not start",
  }[state];
  card.append(message);
  if (state === "failed") {
    const detail = el("p", "mt-2 break-words text-sm text-critical");
    card.append(detail);
    updaters.push(() => {
      detail.textContent = appState.state.Plugins?.[id] || "";
    });
  }
  if (state === "disabled") {
    const configRevision = appState.state.ConfigRevision;
    const enable = viewKey(
      button(
        "Enable " + label,
        () => {
          if (!lifecycleUnavailable())
            send({ Kind: "selection", Plugin: id, Enable: true, ConfigRevision: configRevision });
        },
        ui.primary + " mt-4",
      ),
      "plugin-gate",
      id,
      "enable",
    );
    card.append(enable);
    updaters.push(() => {
      enable.disabled = lifecycleUnavailable();
    });
  } else if (state !== "missing") {
    card.append(
      viewKey(
        button(
          "Plugins",
          () => {
            const nav = document.querySelector<HTMLButtonElement>('[data-screen="plugins"]');
            nav?.focus();
            nav?.click();
          },
          "mt-4",
        ),
        "plugin-gate",
        id,
        "plugins",
      ),
    );
  }
  return true;
}

export function pluginStateText(id: string) {
  const state = pluginStates()[id];
  if (state === "failed")
    return "Could not start" + (appState.state.Plugins?.[id] ? " — " + appState.state.Plugins?.[id] : "");
  return state === "running"
    ? "Running"
    : state === "disabled"
      ? "Off"
      : state === "starting"
        ? "Starting"
        : "Not part of this build";
}
