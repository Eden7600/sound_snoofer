import { expect, test, type Gui } from "./host.ts";

declare global {
  interface Window {
    recoveryProbe: { calls: number[]; fail: boolean };
  }
}

test.beforeEach(async ({ page }) => {
  await page.clock.install();
  await page.clock.pauseAt(new Date(Date.now() + 1000));
});

async function trackPolls(gui: Gui) {
  await gui.update(() => {
    const desktop = window.go?.app?.Desktop;
    if (!desktop) throw new Error("fixture Desktop missing");
    const state = desktop.State.bind(desktop);
    const probe = (window.recoveryProbe = { calls: [] as number[], fail: true });
    desktop.State = async (...args) => {
      probe.calls.push(Date.now());
      if (probe.fail) throw new Error("fixture host disconnected");
      return state(...args);
    };
  });
}

for (const size of [
  { width: 1280, height: 820 },
  { width: 800, height: 600 },
]) {
  test(`connection banner preserves state and action feedback at ${size.width}px`, async ({ gui }) => {
    await gui.page.setViewportSize(size);
    await gui.open((fixture) => {
      fixture.Notice = "Saving the selected device failed";
    });
    await trackPolls(gui);
    const { page } = gui;
    await page.clock.runFor(220);
    const banner = page.locator("#connection");
    await expect(banner).toBeVisible();
    await expect(banner).toHaveAttribute("role", "status");
    await expect(banner).toContainText("Reconnecting to Snoofer");
    await expect(page.locator("#notice")).toContainText("Saving the selected device failed");
    await expect(page.getByRole("heading", { name: "Live Controls" })).toBeVisible();
    const header = await page.locator("main > header").boundingBox();
    const bounds = await banner.boundingBox();
    const notice = await page.locator("#notice").boundingBox();
    expect(
      header && bounds && notice && header.y + header.height <= bounds.y && bounds.y + bounds.height <= notice.y,
    ).toBeTruthy();
    await gui.noHorizontalScroll();
    await page.screenshot({ path: test.info().outputPath("connection.png"), fullPage: true });
    await gui.screen("Soundboard");
    await expect(banner).toBeVisible();
    await gui.update(() => {
      window.recoveryProbe.fail = false;
    });
    await page.clock.runFor(400);
    await expect(banner).toBeHidden();
    await expect(page.locator("#notice")).toContainText("Saving the selected device failed");
    expect(await gui.sent()).toEqual([]);
  });
}

test("failed polls back off to five seconds and recover the normal cadence", async ({ gui }) => {
  await gui.open();
  await trackPolls(gui);
  await gui.page.clock.runFor(22000);
  const calls = await gui.page.evaluate(() => window.recoveryProbe.calls);
  expect(calls.length).toBeGreaterThanOrEqual(8);
  expect(calls.length).toBeLessThanOrEqual(10);
  expect(calls.slice(1).map((at, i) => at - calls[i]!)).toEqual(
    [200, 400, 800, 1600, 3200, 5000, 5000, 5000].slice(0, calls.length - 1),
  );
  await gui.update(() => {
    window.recoveryProbe.fail = false;
  });
  await gui.page.clock.runFor(5200);
  await expect(gui.page.locator("#connection")).toBeHidden();
  await gui.update(() => {
    window.recoveryProbe.calls = [];
  });
  await gui.page.clock.runFor(600);
  const recovered = await gui.page.evaluate(() => window.recoveryProbe.calls);
  expect(recovered.length).toBeGreaterThanOrEqual(5);
  expect(recovered.slice(1).map((at, i) => at - recovered[i]!)).toEqual(Array(recovered.length - 1).fill(100));
  await gui.screen("Plugins");
  await gui.page.clock.runFor(250);
  await gui.update(() => {
    window.recoveryProbe.calls = [];
  });
  await gui.page.clock.runFor(600);
  const slower = await gui.page.evaluate(() => window.recoveryProbe.calls);
  expect(slower.slice(1).map((at, i) => at - slower[i]!)).toEqual(Array(slower.length - 1).fill(200));
});

test("actions refused during recovery are never queued or replayed", async ({ gui }) => {
  await gui.open();
  await trackPolls(gui);
  await gui.page.clock.runFor(220);
  await gui.page.locator("[data-part=tape]").getByRole("button", { name: "Record", exact: true }).click();
  await expect(gui.page.locator('[data-action-feedback="audio.record-toggle"]:visible')).toContainText(
    "Host disconnected; action not sent",
  );
  expect(await gui.sent()).toEqual([]);
  await gui.update(() => {
    window.recoveryProbe.fail = false;
  });
  await gui.page.clock.runFor(600);
  await expect(gui.page.locator("#connection")).toBeHidden();
  await expect(gui.page.locator('[data-action-feedback="audio.record-toggle"]:visible')).toContainText(
    "Host disconnected; action not sent",
  );
  expect(await gui.sent()).toEqual([]);
});

test("an uncertain one-shot failure survives connection recovery without replay", async ({ gui }) => {
  await gui.open();
  await gui.update(() => {
    const desktop = window.go?.app?.Desktop;
    if (!desktop) throw new Error("fixture Desktop missing");
    desktop.Send = async (action) => {
      window.sent.push(action);
      throw new Error("Recording result unavailable");
    };
  });
  await gui.page.locator("[data-part=tape]").getByRole("button", { name: "Record", exact: true }).click();
  await expect(gui.page.locator('[data-action-feedback="audio.record-toggle"]:visible')).toContainText(
    "Recording result unavailable",
  );
  await trackPolls(gui);
  await gui.page.clock.runFor(220);
  await expect(gui.page.locator("#connection")).toBeVisible();
  await gui.update(() => {
    window.recoveryProbe.fail = false;
  });
  await gui.page.clock.runFor(600);
  await expect(gui.page.locator("#connection")).toBeHidden();
  await expect(gui.page.locator('[data-action-feedback="audio.record-toggle"]:visible')).toContainText(
    "Recording result unavailable",
  );
  expect(await gui.sent()).toHaveLength(1);
});

test("render errors do not claim connection loss or stop polling", async ({ gui }) => {
  await gui.open();
  await trackPolls(gui);
  await gui.page.clock.runFor(220);
  await gui.update(() => {
    window.recoveryProbe.fail = false;
    const summary = document.querySelector<HTMLElement>("#summary");
    if (!summary) throw new Error("summary missing");
    const replace = summary.replaceChildren.bind(summary);
    summary.replaceChildren = () => {
      summary.replaceChildren = replace;
      throw new Error("fixture render failed");
    };
  });
  await gui.page.clock.runFor(600);
  await expect(gui.page.locator("#connection")).toBeHidden();
  await expect(gui.page.locator("#notice")).toContainText("fixture render failed");
  const before = await gui.page.evaluate(() => window.recoveryProbe.calls.length);
  await gui.page.clock.runFor(400);
  expect(await gui.page.evaluate(() => window.recoveryProbe.calls.length)).toBeGreaterThan(before);
});
