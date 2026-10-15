import { expect, test } from "./host";

test.describe("Meetings", () => {
  test.beforeEach(async ({ gui }) => {
    await gui.open();
    await gui.screen("Meetings");
    await gui.page.locator("[data-part=discord]").waitFor();
  });

  test("shows the call and sends its commands", async ({ gui }) => {
    const { page } = gui;
    const discord = page.locator("[data-part=discord]");
    await expect(page.locator("[data-part=camera]"), "Camera card still shown").toHaveCount(0);
    await expect(discord.locator("[data-part=state]")).toHaveText("General");
    await expect(page.locator("[data-part=discord-setup]"), "setup hint while configured").toBeHidden();
    await expect(page.locator("[data-part=discord-connect]"), "Connect while connected").toBeHidden();
    await page.locator("[data-part=shared-meetings]").getByRole("button", { name: "Mute", exact: true }).click();
    expect((await gui.last()).ID).toBe("meetings.mute");
    await gui.settled();
    await discord.getByRole("button", { name: "Leave Call", exact: true }).click();
    expect((await gui.last()).ID).toBe("discord.leave");
  });

  test("setup hint and enabling the plugin", async ({ gui }) => {
    const { page } = gui;
    await gui.update(() => {
      const mute = window.ctl("discord.call-mute");
      mute.Available = false;
      mute.Status = "Setup needed";
    });
    await page.locator("[data-part=discord-setup]").waitFor();
    await gui.update(() => {
      const mute = window.ctl("discord.call-mute");
      mute.Available = true;
      mute.Status = "";
      window.fixture.Enabled.discord = false;
      window.fixture.PluginStates.discord = "disabled";
    });
    await page.getByRole("button", { name: "Enable Discord", exact: true }).click();
    const action = (await gui.sent()).at(-1);
    if (!action) throw new Error("expected Discord enable action");
    expect([action.Kind, action.Plugin, action.Enable]).toEqual(["selection", "discord", true]);
    await gui.update(() => {
      window.fixture.Enabled.discord = true;
      window.fixture.PluginStates.discord = "running";
    });
    await page.locator("[data-part=discord]").waitFor();
  });
});
