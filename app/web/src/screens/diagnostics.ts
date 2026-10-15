import { ui, el, button, panel, withIcon, tc } from "../dom.ts";
import { appState, root, updaters, c } from "../state.ts";
import { send } from "../host.ts";
import { control, command } from "../widgets.ts";
import { tone } from "../model.ts";
import { copyText } from "../clipboard.ts";

function currentNotices() {
  const groups = new Map<string, { label: string; status: string; tone: string }[]>();
  let normalOverride = false;
  for (const item of appState.controls.values()) {
    if (!item.Status || item.Hidden || item.ID === "core.startup") continue;
    if (item.ID === "aec.timing" && /^Peak worker \/ queue · gaps \d+ · underruns \d+$/.test(item.Status)) continue;
    if (
      item.ID === "aec.status" &&
      item.Value === "Active" &&
      item.Status.split(" · ").every((part) =>
        /^(16 kHz mono|48 kHz mono|hybrid|~\d+ ms processing latency|−?\d+(?:\.\d+)? dB echo|\d+ ms)$/.test(part),
      )
    )
      continue;
    if (["audio.pause-devices", "audio.pause-sends"].includes(item.ID) && /^\d+ held$/.test(item.Status)) continue;
    let label = item.Label;
    if (
      item.Status === "VR override" &&
      (item.ID.startsWith("audio.normal-") ||
        ["audio.source", "audio.mode", "audio.monitor", "audio.output"].includes(item.ID))
    ) {
      if (normalOverride) continue;
      normalOverride = true;
      label = "Normal profile";
    }
    const plugin = item.ID.split(".")[0] || item.ID;
    const notices = groups.get(plugin) || [];
    notices.push({ label, status: item.Status, tone: tone(item) });
    groups.set(plugin, notices);
  }
  return [...groups].map(([plugin, notices]) => ({ name: appState.state.PluginLabels?.[plugin] || plugin, notices }));
}

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
  control("audio.health", card, "Engine");
  const engine = el("div", ui.actions + " mt-3");
  card.append(engine);
  command("audio.engine-restart", "Restart Audio Engine", engine, "", "refresh-cw");
  command("audio.engine-confirm", "Confirm Restart", engine, ui.danger);
  const details = panel("Current Notices", root, "mt-[18px] text-[13px] break-words"),
    actions = el("div", ui.actions + " mb-3"),
    feedback = el("span", "text-xs text-muted"),
    list = el("div", "whitespace-pre-wrap");
  feedback.setAttribute("role", "status");
  const copy = withIcon(
    button(
      "Copy Notices",
      () => {
        const text = currentNotices()
          .map((group) => group.name + "\n" + group.notices.map((item) => item.label + ": " + item.status).join("\n"))
          .join("\n\n");
        void copyText(text, feedback);
      },
      ui.small,
    ),
    "copy",
  );
  actions.append(copy, feedback);
  details.append(actions, list);
  let signature = "";
  updaters.push(() => {
    const groups = currentNotices(),
      next = JSON.stringify(groups);
    copy.disabled = !groups.length;
    if (signature === next) return;
    signature = next;
    list.replaceChildren();
    if (!groups.length) list.append(el("p", "my-2 text-muted", "No control notices."));
    for (const group of groups) {
      const section = el("div", "mt-4 first:mt-0");
      section.append(el("h3", ui.h3, group.name));
      for (const item of group.notices) {
        const notice = el("div", "my-2");
        notice.append(el("strong", "font-medium", item.label), el("p", tc(item.tone) || "text-muted", item.status));
        section.append(notice);
      }
      list.append(section);
    }
  });
}
