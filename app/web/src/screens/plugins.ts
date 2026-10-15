import { ui, el, button } from "../dom.ts";
import { appState, root, updaters } from "../state.ts";
import { send } from "../host.ts";
import { viewKey } from "../view-state.ts";
import { lifecycleUnavailable, pluginStateText, pluginStates } from "../plugin-gate.ts";

export function buildPlugins() {
  const configRevision = appState.state.ConfigRevision;
  const retry = viewKey(
    button("Retry Plugins", () => send({ Kind: "retry" })),
    "plugins",
    "retry",
  );
  const bar = el("div", ui.toolbar);
  bar.append(retry);
  root.append(bar);
  updaters.push(() => {
    retry.disabled = lifecycleUnavailable() || !Object.values(pluginStates()).includes("failed");
  });
  for (const id of Object.keys(pluginStates()).sort()) {
    const card = el("section", ui.panel + " mb-3 flex items-center justify-between gap-5"),
      text = el("div", "min-w-0"),
      status = el("p", "mt-[5px] text-xs break-words");
    card.dataset.part = "plugin";
    text.append(el("h2", ui.h2, appState.state.PluginLabels?.[id] || id), status);
    card.append(text);
    root.append(card);
    const state = pluginStates()[id];
    if (state !== "missing") {
      const toggle = viewKey(
        button(
          state === "disabled" ? "Enable" : "Disable",
          () =>
            send({
              Kind: "selection",
              Plugin: id,
              Enable: state === "disabled",
              ConfigRevision: configRevision,
            }),
          ui.toggle,
        ),
        "plugin",
        id,
        "toggle",
      );
      card.append(toggle);
      updaters.push(() => {
        toggle.disabled = lifecycleUnavailable();
      });
    }
    updaters.push(() => {
      status.textContent = pluginStateText(id);
      status.className =
        "mt-[5px] text-xs break-words " +
        (state === "running" ? "text-active" : state === "failed" ? "text-critical" : "text-muted");
    });
  }
}
