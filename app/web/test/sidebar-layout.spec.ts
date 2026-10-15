import { expect, test } from "./host";

for (const viewport of [
  { width: 784, height: 562 },
  { width: 800, height: 600 },
  { width: 1280, height: 820 },
]) {
  test(`sidebar brand and navigation fit at ${viewport.width}×${viewport.height}`, async ({ gui, page }, testInfo) => {
    await page.setViewportSize(viewport);
    await gui.open();
    // Reserve the compact sidebar gutter even when this machine normally overlays scrollbars.
    await page.addStyleTag({
      content: `aside { scrollbar-gutter: ${viewport.width < 850 ? "stable" : "auto"}; scrollbar-width: auto } aside::-webkit-scrollbar { width: 17px; height: 17px }`,
    });
    await page.evaluate(() => document.fonts.ready);
    const bounds = await page.locator("aside").evaluate((sidebar: HTMLElement) => {
      const style = getComputedStyle(sidebar);
      const rect = sidebar.getBoundingClientRect();
      const image = sidebar.querySelector('[data-part="brandmark"]')!;
      const wordmark = image.parentElement!.querySelector("strong")!;
      const range = document.createRange();
      range.selectNodeContents(wordmark);
      const left = rect.left + sidebar.clientLeft + parseFloat(style.paddingLeft);
      const right = rect.left + sidebar.clientLeft + sidebar.clientWidth - parseFloat(style.paddingRight);
      return {
        gutter: sidebar.offsetWidth - sidebar.clientWidth - parseFloat(style.borderRightWidth),
        overflow: sidebar.scrollWidth - sidebar.clientWidth,
        left,
        right,
        image: { left: image.getBoundingClientRect().left, right: image.getBoundingClientRect().right },
        wordmark: { left: range.getBoundingClientRect().left, right: range.getBoundingClientRect().right },
      };
    });
    expect(bounds.gutter, "the fixture must reserve the native scrollbar width").toBe(viewport.width < 850 ? 17 : 0);
    expect(bounds.overflow, "sidebar must not need horizontal scrolling").toBeLessThanOrEqual(1);
    for (const [name, rect] of [
      ["mascot", bounds.image],
      ["wordmark", bounds.wordmark],
    ] as const) {
      expect(rect.left, `${name} starts inside the usable sidebar`).toBeGreaterThanOrEqual(bounds.left - 1);
      expect(rect.right, `${name} ends before padding and scrollbar`).toBeLessThanOrEqual(bounds.right + 1);
    }
    await page.screenshot({ path: testInfo.outputPath(`sidebar-${viewport.width}.png`) });
    const navigation = page.getByRole("navigation", { name: "Workspace" });
    await navigation.getByRole("button", { name: "Audio", exact: true }).focus();
    await page.keyboard.press("End");
    const diagnostics = navigation.getByRole("button", { name: "Diagnostics", exact: true });
    await expect(diagnostics).toBeFocused();
    await expect(page.locator("#title")).toHaveText("Diagnostics");
    expect(
      await diagnostics.evaluate((button) => {
        const sidebar = button.closest("aside")!;
        const view = sidebar.getBoundingClientRect(),
          rect = button.getBoundingClientRect();
        return rect.top >= view.top && rect.bottom <= view.top + sidebar.clientHeight;
      }),
      "last navigation item remains visible after keyboard navigation",
    ).toBe(true);
    expect(await gui.sent()).toEqual([]);
  });
}
