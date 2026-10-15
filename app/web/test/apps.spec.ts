import { expect, test } from "./host";

test.describe("Third-Party Apps", () => {
  test.beforeEach(async ({ gui }) => {
    await gui.open();
    await gui.screen("Third-Party Apps");
    await gui.page.getByRole("heading", { name: "Hue", exact: true }).waitFor();
  });

  test("cards summarise each connection", async ({ gui }) => {
    const { page } = gui;
    await expect(page.locator("[data-part=app-card]")).toHaveCount(4);
    expect(await page.locator("[data-part=summary] span").allTextContents()).toEqual(["1 OK", "0 Attention", "1 Problems", "2 Idle"]);
    const voicemeeter = page.locator("[data-part=app-card]", { hasText: "Voicemeeter" });
    await expect(voicemeeter, "required peer down must be critical").toHaveAttribute("data-tone", "critical");
    expect(await voicemeeter.locator("[data-part=app-details] dt").allTextContents(), "Required stays internal").toEqual(["Edition", "Interval"]);
    const bridge = page.locator("[data-part=app-card]", { hasText: "Hue Bridge" });
    await expect(bridge.locator("[data-part=app-times]")).toHaveText(/^Since 10m ago · Last activity (just now|\d+s ago)$/);
    await expect(bridge.locator("[data-part=app-error]")).toHaveText(/^Last error: event stream ended: EOF · 11m ago$/);
    await expect(page.locator("[data-part=app-card]", { hasText: "callback" }).locator("[data-part=app-times]"), "zero times shown").toBeHidden();
    expect(await page.locator("[data-part=not-monitored] li").allTextContents()).toEqual(["media — Disabled"]);
  });

  test("copies details", async ({ gui }) => {
    const { page } = gui;
    const bridge = page.locator("[data-part=app-card]", { hasText: "Hue Bridge" });
    await bridge.getByRole("button", { name: "Copy Details" }).click();
    await page.waitForFunction(() => window.copied.length === 1);
    const copied = await page.evaluate(() => window.copied[0]!);
    expect(copied).toMatch(/^Hue Bridge \(Hue\)\nState: Connected\nEndpoint: 172\.16\.102\.3\nSince: \d{4}-/);
    expect(copied).toContain("Bridge ID: 001788fffe2490e0");
    await page.getByRole("button", { name: "Copy All", exact: true }).click();
    await page.waitForFunction(() => window.copied.length === 2);
    expect((await page.evaluate(() => window.copied[1]!)).split("\n\n")).toHaveLength(4);
  });
});
