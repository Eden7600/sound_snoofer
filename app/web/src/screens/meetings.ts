import { ui, tc, el, button, panel } from "../dom.ts";
import { root, updaters, c, appState } from "../state.ts";
import { send } from "../host.ts";
import { command, control } from "../widgets.ts";
import { display, tone } from "../model.ts";

// pluginOff renders the card for a plugin that is missing or disabled, and
// reports whether it did.
function pluginOff(id: string, title: string, text: string, parent: HTMLElement) {
  if (!(appState.state.Plugins || {})[id]) {
    const card = panel(title, parent);
    card.append(el("p", "text-muted", title + " is not part of this build."));
    return true;
  }
  if (appState.state.Enabled?.[id]) return false;
  const card = panel(title + " is off", parent, "border-[#345365]");
  card.dataset.part = "setup";
  card.append(el("p", "mt-2 mb-3.5 text-muted", text));
  const actions = el("div", ui.actions);
  actions.append(button("Enable " + title, () => send({ Kind: "selection", Plugin: id, Enable: true }), ui.primary));
  card.append(actions);
  return true;
}
// stateLine shows a status control's value under a card title, toned by its appState.state.
function stateLine(id: string, parent: HTMLElement) {
  const line = el("p", "-mt-1.5 mb-3 text-[13px]");
  line.dataset.part = "state";
  parent.append(line);
  updaters.push(() => {
    const item = c(id);
    line.textContent = item ? display(item) : "";
    line.className = "-mt-1.5 mb-3 text-[13px] " + (item ? tc(tone(item)) : "");
  });
}
export function buildMeetings() {
  const grid = el("div", ui.grid);
  grid.dataset.part = "meetings";
  root.append(grid);
  if (!pluginOff("discord", "Discord", "Discord mute, deafen, camera and screen share.", grid)) {
    const card = panel("Discord", grid);
    card.dataset.part = "discord";
    if (!c("discord.channel"))
      card.append(el("p", "text-muted", "Discord: " + (appState.state.Plugins?.discord || "Starting")));
    else {
      stateLine("discord.channel", card);
      const setup = el(
        "p",
        "mb-3 text-attention",
        "Add your Discord application's client_id and client_secret to snoofer.json.",
      );
      setup.dataset.part = "discord-setup";
      card.append(setup);
      const connect = el("div", ui.actions + " mb-3");
      connect.dataset.part = "discord-connect";
      command("discord.connect", "Connect", connect, ui.primary);
      card.append(connect);
      updaters.push(() => {
        setup.hidden = c("discord.mute")?.Status !== "Setup needed";
        connect.hidden = !c("discord.connect")?.Available && c("discord.connect")?.Status !== "Pending";
      });
      control("discord.mute", card, "Mute");
      control("discord.deafen", card, "Deafen");
      control("discord.video", card, "Camera");
      control("discord.screenshare", card, "Screen Share");
      control("discord.leave", card, "Leave Call");
    }
  }
}
