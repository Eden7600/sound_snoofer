import { expect, test, type Gui } from "./host";
import type { DeckView } from "../src/screens/deck-types.ts";

async function range(gui: Gui) {
  await gui.page.getByRole("button", { name: /^Key 19: / }).click();
  await expect(gui.page.locator("[data-part=eyebrow]")).toHaveText("KEY 19");
  await gui.settled();
  await gui.page.getByRole("button", { name: /^Key 20: / }).click({ modifiers: ["Shift"] });
  await expect(gui.page.locator("[data-part=deck-key][data-range=true]")).toHaveCount(2);
}

test.beforeEach(async ({ gui }) => {
  await gui.open();
  await gui.screen("Stream Deck");
  await gui.update(() => {
    const view = window.ctl<DeckView>("streamdeck.preview").ViewData;
    if (!view) throw new Error("missing deck view");
    view.Regions = [];
    for (const slot of [...view.Keys, ...view.Dials]) slot.Region = -1;
    view.Collections = [
      { ID: "soundboard.clips", Label: "Sounds" },
      { ID: "hue.room-scenes", Label: "Scenes" },
    ];
  });
});

test("a local range and Add source survive an unrelated rebuild with their original revision", async ({ gui }) => {
  await range(gui);
  const source = gui.page.getByLabel("New Region Source");
  const revision = await gui.page.evaluate(() => window.ctl("streamdeck.region-add").Revision);
  await source.selectOption("hue.room-scenes");
  await source.evaluate((node) => node.blur());
  const before = (await gui.sent()).length;
  const original = await source.elementHandle();
  await gui.update(() => {
    window.ctl("aec.strength").Label += " changed";
    window.ctl("streamdeck.region-add").Revision++;
  });
  await expect.poll(() => original?.evaluate((node) => node.isConnected)).toBe(false);
  await expect(source).toHaveValue("hue.room-scenes");
  await expect(gui.page.locator("[data-part=deck-key][data-range=true]")).toHaveCount(2);
  expect((await gui.sent()).length).toBe(before);
  await gui.page.locator("[data-part=add-region]").click();
  expect(await gui.last()).toMatchObject({
    ID: "streamdeck.region-add",
    Value: "18,19,hue.room-scenes",
    Revision: revision,
  });
});

for (const changed of ["page", "profile", "shared", "slot", "geometry"] as const) {
  test(`a changed ${changed} clears the local range and source choice`, async ({ gui }) => {
    await range(gui);
    const source = gui.page.getByLabel("New Region Source");
    await source.selectOption("hue.room-scenes");
    await source.evaluate((node) => node.blur());
    const before = (await gui.sent()).length;
    await gui.update((changed) => {
      const view = window.ctl<DeckView>("streamdeck.preview").ViewData;
      if (!view) throw new Error("missing deck view");
      if (changed === "page") window.ctl("streamdeck.page").Value = "another-page";
      if (changed === "profile") window.ctl("streamdeck.profile").Value = "serial:OTHER";
      if (changed === "shared") window.ctl("streamdeck.shared").Value = "On";
      if (changed === "slot") {
        view.Selected = 2;
        window.ctl("streamdeck.slot").Value = "Key 3";
      }
      if (changed === "geometry") view.Columns = 6;
    }, changed);
    await expect(gui.page.locator("[data-part=deck-key][data-range=true]")).toHaveCount(0);
    await expect(source).toHaveValue("soundboard.clips");
    await expect(gui.page.locator("[data-part=add-region]")).toBeDisabled();
    expect((await gui.sent()).length).toBe(before);
  });
}

test("workspace navigation drops the range and source choice", async ({ gui }) => {
  await range(gui);
  await gui.page.getByLabel("New Region Source").selectOption("hue.room-scenes");
  const before = (await gui.sent()).length;
  await gui.screen("Audio");
  await gui.screen("Stream Deck");
  await expect(gui.page.locator("[data-part=deck-key][data-range=true]")).toHaveCount(0);
  await expect(gui.page.getByLabel("New Region Source")).toHaveValue("soundboard.clips");
  await expect(gui.page.locator("[data-part=add-region]")).toBeDisabled();
  expect((await gui.sent()).length).toBe(before);
});

test("a removed collection clears Add source and requires explicit reselection", async ({ gui }) => {
  await range(gui);
  const source = gui.page.getByLabel("New Region Source");
  await source.selectOption("hue.room-scenes");
  await source.evaluate((node) => node.blur());
  const before = (await gui.sent()).length;
  await gui.update(() => {
    const view = window.ctl<DeckView>("streamdeck.preview").ViewData;
    if (!view) throw new Error("missing deck view");
    view.Collections = view.Collections?.filter((item) => item.ID !== "hue.room-scenes") || [];
  });
  await expect(source.locator('option[value="hue.room-scenes"]')).toHaveCount(0);
  await expect(source).toHaveJSProperty("selectedIndex", -1);
  await expect(gui.page.locator("[data-part=add-region]")).toBeDisabled();
  expect((await gui.sent()).length).toBe(before);
  await source.selectOption("soundboard.clips");
  await expect(gui.page.locator("[data-part=add-region]")).toBeEnabled();
});

test("focused Add source defers replacement and restores on a no-change poll after blur", async ({ gui }) => {
  await range(gui);
  const source = gui.page.getByLabel("New Region Source");
  await source.selectOption("hue.room-scenes");
  await source.focus();
  const original = await source.elementHandle();
  const unchanged = await gui.page.evaluate(() => {
    window.liveMeters = false;
    return window.transport.unchanged;
  });
  await gui.update(() => {
    window.ctl("aec.strength").Label += " changed";
  });
  await expect.poll(() => gui.page.evaluate(() => window.transport.unchanged)).toBeGreaterThan(unchanged);
  expect(await original?.evaluate((node) => node.isConnected)).toBe(true);
  await source.evaluate((node) => node.blur());
  await expect.poll(() => original?.evaluate((node) => node.isConnected)).toBe(false);
  await expect(source).toHaveValue("hue.room-scenes");
  await expect(gui.page.locator("[data-part=deck-key][data-range=true]")).toHaveCount(2);
});
