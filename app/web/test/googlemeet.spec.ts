import { expect, test } from "./host.ts";
import { holdActions, reply } from "./action-host.ts";

test("Meet works independently of Discord and upstream mute", async ({ gui, page }) => {
  await gui.open((fixture) => {
    fixture.PluginStates.discord = "disabled";
    fixture.Enabled.discord = false;
  });
  await gui.screen("Meetings");
  const meet = page.locator("[data-part=googlemeet]");
  await expect(page.getByRole("button", { name: "Enable Discord", exact: true })).toBeVisible();
  await expect(meet.locator("[data-part=state]")).toHaveText("Brave · Tab 7");
  await gui.update(() => {
    window.ctl("audio.mic-mute").Value = "On";
    window.ctl("googlemeet.mute").Value = "Off";
  });
  await expect(meet.locator("[data-part=source-muted]")).toHaveText("Source muted");
  const mute = meet.getByRole("button", { name: "Mute", exact: true });
  await expect(mute).toHaveText("Off");
  await mute.focus();
  await mute.press("Space");
  expect((await gui.last()).ID).toBe("googlemeet.mute");
  expect(await page.evaluate(() => window.ctl("audio.mic-mute").Value)).toBe("On");
  expect((await gui.sent()).map((action) => action.Request?.ID)).toEqual(["googlemeet.mute"]);
  await gui.update(() => {
    window.ctl("audio.mic-mute").Value = "Off";
    window.ctl("audio.mic-stack").Value = "Off";
  });
  await expect(meet.locator("[data-part=source-muted]")).toHaveText("Mic Stack off");
  await gui.update(() => {
    window.fixture.PluginStates.audio = "disabled";
  });
  await expect(meet.locator("[data-part=source-muted]")).toBeHidden();
  await expect(mute).toBeEnabled();
});

test("Meet unknown and prejoin keep the call readable without claiming Off", async ({ gui, page }) => {
  await gui.open();
  await gui.screen("Meetings");
  const meet = page.locator("[data-part=googlemeet]");
  await gui.update(() => {
    window.ctl("googlemeet.call").Value = "Not in a call";
    window.ctl("googlemeet.target").Value = "";
    for (const id of ["googlemeet.mute", "googlemeet.camera"]) {
      Object.assign(window.ctl(id), { Value: "N/A", Available: false, Status: "Not in a call" });
    }
  });
  await expect(meet.locator("[data-part=state]")).toHaveText("Not in a call");
  for (const name of ["Mute", "Camera"]) {
    const control = meet.getByRole("button", { name, exact: true });
    await expect(control).toBeDisabled();
    await expect(control).toHaveText("Unavailable");
  }
  await expect(meet.getByLabel("Call", { exact: true })).toHaveValue("");
  await expect(page.locator("[data-part=discord]").getByRole("button", { name: "Mute", exact: true })).toBeEnabled();
  expect(await gui.sent()).toEqual([]);
});

test("Meet call selection needs keyboard commitment and retains the captured revision", async ({ gui, page }) => {
  await gui.open((fixture) => {
    const target = fixture.Controls.find((control) => control.ID === "googlemeet.target")!;
    target.Value = "";
    target.Options = ["", "call-7", "call-9"];
    target.OptionLabels = { "": "Select a call", "call-7": "Brave · Tab 7", "call-9": "Brave · Tab 9" };
    target.Revision = 4;
  });
  await gui.screen("Meetings");
  const call = page.locator("[data-part=googlemeet]").getByLabel("Call", { exact: true });
  await call.focus();
  await call.press("ArrowDown");
  expect(await gui.sent()).toEqual([]);
  await call.press("Escape");
  await expect(call).toHaveValue("");
  await call.focus();
  await call.press("End");
  await gui.update(() => {
    window.ctl("googlemeet.target").Revision = 5;
  });
  await call.press("Enter");
  expect(await gui.last()).toMatchObject({ ID: "googlemeet.target", Operation: "set", Value: "call-9", Revision: 4 });
  expect(await gui.sent()).toHaveLength(1);
});

test("Meet lifecycle loss replaces a focused call selector immediately", async ({ gui, page }) => {
  await gui.open();
  await gui.screen("Meetings");
  const call = page.locator("[data-part=googlemeet]").getByLabel("Call", { exact: true });
  await call.focus();
  await gui.update(() => {
    window.fixture.PluginStates.googlemeet = "failed";
    window.fixture.Plugins.googlemeet = "Bridge unavailable";
  });
  await expect(call).toHaveCount(0);
  await expect(page.locator('[data-part=plugin-gate][data-plugin="googlemeet"]')).toContainText("Bridge unavailable");
  await expect(page.locator("#content")).toBeFocused();
  await expect(page.locator("[data-part=discord]")).toBeVisible();
  expect(await gui.sent()).toEqual([]);
});

test("Meet pending controls are independent and outcomes do not invent observed state", async ({ gui, page }) => {
  await gui.open();
  await gui.screen("Meetings");
  await holdActions(gui);
  const meet = page.locator("[data-part=googlemeet]");
  const mute = meet.getByRole("button", { name: "Mute", exact: true });
  const camera = meet.getByRole("button", { name: "Camera", exact: true });
  await mute.click();
  await expect(mute).toBeDisabled();
  await expect(mute).toHaveText("On");
  await expect(camera).toBeEnabled();
  await camera.click();
  await expect(camera).toHaveText("Off");
  await expect(page.locator("[data-part=discord]").getByRole("button", { name: "Mute", exact: true })).toBeEnabled();
  await reply(gui, "googlemeet.mute", "Unknown", "Meet state not confirmed");
  await expect(mute).toBeEnabled();
  await expect(mute).toHaveText("On");
  await expect(meet.locator('[data-action-feedback="googlemeet.mute"]')).toContainText("Meet state not confirmed");
  await reply(gui, "googlemeet.camera", "Processed", "Meet state observed");
  await expect(camera).toBeEnabled();
  await expect(camera).toHaveText("Off");
  expect((await gui.sent()).map((action) => action.Request?.ID)).toEqual(["googlemeet.mute", "googlemeet.camera"]);
  await gui.update(() => {
    window.ctl("googlemeet.camera").Value = "On";
  });
  await expect(camera).toHaveText("On");
});
