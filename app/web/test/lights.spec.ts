import { expect, test, type Gui } from "./host";

const scenes = (gui: Gui) => gui.page.locator("[data-part=room-scenes] [data-part=scene] strong");

test.describe("Lights", () => {
  test("pairing sends Pair", async ({ gui }) => {
    await gui.open();
    await gui.screen("Lights");
    await gui.page.getByRole("heading", { name: "Bridge found at 172.16.102.3" }).waitFor();
    await gui.page.getByRole("button", { name: "Pair", exact: true }).click();
    expect((await gui.last()).ID).toBe("hue.pair");
  });

  test.describe("connected", () => {
    test.beforeEach(async ({ gui }) => {
      await gui.open((fixture) => {
        const status = fixture.Controls.find((c) => c.ID === "hue.status")!;
        status.Value = "Connected";
        status.ViewData = null;
      });
      await gui.screen("Lights");
      await expect(gui.page.locator("[data-part=setup]")).toBeHidden();
    });

    test("room scenes, artwork and other rooms", async ({ gui }) => {
      const { page } = gui;
      expect(await scenes(gui).allTextContents()).toEqual(["Bright", "Concentrate", "Relax"]);
      await expect(page.locator("[data-part=scene][data-state=playing]")).toHaveCount(1);
      await expect(page.locator("[data-part=scene] img"), "scene artwork not shown").toHaveCount(1);
      expect(await page.locator("details [data-part=room-heading]").allTextContents()).toEqual(["Desk", "Kitchen"]);
    });

    test("brightness, scenes and motion", async ({ gui }) => {
      const { page } = gui;
      await page.getByRole("button", { name: "BRIGHTNESS up", exact: true }).click();
      let r = await gui.last();
      expect([r.ID, r.Operation, r.Delta]).toEqual(["hue.brightness", "adjust", 1]);
      await gui.settled();
      await page.getByRole("slider", { name: "BRIGHTNESS" }).fill("80");
      r = await gui.last();
      expect([r.ID, r.Delta]).toEqual(["hue.brightness", 9]);
      await gui.settled();
      await page.getByRole("button", { name: "Relax", exact: false }).click();
      expect((await gui.last()).ID).toBe("hue.scene-studio-relax");
      await gui.settled();
      await page.getByRole("button", { name: "Motion Sensors", exact: true }).click();
      r = await gui.last();
      expect([r.ID, r.Operation]).toEqual(["hue.motion", "press"]);
      await gui.update(() => {
        const motion = window.ctl("hue.motion");
        motion.Hidden = true;
        motion.Available = false;
        motion.Revision++;
      });
      await page.locator("[data-part=motion]").waitFor({ state: "hidden" });
    });

    test("scene sort follows the host's order", async ({ gui }) => {
      const { page } = gui;
      await page.locator("[data-part=sort]").getByRole("button", { name: "Brightness", exact: true }).click();
      const r = await gui.last();
      expect([r.ID, r.Operation, r.Value]).toEqual(["hue.sort", "set", "brightness"]);
      // The host's Order, not the label, places the cards.
      await gui.update(() => {
        for (const [id, order] of [
          ["hue.scene-studio-relax", 1],
          ["hue.scene-studio-bright", 2],
          ["hue.scene-studio-concentrate", 3],
        ] as const) {
          const scene = window.ctl(id);
          scene.Order = order;
          scene.Revision++;
        }
        const sort = window.ctl("hue.sort");
        sort.Value = "brightness";
        sort.Revision++;
      });
      await expect(scenes(gui)).toHaveText(["Relax", "Bright", "Concentrate"]);
      await expect(page.locator("[data-part=sort] [aria-pressed=true]")).toHaveText("Brightness");
    });

    test("rooms: the host sets the boxes; the last room stays chosen", async ({ gui }) => {
      const { page } = gui;
      await page.getByRole("checkbox", { name: "Desk (zone)", exact: true }).click();
      const r = await gui.last();
      expect([r.ID, r.Value]).toEqual(["hue.rooms", "room-1"]);
      await gui.update(() => {
        const rooms = window.ctl("hue.rooms");
        rooms.Value = "room-1";
        rooms.ViewData[1].Chosen = false;
        rooms.Revision++;
      });
      await expect(page.locator("[data-part=rooms] input[value=zone-1]")).not.toBeChecked();
      const [room, zone] = [page.locator("[data-part=rooms] input").nth(0), page.locator("[data-part=rooms] input").nth(1)];
      await expect(room).toBeDisabled();
      await expect(zone).toBeEnabled();
    });

    test("sync mode appears while syncing is possible", async ({ gui }) => {
      const { page } = gui;
      await expect(page.getByRole("button", { name: "Games", exact: true }), "mode shown while not syncing").toBeHidden();
      await gui.update(() => {
        for (const id of ["hue.sync-mode", "hue.sync-intensity"]) {
          const control = window.ctl(id);
          control.Hidden = false;
          control.Available = true;
          control.Revision++;
        }
      });
      await page.getByRole("button", { name: "Games", exact: true }).click();
      const r = await gui.last();
      expect([r.ID, r.Value]).toEqual(["hue.sync-mode", "games"]);
      await gui.settled();
      await page.getByRole("button", { name: "Start Sync", exact: true }).click();
      expect((await gui.last()).ID).toBe("hue.sync");
    });
  });
});
