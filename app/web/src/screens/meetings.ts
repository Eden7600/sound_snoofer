import { ui, tc, el, panel } from "../dom.ts";
import { root, updaters, c, appState } from "../state.ts";
import { command, control } from "../widgets.ts";
import { display, tone } from "../model.ts";
import { buildProviderGate, pluginStates } from "../plugin-gate.ts";
import { pluginScreens } from "../plugin-lifecycle.ts";

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
  buildShared(grid);
  buildFocused(grid);
  for (const gate of pluginScreens.meetings!) {
    if (buildProviderGate(gate, grid)) continue;
    if (gate.plugin === "discord") buildDiscord(grid);
    else buildMeet(grid);
  }
}

function buildDiscord(grid: HTMLElement) {
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
    setup.hidden = c("discord.call-mute")?.Status !== "Setup needed";
    connect.hidden = !c("discord.connect")?.Available && c("discord.connect")?.Status !== "Pending";
  });
  control("discord.call-mute", card, "Mute");
  control("discord.call-camera", card, "Camera");
}

function buildMeet(grid: HTMLElement) {
  const card = panel("Google Meet", grid);
  card.dataset.part = "googlemeet";
  stateLine("googlemeet.call", card);
  control("googlemeet.target", card, "Call");
  control("googlemeet.call-mute", card, "Mute");
  control("googlemeet.call-camera", card, "Camera");
}

function buildShared(grid: HTMLElement) {
  const card = panel("Call Controls", grid);
  card.dataset.part = "shared-meetings";
  card.append(el("p", "-mt-1.5 mb-3 text-[13px] text-muted", "All compatible calls"));
  control("meetings.mute", card, "Mute");
  control("meetings.camera", card, "Camera");
  const source = el("p", "mt-3 text-[13px] text-muted");
  source.dataset.part = "source-muted";
  card.append(source);
  updaters.push(() => {
    const mute = c("audio.mic-mute"),
      stack = c("audio.mic-stack");
    source.textContent =
      mute?.Available && mute.Value === "On"
        ? "Source muted"
        : stack?.Available && stack.Value === "Off"
          ? "Mic Stack off"
          : "";
    source.hidden = pluginStates().audio !== "running" || !source.textContent;
  });
}

function buildFocused(grid: HTMLElement) {
  const card = panel("Focused Call", grid);
  card.dataset.part = "focused-meeting";
  const calls = [...appState.controls.values()]
    .filter((item) => item.Collection === "meetings.focus")
    .sort((a, b) => (a.Order || 0) - (b.Order || 0));
  const choices = el("div", ui.actions + " mb-3");
  choices.dataset.part = "call-focus";
  card.append(choices);
  for (const item of calls) {
    const focus = command(item.ID, item.Label, choices, ui.toggle + " max-w-full whitespace-normal");
    if (!focus) continue;
    updaters.push(() => {
      const current = c(item.ID);
      focus.setAttribute("aria-pressed", String(current?.Value === "On"));
      focus.dataset.tone = current ? tone(current) : "neutral";
    });
  }
  const target = el("p", "mb-3 text-[13px] text-muted");
  target.dataset.part = "focused-call-name";
  card.append(target);
  updaters.push(() => {
    target.textContent = calls.map((item) => c(item.ID)).find((item) => item?.Value === "On")?.Label || "Not in a call";
  });
  control("meetings.deafen", card, "Deafen");
  control("meetings.share", card, "Share");
  control("meetings.leave", card, "Leave Call");
}
