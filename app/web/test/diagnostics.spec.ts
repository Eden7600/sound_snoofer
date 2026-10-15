import { expect, test } from "./host";

test.describe("Diagnostics", () => {
  test("engine restart and start at login", async ({ gui }) => {
    await gui.open();
    await gui.screen("Diagnostics");
    const { page } = gui;
    await page.getByRole("button", { name: "Restart Audio Engine", exact: true }).click();
    expect((await gui.last()).ID).toBe("audio.engine-restart");
    await gui.settled();
    const startup = page.getByRole("button", { name: "Start at Login", exact: true });
    await expect(startup).toHaveText("Off");
    await expect(startup).toHaveAttribute("aria-pressed", "false");
    await expect(
      page.locator("[data-part=row]", { has: startup }).getByText("Disabled in Windows startup apps"),
      "startup note",
    ).toHaveCount(1);
    await startup.click();
    const r = await gui.last();
    expect([r.ID, r.Operation]).toEqual(["core.startup", "press"]);
  });
});

test.describe("Plugins", () => {
  test("enable-only list with host confirmation", async ({ gui }) => {
    await gui.open();
    await gui.screen("Plugins");
    const { page } = gui;
    await expect(page.locator("[data-part=row]"), "Plugins page shows configuration").toHaveCount(0);
    await expect(page.getByText("Hue Brightness"), "Plugins page shows plugin controls").toHaveCount(0);
    await page.getByRole("button", { name: "Enable", exact: true }).click();
    expect((await gui.sent()).at(-1)?.Plugin).toBe("media");
    await gui.update(() => {
      window.fixture.Confirmation = "Enable media and restart Snoofer?";
    });
    await page.getByRole("dialog").waitFor();
    await page.getByRole("button", { name: "Cancel", exact: true }).click();
    expect((await gui.sent()).at(-1)?.Kind).toBe("cancel");
  });
});
