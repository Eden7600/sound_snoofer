import { dot } from "./fixtures/manual";
import { expect, test } from "./host";

test.describe("Soundboard", () => {
  test.beforeEach(async ({ gui }) => {
    await gui.open();
    await gui.screen("Soundboard");
    await gui.page.getByRole("searchbox", { name: "Search Clips" }).fill("sad");
    await expect(gui.page.locator("[data-part=clip]:visible")).toHaveCount(1);
  });

  test("search filters clips and animated artwork cycles", async ({ gui }) => {
    await gui.page.waitForFunction(
      (dot) =>
        (document.querySelector("[data-part=clip]:not([hidden]) img") as HTMLImageElement | null)?.src.endsWith(dot),
      dot,
    );
  });

  test("a click plays once; a hold loops and swallows the click", async ({ gui }) => {
    const { page } = gui;
    const clip = page.locator("[data-part=clip]:visible");
    const operations = async () =>
      (await gui.sent()).map((a) => {
        if (!a.Request) throw new Error("expected a Soundboard request");
        return a.Request.ID + ":" + a.Request.Operation;
      });
    await clip.click();
    expect(await operations()).toEqual(["soundboard.clip-sadge:press"]);
    await expect(clip).toBeEnabled();
    await gui.clearSent();
    await clip.hover();
    await page.mouse.down();
    await page.waitForTimeout(700);
    await expect(clip).toBeEnabled();
    await page.mouse.up();
    await page.waitForTimeout(300);
    expect(await operations(), "hold did not loop once without a press").toEqual(["soundboard.clip-sadge:hold"]);
    await gui.update(() => {
      const sadge = window.ctl("soundboard.clip-sadge");
      sadge.Value = "Loop";
      sadge.Revision++;
    });
    await expect(clip).toHaveAttribute("data-state", "playing");
    await expect(clip.getByText("Loop", { exact: true })).toHaveCount(1);
  });
});
