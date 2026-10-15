import { ui, el, button } from "../dom.ts";
import { appState, root, updaters } from "../state.ts";
import { send } from "../host.ts";

export function buildPlugins() {
  for (const id of Object.keys(appState.state.Plugins || {}).sort()) {
    const card = el("section", ui.panel + " mb-3 flex items-center justify-between gap-5"),
      text = el("div"),
      status = el("p", "mt-[5px] text-xs break-words"),
      toggle = button(
        "",
        () =>
          send({
            Kind: "selection",
            Plugin: id,
            Enable: !appState.state.Enabled?.[id],
          }),
        ui.toggle,
      );
    card.dataset.part = "plugin";
    text.append(el("h2", ui.h2, id), status);
    card.append(text, toggle);
    root.append(card);
    updaters.push(() => {
      status.textContent = appState.state.Plugins?.[id] || "";
      status.className =
        "mt-[5px] text-xs break-words " +
        (appState.state.Plugins?.[id] === "Running"
          ? "text-active"
          : appState.state.Plugins?.[id] === "Disabled"
            ? "text-muted"
            : "text-critical");
      toggle.textContent = appState.state.Enabled?.[id] ? "Disable" : "Enable";
      toggle.disabled = !!appState.pending;
    });
  }
}
