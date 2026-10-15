import { expect, test } from "./host.ts";

for (const viewport of [
  { width: 1280, height: 820 },
  { width: 800, height: 600 },
]) {
  test(`Deck selectors align and save actions never overlay editing at ${viewport.width}px`, async ({
    gui,
  }, testInfo) => {
    await gui.page.setViewportSize(viewport);
    await gui.open();
    await gui.screen("Stream Deck");
    const device = await gui.page.getByLabel("Device", { exact: true }).boundingBox();
    const page = await gui.page.getByLabel("Page", { exact: true }).boundingBox();
    expect(Math.abs(device!.width - page!.width)).toBeLessThanOrEqual(1);
    if (viewport.width === 1280) expect(Math.abs(device!.y - page!.y)).toBeLessThanOrEqual(1);
    const save = gui.page.getByRole("button", { name: "Save Layout", exact: true });
    for (const fraction of [0, 0.5, 1]) {
      await gui.page.evaluate(
        (fraction) => window.scrollTo(0, document.documentElement.scrollHeight * fraction),
        fraction,
      );
      const bar = await save.boundingBox();
      const firstKey = await gui.page.locator("[data-part=deck-key]").first().boundingBox();
      expect(bar!.y + bar!.height).toBeLessThan(firstKey!.y);
    }
    await gui.page.evaluate(() => window.scrollTo(0, 0));
    await gui.page.screenshot({ path: testInfo.outputPath(`Deck-${viewport.width}.png`), fullPage: true });
    expect(await gui.sent()).toEqual([]);
  });

  test(`Deck shared editing names its all-page scope at ${viewport.width}px`, async ({ gui }) => {
    await gui.page.setViewportSize(viewport);
    await gui.open();
    await gui.screen("Stream Deck");
    await expect(gui.page.getByText("Same position on every page.", { exact: true })).toBeVisible();
    await gui.page.getByRole("button", { name: "Edit All Pages", exact: true }).click();
    expect((await gui.last("streamdeck.shared")).Operation).toBe("press");
  });

  test(`Soundboard distinguishes an empty search from an empty library at ${viewport.width}px`, async ({ gui }) => {
    await gui.page.setViewportSize(viewport);
    await gui.open();
    await gui.screen("Soundboard");
    const search = gui.page.getByRole("searchbox", { name: "Search Clips" });
    const noMatch = gui.page.getByText("No matching clips", { exact: true });
    await search.fill("does not match any clip");
    await expect(noMatch).toBeVisible();
    await expect(gui.page.locator("[data-part=clip]:visible")).toHaveCount(0);
    await search.blur();
    await gui.update(() => {
      window.ctl("soundboard.clip-sadge").Label += " updated";
    });
    await expect(noMatch).toBeVisible();
    await expect(search).toHaveValue("does not match any clip");
    await search.fill("");
    await expect(noMatch).toBeHidden();
    expect(await gui.page.locator("[data-part=clip]:visible").count()).toBeGreaterThan(0);
    await search.blur();
    await gui.update(() => {
      window.fixture.Controls = window.fixture.Controls.filter((c) => !c.ID.startsWith("soundboard.clip-"));
    });
    await expect(
      gui.page.getByText("No clips available. Check the soundboard plugin and library folder."),
    ).toBeVisible();
    await expect(noMatch).toHaveCount(0);
    expect(await gui.sent()).toEqual([]);
  });
}
