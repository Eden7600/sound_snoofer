import { expect, test, type Gui } from "./host.ts";

function normal(gui: Gui) {
  return gui.page.locator("section", { has: gui.page.getByRole("heading", { name: "Normal", exact: true }) });
}

test.describe("Live dropdown commitment", () => {
  test.beforeEach(async ({ gui }) => {
    await gui.open();
  });

  test("Processing browsing waits for Enter and does not repeat the same choice", async ({ gui }) => {
    const processing = normal(gui).getByLabel("Processing");
    await processing.focus();
    await processing.press("ArrowDown");
    await expect(processing).toHaveValue("element");
    expect(await gui.sent()).toEqual([]);
    await processing.press("Enter");
    expect(await gui.last("audio.normal-mode")).toMatchObject({ Operation: "set", Value: "element" });
    await gui.settled();
    await expect(processing).not.toBeFocused();
    await processing.selectOption("element");
    expect(await gui.sent()).toHaveLength(1);
  });

  for (const [key, value] of [
    ["ArrowUp", "off"],
    ["ArrowLeft", "off"],
    ["ArrowRight", "post"],
    ["Home", "off"],
    ["End", "post"],
    ["PageUp", "off"],
    ["PageDown", "post"],
    ["o", "off"],
  ] as const) {
    test(`Monitor ${key} stages its choice`, async ({ gui }) => {
      const monitor = normal(gui).getByLabel("Monitor");
      await monitor.focus();
      await monitor.press(key);
      await expect(monitor).toHaveValue(value);
      expect(await gui.sent()).toEqual([]);
      await monitor.press("Enter");
      expect(await gui.last("audio.normal-monitor")).toMatchObject({ Operation: "set", Value: value });
      expect(await gui.sent()).toHaveLength(1);
    });
  }

  for (const cancel of ["Escape", "Tab"]) {
    test(`${cancel} cancels closed Monitor browsing`, async ({ gui }) => {
      const monitor = normal(gui).getByLabel("Monitor");
      await monitor.focus();
      await monitor.press("End");
      await monitor.press(cancel);
      await expect(monitor).toHaveValue("pre");
      await expect(monitor).not.toBeFocused();
      expect(await gui.sent()).toEqual([]);
    });
  }

  test("non-audio typeahead also waits for confirmation", async ({ gui }) => {
    const strength = gui.page.locator("[data-part=echo]").getByLabel("Strength");
    await strength.focus();
    await strength.press("g");
    await expect(strength).toHaveValue("gentle");
    expect(await gui.sent()).toEqual([]);
    await strength.press("Enter");
    expect(await gui.last("aec.strength")).toMatchObject({ Operation: "set", Value: "gentle" });
    expect(await gui.sent()).toHaveLength(1);
  });

  test("Space within multiword typeahead does not commit a closed select", async ({ gui }) => {
    const playback = normal(gui).getByLabel("Playback");
    await playback.focus();
    await gui.page.keyboard.type("Speakers ");
    await expect(playback).toHaveValue("Speakers (SteelSeries Arena 7)");
    expect(await playback.evaluate((e) => e.matches(":open"))).toBe(false);
    expect(await gui.sent()).toEqual([]);
    await playback.press("Enter");
    expect(await gui.last("audio.normal-output")).toMatchObject({ Value: "Speakers (SteelSeries Arena 7)" });
    expect(await gui.sent()).toHaveLength(1);
  });

  for (const open of ["Alt+ArrowDown", "Space", "F4", "pointer"]) {
    for (const close of ["Enter", "Escape", "Tab"]) {
      test(`${open} native popup ${close} ${close === "Escape" ? "cancels" : "commits once"}`, async ({ gui }) => {
        const processing = normal(gui).getByLabel("Processing");
        await processing.focus();
        if (open === "pointer") await processing.click();
        else await processing.press(open);
        await expect.poll(() => processing.evaluate((e) => e.matches(":open"))).toBe(true);
        await gui.page.keyboard.press("ArrowDown");
        expect(await gui.sent()).toEqual([]);
        if (close === "Tab") await gui.page.keyboard.down(close);
        else await gui.page.keyboard.press(close);
        await expect.poll(() => processing.evaluate((e) => e.matches(":open"))).toBe(false);
        if (close !== "Escape") {
          expect(await gui.last("audio.normal-mode")).toMatchObject({ Value: "element" });
          // Native Tab confirms before release; no timeout may guess which key it was.
          if (close === "Tab") await gui.page.keyboard.up(close);
          expect(await gui.sent()).toHaveLength(1);
        } else {
          // Chrome emits change for the highlighted option even on native Escape.
          await expect(processing).toHaveValue("direct");
          expect(await gui.sent()).toEqual([]);
          await processing.focus();
          await processing.press("ArrowDown");
          expect(await gui.sent(), "popup cancellation must leave closed browsing safe").toEqual([]);
        }
      });
    }
  }

  test("native selection changes commit once without a keyboard draft", async ({ gui }) => {
    await normal(gui).getByLabel("Processing").selectOption("element");
    expect(await gui.last("audio.normal-mode")).toMatchObject({ Value: "element" });
    expect(await gui.sent()).toHaveLength(1);
  });

  test("clicking away from a native popup cancels its highlighted choice", async ({ gui }) => {
    const processing = normal(gui).getByLabel("Processing");
    await processing.focus();
    await processing.press("Space");
    await expect.poll(() => processing.evaluate((e) => e.matches(":open"))).toBe(true);
    await gui.page.keyboard.press("ArrowDown");
    await gui.page.getByRole("heading", { name: "Live Controls", exact: true }).click();
    await expect(processing).not.toBeFocused();
    await expect(processing).toHaveValue("direct");
    expect(await gui.sent()).toEqual([]);
  });

  test("Alt+Up dismisses a native popup without leaving a stuck draft", async ({ gui }) => {
    const processing = normal(gui).getByLabel("Processing");
    await processing.focus();
    await processing.press("Space");
    await gui.page.keyboard.press("ArrowDown");
    await gui.page.keyboard.press("Alt+ArrowUp");
    await expect.poll(() => processing.evaluate((e) => e.matches(":open"))).toBe(false);
    await expect(processing).toHaveValue("direct");
    expect(await gui.sent()).toEqual([]);
    await processing.focus();
    await processing.press("ArrowDown");
    await processing.press("Enter");
    expect(await gui.last("audio.normal-mode")).toMatchObject({ Value: "element" });
    expect(await gui.sent()).toHaveLength(1);
  });

  test("a focused semantic draft keeps its captured revision", async ({ gui }) => {
    const processing = normal(gui).getByLabel("Processing");
    const revision = await gui.page.evaluate(() => window.ctl("audio.normal-mode").Revision);
    await processing.focus();
    await processing.press("ArrowDown");
    await gui.update(() => {
      window.ctl("audio.normal-mode").Revision += 100;
      window.ctl("audio.normal-mode").Status = "Updated while editing";
    });
    await normal(gui).getByText("Updated while editing", { exact: true }).waitFor();
    await expect(processing).toHaveValue("element");
    expect(await gui.sent()).toEqual([]);
    await processing.press("Enter");
    expect(await gui.last("audio.normal-mode")).toMatchObject({ Value: "element", Revision: revision });
  });

  test("cancellation restores the latest observed semantic value", async ({ gui }) => {
    const monitor = normal(gui).getByLabel("Monitor");
    await monitor.focus();
    await monitor.press("End");
    await gui.update(() => {
      const control = window.ctl("audio.normal-monitor");
      control.Value = "off";
      control.Revision++;
      control.Status = "Updated while editing";
    });
    await normal(gui).getByText("Updated while editing", { exact: true }).waitFor();
    await expect(monitor).toHaveValue("post");
    await monitor.press("Escape");
    await expect(monitor).toHaveValue("off");
    expect(await gui.sent()).toEqual([]);
  });
});

test.describe("Routing dropdown commitment", () => {
  test.beforeEach(async ({ gui }) => {
    await gui.open();
    await gui.screen("Routing");
  });

  for (const scenario of [
    {
      label: "Webcam Source",
      key: "Home",
      value: "interface",
      id: "audio.priority-edit",
      edit: { list: "mics", op: "set", index: 2, value: "interface", field: "source" },
    },
    {
      label: "Desk on interface 1",
      key: "End",
      value: "stereo",
      id: "audio.priority-edit",
      edit: { list: "interfaces", op: "set", index: 0, value: "1,2", field: "input:desk" },
    },
    {
      label: "Meter",
      key: "End",
      value: "4",
      id: "audio.priority-edit",
      edit: { list: "activity", op: "set", index: 0, value: "4", field: "check" },
    },
    {
      label: "Music device",
      key: "Home",
      value: "",
      id: "audio.output-edit",
      edit: { op: "device", id: "music", value: "" },
    },
  ]) {
    test(`${scenario.label} cancels browsing and commits only on Enter`, async ({ gui }) => {
      const select = gui.page.getByLabel(scenario.label, { exact: true });
      const initial = await select.inputValue();
      await select.focus();
      await select.press(scenario.key);
      await expect(select).toHaveValue(scenario.value);
      expect(await gui.sent()).toEqual([]);
      await select.press("Escape");
      await expect(select).toHaveValue(initial);
      expect(await gui.sent()).toEqual([]);
      await select.focus();
      await select.press(scenario.key);
      await select.press("Enter");
      await gui.last(scenario.id);
      expect(await gui.sentValues(scenario.id)).toEqual([scenario.edit]);
    });
  }

  test("a dropdown drawn from an older Routing list keeps the stale sentinel", async ({ gui }) => {
    const meter = gui.page.getByLabel("Meter", { exact: true });
    await meter.focus();
    await meter.press("End");
    await gui.update(() => {
      const control = window.ctl("audio.priority-edit");
      control.Value += "-new";
      control.Revision += 100;
      control.Status = "Configuration changed";
    });
    await gui.page.locator("[data-part=priority-error]").getByText("Configuration changed").waitFor();
    await expect(meter).toHaveValue("4");
    expect(await gui.sent()).toEqual([]);
    await meter.press("Enter");
    expect(await gui.last("audio.priority-edit")).toMatchObject({ Revision: 0 });
    expect(await gui.sentValues("audio.priority-edit")).toEqual([
      { list: "activity", op: "set", index: 0, value: "4", field: "check" },
    ]);
  });
});
