import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "./host.ts";
import type { Fixture } from "./types.ts";
import type { PriorityView } from "../src/screens/routing-types.ts";

function neutral(fixture: Fixture) {
  for (const control of fixture.Controls) {
    delete control.Attention;
    if (control.Connection) control.Connection.State = "off";
    if (control.ID === "audio.priorities") {
      const view = control.ViewData as PriorityView;
      for (const list of Object.values(view.Lists || {})) for (const entry of list || []) entry.Matches = [];
    }
  }
}

for (const [source, destination] of [
  ["audio.health", "Audio"],
  ["aec.status", "Audio"],
  ["audio.slot-1", "Routing"],
  ["soundboard.status", "Soundboard"],
  ["appaudio.status", "App Audio"],
  ["nowplaying.status", "Media"],
  ["streamdeck.status", "Stream Deck"],
]) {
  test(`${source} marker navigates by keyboard and clears on recovery`, async ({ gui, page }) => {
    await gui.open(neutral);
    const nav = page
      .getByRole("navigation", { name: "Workspace" })
      .getByRole("button", { name: destination, exact: true });
    await gui.update((source) => {
      window.ctl(source!).Attention = "critical";
    }, source);
    await expect(nav).toHaveAttribute("data-attention", "critical");
    await expect(nav).toHaveAccessibleName(destination!);
    await expect(nav).toHaveAccessibleDescription("Problem");
    await expect(nav.locator('[data-part="nav-attention"]')).toHaveCount(1);
    await nav.focus();
    await page.keyboard.press("Enter");
    await expect(page.locator("#title")).toHaveText(destination!);
    expect(await gui.sent()).toEqual([]);
    await gui.update((source) => {
      delete window.ctl(source!).Attention;
    }, source);
    await expect(nav).not.toHaveAttribute("data-attention", /.+/);
    await expect(nav).toHaveAccessibleDescription("");
    await expect(nav.locator('[data-part="nav-attention"]')).toHaveCount(0);
  });
}

test("failure markers lead to their visible recovery and leave the header transient-only", async ({ gui, page }) => {
  await gui.open((fixture) => {
    neutral(fixture);
    fixture.PluginStates.nowplaying = "failed";
    fixture.Plugins.nowplaying = "Media session service failed";
  });
  const nav = page.getByRole("navigation", { name: "Workspace" });
  await expect(nav.getByRole("button", { name: "Media", exact: true })).toHaveAttribute("data-attention", "critical");
  await expect(nav.getByRole("button", { name: "Plugins", exact: true })).toHaveAttribute("data-attention", "critical");
  await gui.screen("Media");
  await expect(page.locator("[data-part=plugin-gate]")).toContainText("Media session service failed");
  await page.locator("[data-part=plugin-gate]").getByRole("button", { name: "Plugins", exact: true }).click();
  await expect(page.getByRole("button", { name: "Retry Plugins", exact: true })).toBeEnabled();
  await expect(page.locator("#summary")).not.toContainText(/Problem|Needs attention|Media session/);
  expect(await gui.sent()).toEqual([]);
});

test("stale connection attention advances on unchanged polls and historical errors clear", async ({ gui, page }) => {
  await page.clock.install();
  await page.clock.pauseAt(new Date(Date.now() + 1000));
  await gui.open(neutral);
  await gui.update(() => {
    window.liveMeters = false;
    const conn = window.ctl("hue.app-bridge").Connection!;
    Object.assign(conn, {
      State: "connected",
      LastActivity: new Date().toISOString(),
      LastError: "Failed earlier",
      Details: [{ Label: "Interval", Value: "1s" }],
    });
  });
  await page.clock.runFor(200);
  const nav = page.getByRole("navigation", { name: "Workspace" });
  await expect(nav.locator("[data-attention]")).toHaveCount(0);
  const unchanged = await page.evaluate(() => window.transport.unchanged);
  await page.clock.runFor(3000);
  await expect(nav.getByRole("button", { name: "Lights", exact: true })).toHaveAttribute("data-attention", "attention");
  await expect(nav.getByRole("button", { name: "Third-Party Apps", exact: true })).toHaveAttribute(
    "data-attention",
    "attention",
  );
  expect(await page.evaluate(() => window.transport.unchanged)).toBeGreaterThan(unchanged);
  await gui.update(() => {
    window.ctl("hue.app-bridge").Connection!.LastActivity = new Date().toISOString();
  });
  await page.clock.runFor(200);
  await expect(nav.locator("[data-attention]")).toHaveCount(0);
  expect(await gui.sent()).toEqual([]);
});

for (const viewport of [
  { width: 1280, height: 820 },
  { width: 800, height: 600 },
]) {
  test(`attention markers and lifecycle gates fit and are accessible at ${viewport.width}`, async ({
    gui,
    page,
  }, testInfo) => {
    await page.setViewportSize(viewport);
    await gui.open((fixture) => {
      neutral(fixture);
      for (const id of Object.keys(fixture.PluginStates)) fixture.PluginStates[id] = "failed";
      fixture.Plugins.audio = "The configured audio engine could not start. Check the selected driver.";
      const connection = fixture.Controls.find((control) => control.ID === "hue.app-bridge")!;
      connection.ID = "custom.peer";
      connection.Connection!.State = "error";
    });
    await expect(page.locator('[data-screen="apps"]')).toHaveAttribute("data-attention", "critical");
    const issues = await page.locator("[data-screen]").evaluateAll((buttons) =>
      buttons.flatMap((button) => {
        const marker = button.querySelector('[data-part="nav-attention"]');
        const b = button.getBoundingClientRect(),
          m = marker?.getBoundingClientRect();
        return button.scrollWidth > button.clientWidth + 1 ||
          b.right > (button.closest("aside")?.getBoundingClientRect().right || innerWidth) ||
          (m && (m.left < b.left || m.right > b.right))
          ? [button.textContent]
          : [];
      }),
    );
    expect(issues).toEqual([]);
    const accessibility = await new AxeBuilder({ page })
      .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"])
      .analyze();
    expect(accessibility.violations).toEqual([]);
    await page.screenshot({ path: testInfo.outputPath(`attention-${viewport.width}.png`), fullPage: true });
    await page.locator('[data-screen="apps"]').scrollIntoViewIfNeeded();
    await page.screenshot({ path: testInfo.outputPath(`attention-bottom-${viewport.width}.png`), fullPage: true });
    expect(await gui.sent()).toEqual([]);
  });
}
