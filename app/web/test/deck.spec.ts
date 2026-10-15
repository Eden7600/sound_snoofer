import { expect, test, type Gui } from "./host";
import type { DeckView } from "../src/screens/deck-types.ts";

const request = async (gui: Gui): Promise<[string, string]> => {
  const r = await gui.last();
  return [r.ID, r.Value];
};

// model replaces the edited layout's geometry, keeping the + XL bindings.
async function model(
  gui: Gui,
  id: string,
  shape: { columns: number; reserved: number; nav: boolean; keys: number; dials: number },
) {
  await gui.update(
    ([id, shape]) => {
      const preview = window.ctl("streamdeck.preview");
      const v = preview.ViewData as DeckView | undefined;
      if (!v) throw new Error("fixture has no deck view");
      Object.assign(v, {
        Model: id,
        Selected: 0,
        Regions: [],
        Dirty: false,
        Columns: shape.columns,
        Reserved: shape.reserved,
        NavDial: shape.nav,
        Keys: v.Keys.slice(0, shape.keys),
        Dials: v.Dials.slice(0, shape.dials),
      });
      preview.Revision++;
      window.ctl("streamdeck.profile").Value = "model:" + id;
    },
    [id, shape] as const,
  );
}

// Region-edit scenarios start with an empty page, while other tests use the generated layout.
async function clearRegions(gui: Gui) {
  await gui.update(() => {
    const view = window.ctl("streamdeck.preview").ViewData as DeckView | undefined;
    if (!view) throw new Error("fixture has no deck view");
    view.Regions = [];
    for (const slot of [...view.Keys, ...view.Dials]) slot.Region = -1;
  });
}

test.describe("Stream Deck", () => {
  test.beforeEach(async ({ gui }) => {
    await gui.open();
    await gui.screen("Stream Deck");
    await gui.page.getByRole("heading", { name: "Binding", exact: true }).waitFor();
  });

  test("keys are numbered and arrow browsing sends nothing", async ({ gui }) => {
    const { page } = gui;
    await expect(page.getByLabel("Device", { exact: true }).locator("option")).toHaveText([
      "Stream Deck + XL",
      "Stream Deck +",
      "Stream Deck XL",
      "Stream Deck MK.2",
      "Stream Deck Neo",
      "Desk (Stream Deck +)",
    ]);
    await expect(page.locator("[data-part=deck-key]")).toHaveCount(36);
    await expect(page.locator("[data-part=deck-key] small").first(), "key numbers start at 1").toHaveText("1");
    await page.getByRole("button", { name: "Key 1: Mic Mute", exact: true }).focus();
    await page.keyboard.press("ArrowDown");
    await expect(page.locator(":focus")).toHaveAttribute("aria-label", /^Key 10:/);
    expect(await gui.sent(), "arrow browsing fired an action").toEqual([]);
  });

  test("clicking a key selects it", async ({ gui }) => {
    const { page } = gui;
    await page.getByRole("button", { name: "Key 8: Empty", exact: true }).click();
    await expect(page.locator("[data-part=inspector] [data-part=eyebrow]")).toHaveText("KEY 8");
    expect((await gui.last()).ID).toBe("streamdeck.slot");
  });

  test("key regions: range, add, source, overflow and remove", async ({ gui }) => {
    const { page } = gui;
    await clearRegions(gui);
    await gui.update(() => {
      // Hue is still a hand-written provider; include its collection for the source-change scenario.
      const view = window.ctl("streamdeck.preview").ViewData as DeckView | undefined;
      if (!view?.Collections) throw new Error("fixture has no deck collections");
      view.Collections.push({ ID: "hue.room-scenes", Label: "Scenes · Selected Room" });
    });
    await page.getByRole("button", { name: /^Key 19: / }).click();
    await expect(page.locator("[data-part=inspector] [data-part=eyebrow]")).toHaveText("KEY 19");
    await gui.settled();
    const before = (await gui.sent()).length;
    await page.getByRole("button", { name: /^Key 35: / }).click({ modifiers: ["Shift"] });
    await page.keyboard.press("Shift+ArrowRight");
    expect((await gui.sent()).length, "range selection dispatched").toBe(before);
    await expect(page.locator("[data-part=deck-key][data-range=true]")).toHaveCount(18);
    const addRegion = page.locator("[data-part=add-region]");
    await expect(addRegion).toHaveText("Add Region · Keys 19–36");
    await page.getByLabel("New Region Source").selectOption("soundboard.clips");
    await addRegion.click();
    expect(await request(gui)).toEqual(["streamdeck.region-add", "18,35,soundboard.clips"]);
    await page.locator("[data-part=region]").waitFor();
    await expect(page.locator("[data-part=deck-key][data-region=true]")).toHaveCount(18);
    await expect(page.locator("[data-part=deck-key][data-range=true]"), "selection kept after adding").toHaveCount(0);
    await page.getByRole("button", { name: /^Key 27: / }).click({ modifiers: ["Shift"] });
    await expect(addRegion, "region offered over an existing region").toBeDisabled();
    await gui.settled();
    await page.getByLabel("Region 1 Source").selectOption("hue.room-scenes");
    expect(await request(gui)).toEqual(["streamdeck.region-source", "0,hue.room-scenes"]);
    await gui.settled();
    // Overflow: the page shows its sets, and unchecking clips the region.
    await expect(page.locator("[data-part=page-sets]")).toHaveText("3 sets · page dial visits each");
    await expect(
      page.getByRole("button", { name: "Make Home", exact: true }),
      "Make Home offered on Home",
    ).toBeDisabled();
    const overflow = page.getByLabel("Region 1 Overflow");
    await expect(overflow, "new region does not overflow").toBeChecked();
    await overflow.uncheck();
    expect(await request(gui)).toEqual(["streamdeck.region-clip", "0,on"]);
    await page.locator("[data-part=page-sets]").waitFor({ state: "hidden" });
    await expect(page.getByLabel("Region 1 Overflow")).not.toBeChecked();
    await gui.update(() => {
      const preview = window.ctl("streamdeck.preview");
      const v = preview.ViewData as DeckView | undefined;
      if (!v) throw new Error("fixture has no deck view");
      const region = v.Regions?.[0];
      if (!region) throw new Error("fixture has no region");
      region.Stacked = "Apps";
      v.Scrolls = true;
      region.Clip = false;
      v.Sets = 2;
      preview.Revision++;
    });
    await page.locator("[data-part=region]").getByText("+ Apps", { exact: true }).waitFor();
    await expect(page.locator("[data-part=page-sets]")).toHaveText("2 sets · Up/Down");
    await gui.settled();
    await page.getByRole("button", { name: "Remove Region 1", exact: true }).click();
    expect(await request(gui)).toEqual(["streamdeck.region-remove", "0"]);
    await page.locator("[data-part=region]").waitFor({ state: "detached" });
  });

  test("dial regions from a span of dials", async ({ gui }) => {
    const { page } = gui;
    await clearRegions(gui);
    await page.getByRole("button", { name: /^Dial 3: / }).click();
    await expect(page.locator("[data-part=inspector] [data-part=eyebrow]")).toHaveText("DIAL 3");
    await gui.settled();
    const before = (await gui.sent()).length;
    await page.getByRole("button", { name: /^Dial 5: / }).click({ modifiers: ["Shift"] });
    expect((await gui.sent()).length, "dial range selection dispatched").toBe(before);
    await expect(page.locator("[data-part=dial][data-range=true]")).toHaveCount(3);
    const addRegion = page.locator("[data-part=add-region]");
    await expect(addRegion).toHaveText("Add Region · Dials 3–5");
    await page.getByLabel("New Region Source").selectOption("soundboard.clips");
    await addRegion.click();
    expect(await request(gui)).toEqual(["streamdeck.region-add", "38,40,soundboard.clips"]);
    await page.locator("[data-part=region]").waitFor();
    await expect(page.locator("[data-part=dial][data-region=true]")).toHaveCount(3);
    await expect(page.locator("[data-part=region]")).toContainText("Dials 3–5");
    await gui.settled();
    await page.getByRole("button", { name: "Remove Region 1", exact: true }).click();
    await page.locator("[data-part=region]").waitFor({ state: "detached" });
  });

  test("page name drafts survive polling and apply explicitly", async ({ gui }) => {
    const { page } = gui;
    const name = page.getByLabel("Name", { exact: true });
    await name.fill("Unsubmitted name");
    await page.waitForTimeout(600); // Several polls.
    await expect(name).toHaveValue("Unsubmitted name");
    await name.press("Escape");
    await name.fill("Studio");
    await page.getByRole("button", { name: "Apply", exact: true }).first().click();
    expect((await gui.last()).Value).toBe("Studio");
  });

  test("a dirty layout saves", async ({ gui }) => {
    const { page } = gui;
    await gui.update(() => {
      const view = window.ctl("streamdeck.preview").ViewData as DeckView | undefined;
      if (!view) throw new Error("fixture has no deck view");
      view.Dirty = true;
    });
    const save = page.getByRole("button", { name: "Save Layout", exact: true });
    await expect(save).toBeEnabled();
    await save.click();
    expect((await gui.last()).ID).toBe("streamdeck.save");
  });

  test("Stream Deck + draws its grid, dials and page dial", async ({ gui }) => {
    const { page } = gui;
    await model(gui, "plus", { columns: 4, reserved: -1, nav: true, keys: 8, dials: 3 });
    await expect(page.locator("[data-part=deck-key]")).toHaveCount(8);
    await expect(page.locator("[data-part=dial]")).toHaveCount(3);
    await expect(page.locator("[data-part=reserved]"), "+ page dial").toHaveCount(1);
    await page.getByRole("button", { name: "Key 1: Mic Mute", exact: true }).focus();
    await page.keyboard.press("ArrowDown");
    await expect(page.locator(":focus")).toHaveAttribute("aria-label", /^Key 5:/);
    await page.getByRole("button", { name: /^Dial 3: / }).waitFor();
  });

  test("MK.2 draws keys without dials", async ({ gui }) => {
    const { page } = gui;
    await model(gui, "mk2", { columns: 5, reserved: 14, nav: false, keys: 15, dials: 0 });
    await expect(page.locator("[data-part=deck-key]")).toHaveCount(14);
    await expect(page.locator("[data-part=dial]")).toHaveCount(0);
    await expect(page.locator("[data-part=reserved]"), "MK.2 Pages key").toHaveCount(1);
    await expect(page.locator("[aria-label='Deck Dials']"), "dial row on a deck without dials").toBeHidden();
    await page.getByRole("button", { name: "Key 1: Mic Mute", exact: true }).focus();
    await page.keyboard.press("ArrowDown");
    await expect(page.locator(":focus")).toHaveAttribute("aria-label", /^Key 6:/);
  });

  test("decks: paging, and forget only after confirmation", async ({ gui }) => {
    const { page } = gui;
    const decks = page.locator("[data-part=decks]");
    await expect(decks.getByLabel("Paging")).toHaveValue("independent");
    page.once("dialog", (d) => d.dismiss());
    await decks.getByRole("button", { name: "Forget Deck", exact: true }).click();
    expect(
      (await gui.sent()).map((a) => a.Request?.ID),
      "Forget without confirmation",
    ).not.toContain("streamdeck.deck-forget");
    page.once("dialog", (d) => d.accept());
    await decks.getByRole("button", { name: "Forget Deck", exact: true }).click();
    expect((await gui.last()).ID).toBe("streamdeck.deck-forget");
    await gui.settled();
    const count = (await gui.sent()).length;
    page.once("dialog", (d) => d.dismiss());
    await decks.getByRole("button", { name: "Own Layout" }).click();
    expect((await gui.sent()).length, "own layout dropped without confirmation").toBe(count);
  });

  test("host notices show as an alert", async ({ gui }) => {
    await gui.update(() => {
      window.fixture.Notice = "Control changed; try again";
    });
    await gui.page.getByRole("alert").filter({ hasText: "Control changed; try again" }).waitFor();
  });
});
