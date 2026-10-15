import { expect, test } from "./host.ts";
import { holdActions, reply } from "./action-host.ts";

test("shared meeting actions stay independent of provider gates and show actual app state", async ({ gui, page }) => {
  await gui.open((fixture) => {
    fixture.PluginStates.discord = "disabled";
    fixture.Enabled.discord = false;
  });
  await gui.screen("Meetings");
  const shared = page.locator("[data-part=shared-meetings]");
  const meet = page.locator("[data-part=googlemeet]");
  await expect(page.getByRole("button", { name: "Enable Discord", exact: true })).toBeVisible();
  await expect(meet.locator("[data-part=state]")).toHaveText("Brave · Tab 7");
  await gui.update(() => {
    window.ctl("audio.mic-mute").Value = "On";
    window.ctl("googlemeet.call-mute").Value = "Off";
    window.ctl("meetings.mute").Status = "Pending";
  });
  await expect(shared.locator("[data-part=source-muted]")).toHaveText("Source muted");
  const mute = shared.getByRole("button", { name: "Mute", exact: true });
  await expect(mute).toHaveText("On");
  await expect(meet.getByRole("button", { name: "Mute", exact: true })).toHaveCount(0);
  await expect(meet.locator("[data-part=row]").filter({ hasText: "Mute" })).toContainText("Off");
  await mute.focus();
  await mute.press("Space");
  expect((await gui.last()).ID).toBe("meetings.mute");
  expect((await gui.sent()).map((action) => action.Request?.ID)).toEqual(["meetings.mute"]);
  expect(await page.evaluate(() => window.ctl("googlemeet.call-mute").Value)).toBe("Off");
  await gui.update(() => {
    window.ctl("audio.mic-mute").Value = "Off";
    window.ctl("audio.mic-stack").Value = "Off";
  });
  await expect(shared.locator("[data-part=source-muted]")).toHaveText("Mic Stack off");
  await gui.update(() => {
    window.fixture.PluginStates.audio = "disabled";
  });
  await expect(shared.locator("[data-part=source-muted]")).toBeHidden();
  await expect(mute).toBeEnabled();
});

test("unknown app readback does not erase another call's shared target", async ({ gui, page }) => {
  await gui.open();
  await gui.screen("Meetings");
  const meet = page.locator("[data-part=googlemeet]");
  await gui.update(() => {
    window.ctl("googlemeet.call").Value = "Not in a call";
    window.ctl("googlemeet.target").Value = "";
    for (const id of ["googlemeet.call-mute", "googlemeet.call-camera"]) {
      Object.assign(window.ctl(id), { Value: "N/A", Status: "Not in a call" });
    }
  });
  await expect(meet.locator("[data-part=state]")).toHaveText("Not in a call");
  await expect(meet.locator("[data-part=row]").filter({ hasText: "Mute" })).toContainText("N/A");
  await expect(meet.locator("[data-part=row]").filter({ hasText: "Camera" })).toContainText("N/A");
  await expect(meet.getByLabel("Call", { exact: true })).toHaveValue("");
  await expect(
    page.locator("[data-part=shared-meetings]").getByRole("button", { name: "Mute", exact: true }),
  ).toBeEnabled();
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

test("shared target acceptance does not invent app convergence", async ({ gui, page }) => {
  await gui.open();
  await gui.screen("Meetings");
  await holdActions(gui);
  const shared = page.locator("[data-part=shared-meetings]");
  const mute = shared.getByRole("button", { name: "Mute", exact: true });
  const camera = shared.getByRole("button", { name: "Camera", exact: true });
  await mute.click();
  await expect(mute).toBeDisabled();
  await expect(mute).toHaveText("On");
  await expect(camera).toBeEnabled();
  await camera.click();
  await expect(camera).toHaveText("Off");
  await reply(gui, "meetings.mute", "Unknown", "Mute save not confirmed");
  await expect(mute).toBeEnabled();
  await expect(shared.locator('[data-action-feedback="meetings.mute"]')).toContainText("Mute save not confirmed");
  await reply(gui, "meetings.camera", "Processed", "Camera target accepted");
  await expect(camera).toBeEnabled();
  await expect(camera).toHaveText("Off");
  expect((await gui.sent()).map((action) => action.Request?.ID)).toEqual(["meetings.mute", "meetings.camera"]);
  await gui.update(() => {
    Object.assign(window.ctl("meetings.camera"), { Value: "On", Status: "Pending" });
  });
  await expect(camera).toHaveText("On");
  const actual = page.locator("[data-part=googlemeet] [data-part=row]").filter({ hasText: "Camera" });
  await expect(actual).toContainText("Off");
  await expect(shared).toContainText("Pending");
  await gui.update(() => {
    window.ctl("googlemeet.call-camera").Value = "On";
    window.ctl("meetings.camera").Status = "";
  });
  await expect(actual).toContainText("On");
});

test("last call ends camera target while retained mute remains usable", async ({ gui, page }) => {
  await gui.open();
  await gui.screen("Meetings");
  await gui.update(() => {
    Object.assign(window.ctl("meetings.camera"), { Value: "N/A", Available: false, Status: "Not in a call" });
    window.fixture.PluginStates.discord = "disabled";
    window.fixture.PluginStates.googlemeet = "disabled";
  });
  const shared = page.locator("[data-part=shared-meetings]");
  const camera = shared.getByRole("button", { name: "Camera", exact: true });
  await expect(camera).toBeDisabled();
  await expect(camera).not.toHaveText("Off");
  await expect(shared.getByRole("button", { name: "Mute", exact: true })).toBeEnabled();
  await expect(page.getByRole("button", { name: "Enable Discord", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Enable Google Meet", exact: true })).toBeVisible();
  expect(await gui.sent()).toEqual([]);
});
