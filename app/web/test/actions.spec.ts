import { expect, test, type Gui } from "./host.ts";
import { holdActions, reply } from "./action-host.ts";
import AxeBuilder from "@axe-core/playwright";
import type { PriorityView } from "../src/screens/routing-types.ts";
const feedback = (gui: Gui, id: string) => gui.page.locator(`[data-action-feedback="${id}"]`).filter({ visible: true });

test("independent mutes remain usable and settle out of order", async ({ gui }) => {
  await gui.open();
  await holdActions(gui);
  const mic = gui.page.getByRole("button", { name: "Mic Mute", exact: true });
  const playback = gui.page.getByRole("button", { name: "Playback Mute", exact: true });
  await mic.click();
  await expect(mic).toBeDisabled();
  await expect(playback).toBeEnabled({ timeout: 500 });
  await playback.click();
  const sent = await gui.sent();
  expect(sent).toHaveLength(2);
  expect(sent[0]?.ID).toBeTruthy();
  expect(sent[0]?.ID).not.toBe(sent[1]?.ID);
  await reply(gui, "audio.speaker-mute", "Processed", "Playback preference saved");
  await expect(playback).toBeEnabled();
  await expect(mic).toBeDisabled();
  await expect(feedback(gui, "audio.speaker-mute")).toContainText("Processed");
  await reply(gui, "audio.mic-mute", "Failed", "Settings could not be saved");
  await expect(mic).toBeEnabled();
  await expect(feedback(gui, "audio.mic-mute")).toContainText("Settings could not be saved");
});

test("terminal feedback and uncertainty stay announced while another action is pending", async ({ gui }) => {
  await gui.open();
  await holdActions(gui);
  await gui.page.clock.install();
  const mic = gui.page.getByRole("button", { name: "Mic Mute", exact: true });
  const playback = gui.page.getByRole("button", { name: "Playback Mute", exact: true });
  await mic.click();
  await playback.click();
  await reply(gui, "audio.mic-mute", "Failed", "Settings could not be saved");
  await expect(mic).toBeEnabled();
  await expect(playback).toBeDisabled();
  const failed = feedback(gui, "audio.mic-mute").getByRole("status");
  await expect(failed).toContainText("Settings could not be saved");
  await expect(failed).toBeVisible();
  expect(await failed.evaluate((node) => node.closest('[aria-busy="true"]'))).toBeNull();
  await gui.page.clock.runFor(2700);
  const uncertain = feedback(gui, "audio.speaker-mute").getByRole("status");
  await expect(uncertain).toContainText("Outcome unknown; still waiting.");
  await expect(uncertain).toBeVisible();
  await expect(playback).toBeDisabled();
  expect(await uncertain.evaluate((node) => node.closest('[aria-busy="true"]'))).toBeNull();
});

test("deadline and state changes cannot release an unresolved action", async ({ gui }) => {
  await gui.open();
  await holdActions(gui);
  await gui.page.clock.install();
  const mic = gui.page.getByRole("button", { name: "Mic Mute", exact: true });
  await mic.click();
  await gui.update(() => {
    window.ctl("audio.mic-mute").Revision++;
    window.fixture.Notice = "Unrelated diagnostic notice";
  });
  await gui.page.clock.runFor(2700);
  await expect(mic).toBeDisabled();
  await expect(feedback(gui, "audio.mic-mute")).toContainText("Outcome unknown; still waiting.");
  await reply(gui, "audio.mic-mute", "Processed", "Preference saved");
  await expect(mic).toBeEnabled();
  await expect(feedback(gui, "audio.mic-mute")).toContainText("Preference saved");
  expect(await gui.sent()).toHaveLength(1);
});

test("a late failure remains on its origin screen even when another screen shows the same control", async ({ gui }) => {
  await gui.open();
  await holdActions(gui);
  await gui.page.getByRole("button", { name: "Restart Audio Engine", exact: true }).click();
  await gui.screen("Diagnostics");
  await reply(gui, "audio.engine-restart", "Failed", "Engine could not restart");
  await expect(gui.page.locator("#content")).not.toContainText("Engine could not restart");
  await gui.screen("Audio");
  await expect(feedback(gui, "audio.engine-restart")).toContainText("Engine could not restart");
  await feedback(gui, "audio.engine-restart").getByRole("button", { name: "Dismiss" }).click();
  await expect(gui.page.locator("#content")).not.toContainText("Engine could not restart");
});

test("Submitted acknowledges dispatch without claiming the operation succeeded", async ({ gui }) => {
  await gui.open();
  await holdActions(gui);
  await gui.page.getByRole("button", { name: "Mic Mute", exact: true }).click();
  await reply(gui, "audio.mic-mute", "Submitted", "Queued for processing");
  await expect(feedback(gui, "audio.mic-mute")).toContainText("Submitted");
  await expect(feedback(gui, "audio.mic-mute")).not.toContainText(/Applied|Success/);
  await expect(gui.page.getByRole("button", { name: "Mic Mute", exact: true })).toBeEnabled();
});

test("an explicit repeat is rejected without releasing or replacing the original reservation", async ({ gui }) => {
  await gui.open();
  await holdActions(gui);
  const mic = gui.page.getByRole("button", { name: "Mic Mute", exact: true });
  await mic.click();
  // Programmatic dispatch exercises the send guard even if native disabled UI
  // prevents an ordinary second click from reaching it.
  await mic.dispatchEvent("click");
  await expect(feedback(gui, "audio.mic-mute")).toContainText("Rejected");
  expect(await gui.sent()).toHaveLength(1);
  await expect(mic).toBeDisabled();
  await reply(gui, "audio.mic-mute", "Processed", "Original request processed");
  await expect(mic).toBeEnabled();
  await expect(feedback(gui, "audio.mic-mute")).toContainText("Rejected");
  await expect(feedback(gui, "audio.mic-mute")).not.toContainText("Original request processed");
});

test("a reply naming another outstanding ID cannot settle either reservation", async ({ gui }) => {
  await gui.open();
  await holdActions(gui);
  const mic = gui.page.getByRole("button", { name: "Mic Mute", exact: true });
  const playback = gui.page.getByRole("button", { name: "Playback Mute", exact: true });
  await mic.click();
  await playback.click();
  await gui.update(() => {
    const [first, second] = window.sent;
    if (!first || !second) throw new Error("Missing actions");
    window.actionReplies[first.ID]?.({ ID: second.ID, Status: "Processed", Detail: "Wrong result" });
  });
  await expect(mic).toBeDisabled();
  await expect(playback).toBeDisabled();
  await expect(feedback(gui, "audio.mic-mute")).toContainText("unmatched result");
  await reply(gui, "audio.speaker-mute", "Processed");
  await expect(playback).toBeEnabled();
  await expect(mic).toBeDisabled();
});

test("disconnect releases uncertain reservations and ignores a late reply without replay", async ({ gui }) => {
  await gui.open();
  await holdActions(gui);
  await gui.page.clock.install();
  const mic = gui.page.getByRole("button", { name: "Mic Mute", exact: true });
  await mic.click();
  await gui.update(() => {
    const desktop = window.go?.app?.Desktop;
    if (!desktop) throw new Error("Desktop missing");
    const state = desktop.State;
    desktop.State = async () => {
      desktop.State = state;
      throw new Error("Connection lost");
    };
  });
  await gui.page.clock.runFor(150);
  await expect(feedback(gui, "audio.mic-mute")).toContainText("host disconnected");
  await reply(gui, "audio.mic-mute", "Processed", "Late result must not replace uncertainty");
  await gui.page.clock.runFor(500);
  await expect(gui.page.locator("#connection")).toBeHidden();
  await expect(mic).toBeEnabled();
  await expect(feedback(gui, "audio.mic-mute")).toContainText("host disconnected");
  expect(await gui.sent()).toHaveLength(1);
});

test("a removed target keeps one origin-screen summary for its delayed failure", async ({ gui }) => {
  await gui.open();
  await holdActions(gui);
  await gui.page.getByRole("button", { name: "Mic Mute", exact: true }).click();
  await gui.update(() => {
    window.fixture.Controls = window.fixture.Controls.filter((control) => control.ID !== "audio.mic-mute");
  });
  await expect(gui.page.getByRole("button", { name: "Mic Mute", exact: true })).toHaveCount(0);
  await reply(gui, "audio.mic-mute", "Failed", "The original device disappeared");
  await expect(gui.page.locator("[data-action-summary]")).toContainText("The original device disappeared");
  await gui.screen("Routing");
  await expect(gui.page.locator("#content")).not.toContainText("The original device disappeared");
  await gui.screen("Audio");
  await expect(gui.page.locator("[data-action-summary] [data-action-id]")).toHaveCount(1);
  await expect(gui.page.locator("[data-action-summary]")).toContainText("The original device disappeared");
});

test("semantic mirrors share busy state while unrelated controls remain usable", async ({ gui }) => {
  await gui.open((fixture) => {
    const gain = fixture.Controls.find((control) => control.ID === "audio.gain-mic");
    if (!gain) throw new Error("Missing gain");
    gain.Mirrors = "audio.mic-mute";
  });
  await holdActions(gui);
  await gui.page.getByRole("button", { name: "Mic Mute", exact: true }).click();
  await expect(gui.page.getByRole("button", { name: "MICROPHONE down", exact: true })).toBeDisabled();
  await expect(gui.page.getByRole("button", { name: "Playback Mute", exact: true })).toBeEnabled();
  await reply(gui, "audio.mic-mute", "Processed");
  await expect(gui.page.getByRole("button", { name: "MICROPHONE down", exact: true })).toBeEnabled();
});

test("the seventeenth outstanding action is rejected locally with no dispatch", async ({ gui }) => {
  await gui.open((fixture) => {
    for (let n = 0; n < 17; n++)
      fixture.Controls.push({
        ID: "audio.capacity-" + n,
        Label: "Capacity " + n,
        Kind: "toggle",
        Value: "Off",
        Available: true,
        Revision: 1,
        Operations: ["press"],
        Group: "Recovery",
      });
  });
  await holdActions(gui);
  await gui.update(() => {
    for (let n = 0; n < 17; n++) document.querySelector<HTMLButtonElement>(`[aria-label="Capacity ${n}"]`)?.click();
  });
  expect(await gui.sent()).toHaveLength(16);
  await expect(feedback(gui, "audio.capacity-16")).toContainText("Too many actions");
  await expect(gui.page.getByRole("button", { name: "Capacity 16", exact: true })).toBeEnabled();
  await expect(gui.page.getByRole("button", { name: "Capacity 0", exact: true })).toBeDisabled();
});

for (const kind of ["confirm", "cancel"] as const) {
  test(`host ${kind} keeps the proposal ID and wording displayed when the dialog opened`, async ({ gui }) => {
    await gui.open((fixture) => {
      fixture.Confirmation = "Enable Soundboard?";
      fixture.ProposalID = "proposal-one";
    });
    await holdActions(gui);
    const dialog = gui.page.getByRole("dialog");
    await expect(dialog).toContainText("Enable Soundboard?");
    await gui.update(() => {
      window.fixture.Confirmation = "Disable Audio?";
      window.fixture.ProposalID = "proposal-two";
    });
    await gui.page.waitForFunction(() => window.transport.polls > 2);
    await expect(dialog).toContainText("Enable Soundboard?");
    await expect(dialog).not.toContainText("Disable Audio?");
    await dialog.getByRole("button", { name: kind === "confirm" ? "Apply & restart" : "Cancel", exact: true }).click();
    expect((await gui.sent()).at(-1)).toMatchObject({ Kind: kind, ProposalID: "proposal-one" });
    await expect(dialog.getByRole("button", { name: "Apply & restart", exact: true })).toBeDisabled();
    await expect(dialog.getByRole("button", { name: "Cancel", exact: true })).toBeDisabled();
  });
}

test("an unrelated pending action never bypasses focused-input rebuild deferral", async ({ gui }) => {
  await gui.open();
  await holdActions(gui);
  await gui.page.getByRole("button", { name: "Mic Mute", exact: true }).click();
  await gui.screen("Routing");
  const pattern = gui.page.locator("[data-part=list-playback]").getByLabel("Playback 2 pattern");
  const initial = await pattern.inputValue();
  await pattern.focus();
  await gui.update(() => {
    const row = window.ctl<PriorityView>("audio.priorities").ViewData?.Lists?.playback?.[1];
    if (!row) throw new Error("Priority row missing");
    row.Pattern = "changed-source";
    window.ctl("audio.priority-edit").Value = "changed-configuration";
    window.liveMeters = false;
  });
  await expect.poll(() => gui.page.evaluate(() => window.transport.unchanged)).toBeGreaterThan(0);
  await expect(pattern).toBeFocused();
  await expect(pattern).toHaveValue(initial);
  await pattern.blur();
  await expect(pattern).toHaveValue("changed-source");
  expect(await gui.sent()).toHaveLength(1);
});

for (const size of [
  { width: 1280, height: 820 },
  { width: 800, height: 600 },
]) {
  test(`local action feedback remains readable and accessible at ${size.width}px`, async ({ gui }, info) => {
    await gui.page.setViewportSize(size);
    await gui.open();
    await holdActions(gui);
    await gui.page.getByRole("button", { name: "Mic Mute", exact: true }).click();
    await reply(
      gui,
      "audio.mic-mute",
      "Failed",
      "Could not save microphone settings. Check the settings file and try again.",
    );
    await gui.page.getByRole("button", { name: "Playback Mute", exact: true }).click();
    await expect(feedback(gui, "audio.mic-mute")).toContainText("Failed");
    await expect(feedback(gui, "audio.speaker-mute")).toContainText("Sending");
    await gui.noHorizontalScroll();
    const results = await new AxeBuilder({ page: gui.page })
      .include("#content")
      .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"])
      .analyze();
    expect(results.violations).toEqual([]);
    const screenshot = info.outputPath("action-feedback.png");
    await gui.page.screenshot({ path: screenshot, fullPage: true });
    await info.attach("action-feedback", { path: screenshot, contentType: "image/png" });
  });
}
