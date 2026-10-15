import { expect, test } from "./host.ts";

test("notices group errors, contextualize VR and omit descriptive duplicates", async ({ gui }) => {
  await gui.open((fixture) => {
    fixture.PluginLabels!.audio = "Studio Audio";
    for (const item of fixture.Controls) {
      if (item.ID.startsWith("audio.normal-")) item.Status = "VR override";
      if (item.ID === "audio.mic-mute") item.Status = "Unknown";
      if (item.ID === "aec.engine") {
        item.Status = "Save failed: access denied";
        item.Available = false;
      }
    }
  });
  await gui.screen("Diagnostics");
  const notices = gui.page.locator("section", {
    has: gui.page.getByRole("heading", { name: "Current Notices", exact: true }),
  });
  await expect(notices.getByRole("heading", { name: "Studio Audio", exact: true })).toHaveCount(1);
  await expect(notices.getByText("Normal profile", { exact: true })).toHaveCount(1);
  await expect(notices.getByText("VR override", { exact: true })).toHaveCount(1);
  await expect(notices.getByText("Save failed: access denied", { exact: true })).toHaveCount(1);
  await expect(notices.getByText("Unknown", { exact: true })).not.toHaveCount(0);
  for (const text of [
    "Disabled in Windows startup apps",
    "3 held",
    "Peak worker / queue · gaps 3 · underruns 0",
    "−32 dB echo · 54 ms",
  ])
    await expect(notices.getByText(text, { exact: true })).toHaveCount(0);
  const engine = gui.page.locator("section", {
    has: gui.page.getByRole("heading", { name: "Audio Engine", exact: true }),
  });
  await expect(engine.getByText("Engine", { exact: true })).toHaveCount(1);
  await expect(engine.getByRole("button", { name: "Restart Audio Engine", exact: true })).toHaveCount(1);
  await expect(engine.getByText("Automatic Recovery", { exact: true })).toHaveCount(0);
  await expect(engine.getByText("Interface", { exact: true })).toHaveCount(0);
  expect(await gui.sent()).toEqual([]);
});

test("copy notices keeps full errors and feedback across polls", async ({ gui }) => {
  await gui.open();
  await gui.screen("Diagnostics");
  await gui.update(() => {
    window.ctl("aec.engine").Status = "Write failed\nC:/settings: access denied";
  });
  const copy = gui.page.getByRole("button", { name: "Copy Notices", exact: true });
  await copy.click();
  const text = await gui.page.evaluate(() => window.copied.at(-1));
  expect(text).toContain("Echo Cancellation\n");
  expect(text).toContain("Write failed\nC:/settings: access denied");
  await gui.update(() => {
    window.ctl("aec.engine").Status = "New warning";
  });
  await expect(copy).toBeFocused();
  await expect(gui.page.getByText("Copied", { exact: true })).toBeVisible();
  expect(await gui.sent()).toEqual([]);
});

test("copy notices uses the existing selection fallback", async ({ gui }) => {
  await gui.open();
  await gui.screen("Diagnostics");
  await gui.page.evaluate(() => {
    navigator.clipboard.writeText = async () => {
      throw new Error("permission denied");
    };
    document.execCommand = (command) => {
      if (command !== "copy") return false;
      window.copied.push((document.activeElement as HTMLTextAreaElement).value);
      return true;
    };
  });
  await gui.page.getByRole("button", { name: "Copy Notices", exact: true }).click();
  expect(await gui.page.evaluate(() => window.copied.at(-1))).toContain("Unknown");
  await expect(gui.page.getByText("Copied", { exact: true })).toBeVisible();
  await expect(gui.page.locator("textarea")).toHaveCount(0);
});

test("unrecognised unavailable warnings survive filtering and clear live", async ({ gui }) => {
  await gui.open((fixture) => {
    for (const item of fixture.Controls) item.Status = "";
  });
  await gui.screen("Diagnostics");
  const copy = gui.page.getByRole("button", { name: "Copy Notices", exact: true });
  await expect(copy).toBeDisabled();
  await gui.update(() => {
    const item = window.ctl("aec.status");
    item.Status = "Settings directory is read-only";
    item.Available = false;
  });
  await expect(gui.page.getByText("Settings directory is read-only", { exact: true })).toHaveCount(1);
  await expect(copy).toBeEnabled();
  await gui.update(() => {
    window.ctl("aec.status").Status = "";
  });
  await expect(copy).toBeDisabled();
  await expect(gui.page.getByText("No control notices.", { exact: true })).toBeVisible();
});
