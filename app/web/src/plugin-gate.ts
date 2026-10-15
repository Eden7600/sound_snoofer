import { button, el, panel, ui } from "./dom.ts";
import { appState, root, updaters } from "./state.ts";
import { send } from "./host.ts";
import { viewKey } from "./view-state.ts";

import { gateState, lifecycleStates, pluginScreens } from "./plugin-lifecycle.ts";

export function pluginStates() {
  return lifecycleStates(appState.state);
}
export function pluginGateState(screen: string) {
  return gateState(appState.state, screen);
}

// Selection, retry and confirmation share one lifecycle action group.
export function lifecycleUnavailable() {
  return appState.reconnecting || !!appState.pending;
}

export function buildPluginGate(screen: string) {
  const gate = pluginScreens[screen],
    state = pluginGateState(screen);
  if (!gate || !state || state === "running") return false;
  const id = gate.plugin,
    label = appState.state.PluginLabels?.[id] || id;
  const card = panel(label, root),
    message = el("p", state === "failed" ? "text-critical" : "text-muted");
  card.dataset.part = "plugin-gate";
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
    const enable = viewKey(
      button(
        "Enable " + label,
        () => {
          if (!lifecycleUnavailable()) send({ Kind: "selection", Plugin: id, Enable: true });
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
