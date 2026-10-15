import { expect, test } from "./host.ts";

for (const viewport of [
  { width: 1280, height: 820 },
  { width: 800, height: 600 },
]) {
  test(`shared notice contains and centers Dismiss at ${viewport.width}px`, async ({ gui, page }, testInfo) => {
    await page.setViewportSize(viewport);
    await gui.open();
    const notice = page.locator("#notice");
    for (const [name, message] of [
      ["short", "Control changed; try again"],
      ["wrapped", "Could not save the selected device.\nResource " + "0123456789".repeat(18) + " is unavailable."],
    ] as const) {
      await gui.update((message) => {
        window.fixture.Notice = message;
      }, message);
      await expect(notice).toContainText(message);
      const bounds = await notice.evaluate((node) => {
        const button = node.querySelector("button")!;
        const rect = node.getBoundingClientRect(),
          control = button.getBoundingClientRect();
        const style = getComputedStyle(node);
        return {
          centerOffset: Math.abs(control.top + control.height / 2 - (rect.top + rect.height / 2)),
          topGap: control.top - rect.top - parseFloat(style.borderTopWidth),
          bottomGap: rect.bottom - control.bottom - parseFloat(style.borderBottomWidth),
          padding: parseFloat(style.paddingTop),
          overflow: node.scrollWidth - node.clientWidth,
        };
      });
      expect(bounds.centerOffset, name + ": Dismiss is vertically centered").toBeLessThanOrEqual(1);
      expect(bounds.topGap, name + ": Dismiss clears the top padding").toBeGreaterThanOrEqual(bounds.padding - 1);
      expect(bounds.bottomGap, name + ": Dismiss clears the bottom padding").toBeGreaterThanOrEqual(bounds.padding - 1);
      expect(bounds.overflow, name + ": notice does not overflow horizontally").toBeLessThanOrEqual(1);
      await gui.noHorizontalScroll();
      await page.screenshot({ path: testInfo.outputPath(`notice-${name}.png`) });
    }
    await notice.getByRole("button", { name: "Dismiss" }).click();
    await expect(notice).toBeHidden();
    expect(await gui.sent()).toEqual([]);
  });
}
