import { ui, el, button, panel, withIcon, tc } from "../dom.ts";
import { appState, root, updaters, c } from "../state.ts";
import { send } from "../host.ts";
import { control, command } from "../widgets.ts";
import { tone } from "../model.ts";

export function buildDiagnostics() {
  const toolbar = el("div", ui.toolbar);
  toolbar.append(
    withIcon(
      button("Retry Plugins", () => send({ Kind: "retry" })),
      "refresh-cw",
    ),
  );
  root.append(toolbar);
  if (c("core.startup")) {
    const startup = panel("Startup", root, "mb-[18px]");
    control("core.startup", startup);
  }
  const card = panel("Audio Engine", root);
  control("audio.health", card);
  control("audio.interface", card);
  control("audio.asio-unavailable", card);
  control("audio.auto-recover", card, "Automatic Recovery");
  const engine = el("div", ui.actions + " mt-3");
  card.append(engine);
  command("audio.engine-restart", "Restart Audio Engine", engine, "", "refresh-cw");
  command("audio.engine-confirm", "Confirm Restart", engine, ui.danger);
  const details = el("section", ui.panel + " mt-[18px] text-[13px] break-words whitespace-pre-wrap");
  root.append(details);
  updaters.push(() => {
    details.replaceChildren(el("h2", ui.h2, "Current Notices"));
    const notices = [...appState.controls.values()].filter((v) => v.Status);
    if (!notices.length) details.append(el("p", "my-2 text-muted", "No control notices."));
    for (const item of notices) {
      details.append(
        el("h3", ui.h3 + " mt-3", item.Label),
        el("p", "my-2 " + (tc(tone(item)) || "text-muted"), item.Status),
      );
    }
  });
}
