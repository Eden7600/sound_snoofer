import { expect, test } from "./host";

test("sidebar keyboard navigation visits every screen without dispatch", async ({ gui }) => {
  await gui.open();
  const { page } = gui;
  const navigation = page.getByRole("navigation", { name: "Workspace" });
  const screens = await navigation.getByRole("button").allTextContents();
  await navigation.getByRole("button", { name: "Audio", exact: true }).focus();
  for (const name of screens.slice(1)) {
    await page.keyboard.press("ArrowDown");
    await expect(navigation.getByRole("button", { name, exact: true })).toBeFocused();
    await expect(page.locator("#title")).toHaveText(name);
  }
  await page.keyboard.press("Home");
  await expect(navigation.getByRole("button", { name: "Audio", exact: true })).toBeFocused();
  await expect(page.locator("#title")).toHaveText("Audio");
  await page.keyboard.press("End");
  await expect(page.locator("#title")).toHaveText("Diagnostics");
  await page.keyboard.press("ArrowUp");
  await expect(page.locator("#title")).toHaveText("Third-Party Apps");
  expect(await gui.sent(), "navigation must not write settings").toEqual([]);
});

test("deck keyboard browsing, selection and text commitment are separate", async ({ gui }) => {
  await gui.open();
  await gui.screen("Stream Deck");
  const { page } = gui;
  await page.getByRole("button", { name: /^Key 1: / }).focus();
  await page.keyboard.press("ArrowDown");
  await expect(page.locator(":focus")).toHaveAttribute("aria-label", /^Key 10:/);
  expect(await gui.sent()).toEqual([]);
  await page.keyboard.press("Enter");
  expect(await gui.last("streamdeck.slot")).toMatchObject({ Operation: "set", Value: "Key 10" });
  await gui.settled();
  await gui.clearSent();

  const name = page.getByLabel("Name", { exact: true });
  const original = await name.inputValue();
  await name.fill("Discard this draft");
  await name.press("Escape");
  await expect(name).toHaveValue(original);
  expect(await gui.sent(), "Escape must discard without dispatch").toEqual([]);
  await name.fill("Keyboard Studio");
  expect(await gui.sent(), "typing must not dispatch").toEqual([]);
  await name.press("Enter");
  expect(await gui.last("streamdeck.name")).toMatchObject({ Operation: "set", Value: "Keyboard Studio" });
  expect((await gui.sent()).length).toBe(1);
});

for (const [keys, outcome] of [
  [["Escape"], "cancel"],
  [["Enter"], "cancel"],
  [["Tab", "Enter"], "confirm"],
] as const) {
  test(`host confirmation by keyboard: ${keys.join(" then ")}`, async ({ gui }) => {
    await gui.open();
    await gui.screen("Plugins");
    await gui.update(() => {
      window.fixture.Confirmation = "Enable media and restart Snoofer?";
    });
    const { page } = gui;
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();
    await expect(dialog.getByRole("button", { name: "Cancel", exact: true })).toBeFocused();
    for (const key of keys) await page.keyboard.press(key);
    await expect.poll(() => gui.sent()).toEqual([{ Kind: outcome }]);
    await gui.update(() => {
      window.fixture.Confirmation = "";
    });
    await expect(dialog).toBeHidden();
    await expect(page.locator("#title")).toHaveText("Plugins");
  });
}

test("a dismissed host notice stays dismissed across screens and polling", async ({ gui }) => {
  await gui.open((fixture) => {
    fixture.Notice = "Control changed; try again";
  });
  const notice = gui.page.getByRole("alert");
  await expect(notice).toContainText("Control changed; try again");
  await notice.getByRole("button", { name: "Dismiss" }).press("Enter");
  await expect(notice).toBeHidden();
  await gui.screen("Soundboard");
  await gui.screen("Diagnostics");
  await expect(notice).toBeHidden();
  await gui.update(() => {
    window.fixture.Notice = "A different action failed";
  });
  await expect(notice).toContainText("A different action failed");
  expect(await gui.sent()).toEqual([]);
});

test("an edit from an older list is sent as stale instead of retargeted", async ({ gui }) => {
  await gui.open();
  await gui.screen("Routing");
  const pattern = gui.page.locator("[data-part=list-playback]").getByLabel("Playback 2 pattern");
  await pattern.fill("(?i)keyboard");
  const original = await gui.page.evaluate(() => window.ctl("audio.priority-edit").Revision);
  expect(original).toBeGreaterThan(0);
  await gui.update(() => {
    window.ctl("audio.priority-edit").Revision += 100;
    window.ctl("audio.priority-edit").Value = "newer configuration";
    window.ctl("audio.priority-edit").Status = "Another surface changed priorities";
  });
  await expect(gui.page.locator("[data-part=priority-error]")).toContainText("Another surface changed priorities");
  await pattern.press("Enter");
  // Revision zero is the explicit stale sentinel used by listEdit, never a live revision.
  expect(await gui.last("audio.priority-edit")).toMatchObject({ Revision: 0 });
});
