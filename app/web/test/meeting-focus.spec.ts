import { expect, test } from "./host.ts";
import { holdActions, reply } from "./action-host.ts";

test("focused call buttons retain focus and route only focused action IDs", async ({ gui, page }) => {
  await gui.open();
  await gui.screen("Meetings");
  const card = page.locator("[data-part=focused-meeting]");
  const discord = card.getByRole("button", { name: "Discord", exact: true });
  const meet = card.getByRole("button", { name: "Brave · Tab 7", exact: true });
  await expect(meet).toHaveAttribute("aria-pressed", "true");
  await expect(discord).toHaveAttribute("aria-pressed", "false");
  await expect(card.getByRole("button", { name: "Deafen", exact: true })).toBeDisabled();
  await expect(card.getByRole("button", { name: "Deafen", exact: true })).toHaveText("N/A");
  await discord.focus();
  await discord.press("Space");
  expect(await gui.last()).toMatchObject({ ID: "meetings.focus-1", Operation: "press" });
  await gui.update(() => {
    window.ctl("meetings.focus-1").Value = "On";
    window.ctl("meetings.focus-2").Value = "Off";
    Object.assign(window.ctl("meetings.deafen"), { Value: "Off", Available: true, Status: "" });
  });
  await expect(discord).toBeFocused();
  await expect(card.locator("[data-part=focused-call-name]")).toHaveText("Discord");
  await expect(meet).toHaveAttribute("aria-pressed", "false");
  await card.getByRole("button", { name: "Deafen", exact: true }).click();
  expect((await gui.last()).ID).toBe("meetings.deafen");
  await gui.settled();
  await card.getByRole("button", { name: "Share", exact: true }).click();
  expect((await gui.last()).ID).toBe("meetings.share");
  await gui.settled();
  await card.getByRole("button", { name: "Leave Call", exact: true }).click();
  expect((await gui.last()).ID).toBe("meetings.leave");
});

test("departed calls remove focus buttons and leave unsupported actions inert", async ({ gui, page }) => {
  await gui.open();
  await gui.screen("Meetings");
  const card = page.locator("[data-part=focused-meeting]");
  await card.getByRole("button", { name: "Brave · Tab 7", exact: true }).focus();
  await gui.update(() => {
    window.fixture.Controls = window.fixture.Controls.filter((item) => item.Collection !== "meetings.focus");
    for (const id of ["meetings.deafen", "meetings.share", "meetings.leave"])
      Object.assign(window.ctl(id), { Available: false, Value: "N/A", Status: "Not in a call" });
  });
  await expect(card.locator("[data-part=call-focus] button")).toHaveCount(0);
  await expect(card.locator("[data-part=focused-call-name]")).toHaveText("Not in a call");
  for (const name of ["Deafen", "Share", "Leave Call"])
    await expect(card.getByRole("button", { name, exact: true })).toBeDisabled();
  await expect(page.locator("#content")).toBeFocused();
  expect(await gui.sent()).toEqual([]);
});

test("presentation submission keeps observed Off and leaves universal controls independent", async ({ gui, page }) => {
  await gui.open();
  await gui.screen("Meetings");
  await holdActions(gui);
  const card = page.locator("[data-part=focused-meeting]");
  const share = card.getByRole("button", { name: "Share", exact: true });
  await share.click();
  await expect(share).toBeDisabled();
  await expect(
    page.locator("[data-part=shared-meetings]").getByRole("button", { name: "Mute", exact: true }),
  ).toBeEnabled();
  await reply(gui, "meetings.share", "Submitted", "Choose what to share in Meet");
  await expect(share).toBeEnabled();
  await expect(share).toHaveText("Off");
  await expect(card.locator('[data-action-feedback="meetings.share"]')).toContainText("Choose what to share in Meet");
});
