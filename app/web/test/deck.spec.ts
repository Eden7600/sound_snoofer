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

  test("status bindings are offered on keys and excluded from dials", async ({ gui }) => {
    const binding = gui.page.getByLabel("Binding", { exact: true });
    for (const id of ["audio.playback-device", "audio.slot-1"]) {
      await expect(binding.locator('option[value="' + id + '"]')).toHaveCount(1);
    }
    for (const id of ["audio.priority-edit", "audio.app-voicemeeter"]) {
      await expect(binding.locator('option[value="' + id + '"]')).toHaveCount(0);
    }
    await gui.page.getByRole("button", { name: /^Dial 1: / }).click();
    await expect(gui.page.locator("[data-part=inspector] [data-part=eyebrow]")).toHaveText("DIAL 1");
    for (const id of ["audio.playback-device", "audio.slot-1", "audio.priority-edit", "audio.app-voicemeeter"]) {
      await expect(binding.locator('option[value="' + id + '"]')).toHaveCount(0);
    }
    await expect(binding.locator('option[value="audio.gain-playback"]')).toHaveCount(1);
    await gui.settled();
    await gui.page.getByRole("button", { name: "Key 8: Empty", exact: true }).click();
    await gui.settled();
    await binding.selectOption("audio.playback-device");
    expect(await request(gui)).toEqual(["streamdeck.binding", "audio.playback-device"]);
    await gui.settled();
  });

  test("clicking a key selects it", async ({ gui }) => {
    const { page } = gui;
    await page.getByRole("button", { name: "Key 8: Empty", exact: true }).click();
    await expect(page.locator("[data-part=inspector] [data-part=eyebrow]")).toHaveText("KEY 8");
    expect((await gui.last()).ID).toBe("streamdeck.slot");
  });

  test("binding names distinguish profiles and preserve custom labels and IDs", async ({ gui }) => {
    await gui.update(() => {
      const normal = window.ctl("audio.normal-mode");
      window.fixture.Controls.push(
        { ...normal, ID: "audio.vr-profile-mode", Label: "VR Processing" },
        { ...normal, ID: "appaudio.custom", Label: "mY studio" },
      );
    });
    const binding = gui.page.getByLabel("Binding", { exact: true });
    await expect(binding.locator('option[value="audio.normal-mode"]')).toHaveText("Normal Processing · Audio");
    await expect(binding.locator('option[value="audio.vr-profile-mode"]')).toHaveText("VR Processing · Audio");
    const custom = binding.locator('option[value="appaudio.custom"]');
    await expect(custom).toHaveText("mY studio · App Audio");
    await expect(custom).toHaveAttribute("title", "appaudio.custom");
    const search = gui.page.getByLabel("Find a Binding");
    for (const query of ["App Audio", "appaudio", "mY studio"]) {
      await search.fill(query);
      await expect(custom).toHaveCount(1);
    }
    await binding.selectOption("appaudio.custom");
    expect(await request(gui)).toEqual(["streamdeck.binding", "appaudio.custom"]);
    await gui.settled();
    await search.blur(); // Structural updates preserve an active text edit until blur.
    await gui.update(() => {
      window.fixture.PluginLabels = { ...window.fixture.PluginLabels, appaudio: "My Apps" };
    });
    await expect(custom).toHaveText("mY studio · My Apps");
    await gui.update(() => {
      delete window.fixture.PluginLabels;
    });
    await expect(custom).toHaveText("mY studio · appaudio");
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

  test("page selection preserves built-in and custom names with raw page IDs", async ({ gui }) => {
    const pages = gui.page.getByLabel("Page", { exact: true });
    await expect(pages.locator('option[value="home"]')).toHaveText("Home");
    await gui.update(() => {
      const page = window.ctl("streamdeck.page");
      page.Options = [...(page.Options || []), "my_studio"];
      page.OptionLabels = { ...page.OptionLabels, my_studio: "mY Studio" };
      page.Revision++;
    });
    await expect(pages.locator('option[value="my_studio"]')).toHaveText("mY Studio");
    await pages.selectOption({ label: "mY Studio" });
    expect(await request(gui)).toEqual(["streamdeck.page", "my_studio"]);
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
    await decks.getByRole("button", { name: "Forget Deck", exact: true }).click();
    await page.getByRole("dialog").getByRole("button", { name: "Cancel", exact: true }).click();
    expect(
      (await gui.sent()).map((a) => a.Request?.ID),
      "Forget without confirmation",
    ).not.toContain("streamdeck.deck-forget");
    await decks.getByRole("button", { name: "Forget Deck", exact: true }).click();
    await page.getByRole("dialog").getByRole("button", { name: "Forget Deck", exact: true }).click();
    expect((await gui.last()).ID).toBe("streamdeck.deck-forget");
    await gui.settled();
    const count = (await gui.sent()).length;
    await decks.getByRole("button", { name: "Own Layout" }).click();
    await page.getByRole("dialog").getByRole("button", { name: "Cancel", exact: true }).click();
    expect((await gui.sent()).length, "own layout dropped without confirmation").toBe(count);
  });

  test("host notices show as an alert", async ({ gui }) => {
    await gui.update(() => {
      window.fixture.Notice = "Control changed; try again";
    });
    await gui.page.getByRole("alert").filter({ hasText: "Control changed; try again" }).waitFor();
  });
});
