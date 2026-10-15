import { expect, test, type Gui } from "./host";
import type { AppAudioView } from "../src/screens/appaudio-types.ts";

const edit = async (gui: Gui): Promise<unknown> => JSON.parse((await gui.last("appaudio.edit")).Value);

// Accept the last edit, as the host does after saving it.
const accept = (gui: Gui) =>
  gui.update(() => {
    window.ctl("appaudio.edit").Revision++;
  });

test.describe("App Audio", () => {
  test.beforeEach(async ({ gui }) => {
    await gui.open();
    await gui.screen("App Audio");
    await gui.page.locator("[data-part=app]").first().waitFor();
  });

  test("lists picked and heard apps", async ({ gui }) => {
    expect(await gui.page.locator("[data-part=app] h3").allTextContents()).toEqual([
      "Discord",
      "Google Chrome",
      "Spotify",
    ]);
    await expect(gui.page.getByRole("slider", { name: "Spotify volume" }), "closed app adjustable").toBeHidden();
  });

  test("recent window accepts only valid minutes", async ({ gui }) => {
    const { page } = gui;
    const recent = page.getByRole("spinbutton", { name: "Recent Window in Minutes" });
    await expect(recent).toHaveValue("5");
    await recent.fill("30");
    await page.keyboard.press("Enter");
    expect(await edit(gui)).toEqual({ op: "recent", app: "recent", value: "30" });
    await gui.update(() => {
      const status = window.ctl("appaudio.status");
      const view = status.ViewData as AppAudioView | undefined;
      if (!view) throw new Error("missing App Audio fixture view");
      view.RecentMinutes = 30;
      status.Revision++;
      window.ctl("appaudio.edit").Revision++;
    });
    await expect(page.locator("[data-part=recent]")).toHaveValue("30");
    const count = (await gui.sent()).length;
    await recent.fill("0");
    await page.keyboard.press("Enter");
    expect((await gui.sent()).length, "invalid window sent").toBe(count);
    await expect(recent).toHaveValue("30");
  });

  test("volume, mute and pin", async ({ gui }) => {
    const { page } = gui;
    await page.getByRole("slider", { name: "Discord volume" }).fill("60");
    let r = await gui.last();
    expect([r.ID, r.Operation, r.Value]).toEqual(["appaudio.app-discord", "set", "60"]);
    await gui.settled();
    await page.locator("[data-part=app]").nth(1).locator("[data-part=mute]").click();
    r = await gui.last();
    expect([r.ID, r.Operation]).toEqual(["appaudio.app-chrome", "press"]);
    await gui.settled();
    await page.getByRole("button", { name: "Pin Google Chrome", exact: true }).click();
    expect(await edit(gui)).toEqual({ op: "pick", app: "Google Chrome", value: "" });
  });

  test("separate undoes a naming rule; hide hides an app", async ({ gui }) => {
    const { page } = gui;
    const discord = page.locator("[data-part=app]").first();
    await discord.locator("summary").click();
    await expect(discord.locator("[data-part=naming]").getByText("updater.exe")).toHaveCount(1);
    await page.getByRole("button", { name: "Separate updater.exe", exact: true }).click();
    expect(await edit(gui)).toEqual({ op: "separate", app: "Discord", value: "(^|\\\\)updater\\.exe$" });
    await accept(gui);
    await gui.settled();
    const chrome = page.locator("[data-part=app]").nth(1);
    await chrome.locator("summary").click();
    await chrome.locator("[data-part=hide]").click();
    expect(await edit(gui)).toEqual({ op: "hide", app: "Google Chrome", value: "" });
  });

  test("exclusions: include, exclude and unhide", async ({ gui }) => {
    const { page } = gui;
    const excluded = page.locator("[data-part=excluded]");
    await expect(excluded).toContainText("huesync.exeHiding Hue Sync");
    await excluded.getByRole("button", { name: "Remove huesync.exe", exact: true }).click();
    expect(await edit(gui)).toEqual({ op: "include", app: "huesync.exe", value: "huesync.exe" });
    await accept(gui);
    await gui.settled();
    await excluded.getByRole("textbox", { name: "Program to Exclude" }).fill("steam.exe");
    await excluded.locator("[data-part=exclude]").click();
    expect(await edit(gui)).toEqual({ op: "exclude", app: "steam.exe", value: "steam.exe" });
    await accept(gui);
    await gui.settled();
    await excluded.getByRole("button", { name: "Unhide Launcher", exact: true }).click();
    expect(await edit(gui)).toEqual({ op: "unhide", app: "Launcher", value: "" });
  });
});
