import { expect, test } from "./host";
import type { MediaView } from "../src/screens/media-types.ts";

test.describe("Media", () => {
  test.beforeEach(async ({ gui }) => {
    await gui.open();
    await gui.screen("Media");
    await gui.page.locator("[data-part=media-session]").first().waitFor();
  });

  test("sessions: the focused one offers its transport", async ({ gui }) => {
    const { page } = gui;
    expect(await page.locator("[data-part=media-session] h3").allTextContents()).toEqual(["Video A", "Song"]);
    const [tab, song] = [
      page.locator("[data-part=media-session]").nth(0),
      page.locator("[data-part=media-session]").nth(1),
    ];
    await expect(tab).toHaveAttribute("data-focused", "true");
    await expect(tab.getByRole("button", { name: "Next" }), "focused card lacks Next").toBeVisible();
    await expect(song.getByRole("button", { name: "Next" }), "unfocused card shows Next").toBeHidden();
    await expect(
      page.locator("[data-part=media-session] button", { hasText: "Focus" }),
      "Focus button still offered",
    ).toHaveCount(0);
  });

  test("play, seek and the slider follows the session", async ({ gui }) => {
    const { page } = gui;
    const song = page.locator("[data-part=media-session]").nth(1);
    await song.locator("[data-part=play]").click();
    let r = await gui.last();
    expect([r.ID, r.Operation]).toEqual(["nowplaying.s-spotify", "press"]);
    await gui.settled();
    await song.locator("[data-part=seek]").fill("90000");
    r = await gui.last();
    expect([r.ID, r.Operation, r.Value]).toEqual(["nowplaying.s-spotify", "set", "90000"]);
    // After a seek the slider follows the session again, even while focused.
    await gui.update(() => {
      const view = window.ctl("nowplaying.status").ViewData as MediaView | undefined;
      const song = view?.Sessions?.[1];
      if (!song) throw new Error("missing Spotify fixture session");
      song.PositionMs = 120000;
    });
    await expect(page.locator("[data-part=seek]").nth(1)).toHaveValue("120000");
  });

  test("a playing session advances between polls", async ({ gui }) => {
    const seek = gui.page.locator("[data-part=media-session]").nth(0).locator("[data-part=seek]");
    const first = Number(await seek.inputValue());
    await expect
      .poll(async () => Number(await seek.inputValue()), { timeout: 3000 })
      .toBeGreaterThanOrEqual(first + 1000);
  });

  test("mute and the extension card", async ({ gui }) => {
    const { page } = gui;
    await page.locator("[data-part=media-session]").nth(0).locator("[data-part=mute]").click();
    expect((await gui.last()).ID).toBe("nowplaying.mute");
    const extension = page.locator("[data-part=extension]");
    await expect(extension).toContainText(/Brave connected · 1 with media · extension 1\.0\.0/);
    await expect(extension).toContainText("Update the Snoofer Media extension in Firefox");
    await expect(extension.getByRole("button"), "extension card offers export or token actions").toHaveCount(0);
  });
});
