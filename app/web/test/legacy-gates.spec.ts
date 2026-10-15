import { expect, test } from "./host.ts";

for (const absent of ["omitted", "null"] as const) {
  test(`legacy ${absent} lifecycle map renders native fixture controls and keeps revisions`, async ({ gui, page }) => {
    await gui.open((fixture) => {
      if (absent === "omitted") delete (fixture as { PluginStates?: unknown }).PluginStates;
      else Object.assign(fixture, { PluginStates: null });
      fixture.Controls = fixture.Controls.filter((control) =>
        ["audio.health", "hue.status", "hue.sync"].includes(control.ID),
      );
      fixture.Plugins = { hue: "Failure wording is not lifecycle" };
      fixture.Enabled = { hue: true };
      const sync = fixture.Controls.find((control) => control.ID === "hue.sync");
      if (!sync) throw new Error("Hue Sync missing");
      sync.Revision = 9;
      sync.Value = "Off";
    });
    await expect(page.locator("[data-part=plugin-gate]")).toHaveCount(0);
    await gui.screen("Lights");
    await expect(page.locator("[data-part=plugin-gate]")).toHaveCount(0);
    await page.getByRole("button", { name: "Start Sync", exact: true }).click();
    expect(await gui.last()).toMatchObject({ ID: "hue.sync", Revision: 9 });
    await gui.screen("Plugins");
    await expect(page.getByRole("button", { name: "Retry Plugins", exact: true })).toBeDisabled();
    await gui.screen("Third-Party Apps");
    await expect(page.locator("[data-part=not-monitored]")).toHaveCount(0);
  });
}

test("legacy presence and saved enablement distinguish off, starting and missing without parsing text", async ({
  gui,
  page,
}) => {
  await gui.open((fixture) => {
    delete (fixture as { PluginStates?: unknown }).PluginStates;
    fixture.Controls = [];
    fixture.Plugins = { audio: "Running", hue: "Disabled" };
    fixture.Enabled = { audio: false, hue: true };
  });
  await expect(page.locator("[data-part=plugin-gate]")).toContainText("Off");
  await gui.screen("Lights");
  await expect(page.locator("[data-part=plugin-gate]")).toContainText("Starting");
  await gui.screen("Media");
  await expect(page.locator("[data-part=plugin-gate]")).toContainText("Not part of this build");
  await gui.screen("Plugins");
  await expect(page.getByRole("button", { name: "Retry Plugins", exact: true })).toBeDisabled();
  expect(await gui.sent()).toEqual([]);
});

test("a supplied empty lifecycle map overrides ready publications", async ({ gui, page }) => {
  await gui.open((fixture) => {
    fixture.PluginStates = {};
  });
  await expect(page.locator("[data-part=plugin-gate]")).toContainText("Not part of this build");
  await expect(page.locator("[data-part=plugin-gate]").getByRole("button")).toHaveCount(0);
  expect(await gui.sent()).toEqual([]);
});
