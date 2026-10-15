import { ui, tc, el, panel } from "../dom.ts";
import { root, updaters, c } from "../state.ts";
import { command, control } from "../widgets.ts";
import { display, tone } from "../model.ts";

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
  const card = panel("Discord", grid);
  card.dataset.part = "discord";
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
