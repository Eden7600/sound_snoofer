import { expect, test, type Gui } from "./host.ts";
import type { PriorityView } from "../src/screens/routing-types.ts";

async function rebuild(gui: Gui) {
  const old = await gui.page.locator("#content > :first-child").elementHandle();
  if (!old) throw new Error("content missing");
  await gui.update(() => {
    window.ctl("aec.strength").Label += " updated";
  });
  await expect.poll(() => old.evaluate((node) => node.isConnected)).toBe(false);
}

test.beforeEach(async ({ gui }) => {
  await gui.open();
  await gui.screen("Routing");
});

test("Routing explicit pattern and name drafts retain their original edit revision", async ({ gui }) => {
  const pattern = gui.page.getByLabel("Playback 2 pattern");
  const mic = gui.page.getByLabel("Microphone 1 name");
  const output = gui.page.getByLabel("Rename Music");
  const revision = await gui.page.evaluate(() => window.ctl("audio.priority-edit").Revision);
  await pattern.fill("(?i)my headphones");
  await mic.fill("Desk Draft");
  await output.fill("Music Draft");
  await output.blur();
  await gui.update(() => {
    window.ctl("audio.priority-edit").Revision++;
  });
  await rebuild(gui);
  await expect(pattern).toHaveValue("(?i)my headphones");
  await expect(mic).toHaveValue("Desk Draft");
  await expect(output).toHaveValue("Music Draft");
  await expect(mic.locator("..").getByRole("button", { name: "Apply", exact: true })).toBeVisible();
  expect(await gui.sent()).toEqual([]);
  await pattern.press("Enter");
  expect(await gui.last()).toMatchObject({ ID: "audio.priority-edit", Revision: revision });
});

test("Routing configuration replacement discards drafts drawn from the previous lists", async ({ gui }) => {
  const mic = gui.page.getByLabel("Microphone 1 name");
  const pattern = gui.page.getByLabel("Playback 2 pattern");
  await mic.fill("Old microphone");
  await pattern.fill("old device");
  await pattern.blur();
  await gui.update(() => {
    const view = window.ctl<PriorityView>("audio.priorities").ViewData;
    const playback = view?.Lists?.playback;
    if (!view?.Mics || !playback?.[1]) throw new Error("Routing fixture missing");
    view.Mics.reverse();
    playback[1].Pattern = "replacement pattern";
    window.ctl("audio.priority-edit").Value = "changed-configuration";
    window.ctl("audio.output-edit").Value = "changed-configuration";
  });
  await expect(mic).toHaveValue("Webcam");
  await expect(pattern).toHaveValue("replacement pattern");
  expect(await gui.sent()).toEqual([]);
});

test("Routing Add names and choices survive unrelated structure without dispatch", async ({ gui }) => {
  const micName = gui.page.getByLabel("New Microphone Name");
  const micSource = gui.page.getByLabel("New Microphone Source");
  const outputName = gui.page.getByLabel("New Output Name");
  const outputDevice = gui.page.getByLabel("New Output Device");
  const priority = gui.page.getByLabel("Microphone to Add");
  await micName.fill("Portable");
  await micName.blur();
  await micSource.selectOption("0");
  await outputName.fill("Broadcast");
  await outputName.blur();
  await outputDevice.selectOption("1");
  await priority.selectOption("off");
  await rebuild(gui);
  await expect(micName).toHaveValue("Portable");
  await expect(micSource).toHaveValue("0");
  await expect(outputName).toHaveValue("Broadcast");
  await expect(outputDevice).toHaveValue("1");
  await expect(priority).toHaveValue("off");
  expect(await gui.sent()).toEqual([]);
});

test("No device creates a named output without choosing the first available device", async ({ gui }) => {
  await gui.page.getByLabel("New Output Name").fill("Unassigned");
  const device = gui.page.getByLabel("New Output Device");
  expect(await device.locator("option").count()).toBeGreaterThan(1);
  await device.selectOption("");
  await gui.page.getByRole("button", { name: "Add Output", exact: true }).click();
  const edit = JSON.parse((await gui.last("audio.output-edit")).Value) as { value: string };
  expect(JSON.parse(edit.value)).toEqual({ name: "Unassigned" });
});

test("a reused output option number clears and stays invalid until an explicit new choice", async ({ gui }) => {
  const name = gui.page.getByLabel("New Output Name");
  const device = gui.page.getByLabel("New Output Device");
  const add = gui.page.getByRole("button", { name: "Add Output", exact: true });
  await name.fill("Broadcast");
  await name.blur();
  await device.selectOption("0");
  await gui.update(() => {
    window.ctl<PriorityView>("audio.priorities").ViewData?.OutputDevices?.reverse();
  });
  await expect(name).toHaveValue("Broadcast");
  await expect(device).toHaveJSProperty("selectedIndex", -1);
  await expect(add).toBeDisabled();
  await rebuild(gui);
  await expect(device).toHaveJSProperty("selectedIndex", -1);
  await expect(add).toBeDisabled();
  expect(await gui.sent()).toEqual([]);
  await device.selectOption("");
  await expect(add).toBeEnabled();
  await add.click();
  const edit = JSON.parse((await gui.last()).Value) as { value: string };
  expect(JSON.parse(edit.value)).toEqual({ name: "Broadcast" });
});

test("changed microphone and interface option identities require reselection", async ({ gui }) => {
  const name = gui.page.getByLabel("New Microphone Name");
  const source = gui.page.getByLabel("New Microphone Source");
  const driver = gui.page.getByLabel("New Interface Driver");
  await name.fill("Portable");
  await name.blur();
  await source.selectOption("0");
  await driver.selectOption("0");
  await gui.update(() => {
    const view = window.ctl<PriorityView>("audio.priorities").ViewData;
    const mic = view?.Suggestions?.["mic-devices:"]?.[0];
    const driver = view?.Drivers?.[0];
    if (!mic || !driver) throw new Error("catalogue fixture missing");
    mic.ID = "replacement-mic";
    mic.Name = "Replacement Microphone";
    driver.ID = "replacement-driver";
    driver.Name = "Replacement Driver";
  });
  await expect(name).toHaveValue("Portable");
  await expect(source).toHaveJSProperty("selectedIndex", -1);
  await expect(driver).toHaveJSProperty("selectedIndex", -1);
  await expect(gui.page.getByRole("button", { name: "Add Microphone", exact: true })).toBeDisabled();
  await expect(
    gui.page.locator("[data-part=interface-add]").getByRole("button", { name: "Add", exact: true }),
  ).toBeDisabled();
  expect(await gui.sent()).toEqual([]);
});

test("an unavailable microphone priority option does not become the default neighbor", async ({ gui }) => {
  const select = gui.page.getByLabel("Microphone to Add");
  await select.selectOption("off");
  await gui.update(() => {
    const view = window.ctl<PriorityView>("audio.priorities").ViewData;
    if (!view) throw new Error("view missing");
    view.Microphones = (view.Microphones || []).filter((id) => id !== "off");
  });
  await expect(select.locator("option[value=off]")).toHaveCount(0);
  await expect(select).toHaveJSProperty("selectedIndex", -1);
  await expect(
    gui.page.locator("[data-part=list-microphones]").getByRole("button", { name: "Add", exact: true }),
  ).toBeDisabled();
  expect(await gui.sent()).toEqual([]);
});
