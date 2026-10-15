import { expect, test } from "./host.ts";
import type { PriorityView } from "../src/screens/routing-types.ts";

for (const viewport of [
  { width: 1280, height: 820 },
  { width: 800, height: 600 },
]) {
  test(`Routing sections and output management stay separate at ${viewport.width}px`, async ({ gui }, testInfo) => {
    await gui.page.setViewportSize(viewport);
    await gui.open();
    await gui.screen("Routing");
    const links = gui.page.getByRole("navigation", { name: "Routing sections" });
    for (const [label, id] of [
      ["Outputs", "outputs"],
      ["Microphones", "microphones"],
      ["Interfaces", "interfaces"],
      ["Playback", "playback"],
      ["Mic Priority", "mic-priority"],
      ["Activity", "activity"],
    ]) {
      await links.getByRole("link", { name: label, exact: true }).focus();
      await gui.page.keyboard.press("Enter");
      await expect(gui.page.locator(`#routing-${id}`)).toBeFocused();
    }
    const outputs = gui.page.locator("[data-part=outputs]");
    await expect(outputs.locator("thead input, thead select")).toHaveCount(0);
    const table = await outputs.locator("table").boundingBox();
    const editor = await outputs.locator("[data-part=output-editor]").boundingBox();
    expect(editor!.y).toBeGreaterThan(table!.y + table!.height);
    expect(await gui.sent()).toEqual([]);
    await gui.noHorizontalScroll();
    await gui.page.screenshot({ path: testInfo.outputPath(`Routing-${viewport.width}.png`), fullPage: true });
  });

  test(`Routing long candidates keep their actions beside the field at ${viewport.width}px`, async ({ gui }) => {
    await gui.page.setViewportSize(viewport);
    await gui.open((fixture) => {
      const view = fixture.Controls.find((c) => c.ID === "audio.priorities")!.ViewData as PriorityView;
      view.Lists!.playback![0]!.Matches = [
        "A very long playback device name with several words and a detailed connected device description",
      ];
    });
    await gui.screen("Routing");
    const playback = gui.page.locator("[data-part=list-playback]");
    const rows = playback.locator("[data-part=priority]");
    for (let n = 0; n < (await rows.count()); n++) {
      const row = rows.nth(n);
      const field = await row.locator("[data-part=device-entry], input").first().boundingBox();
      const remove = await row.getByRole("button", { name: `Remove Playback ${n + 1}`, exact: true }).boundingBox();
      expect(remove!.y + remove!.height / 2).toBeGreaterThanOrEqual(field!.y);
      expect(remove!.y + remove!.height / 2).toBeLessThanOrEqual(field!.y + field!.height);
      expect(remove!.x).toBeGreaterThan(field!.x);
    }
    await gui.noHorizontalScroll();
    expect(await gui.sent()).toEqual([]);
  });

  test(`Microphone rename exposes Apply only for a draft at ${viewport.width}px`, async ({ gui }) => {
    await gui.page.setViewportSize(viewport);
    await gui.open();
    await gui.screen("Routing");
    const row = gui.page.locator("[data-part=mic]").first();
    const name = row.getByLabel("Microphone 1 name");
    const apply = row.getByRole("button", { name: "Apply", exact: true });
    const original = await name.inputValue();
    await expect(apply).toBeHidden();
    await name.fill("Studio");
    await expect(apply).toBeVisible();
    await name.press("Escape");
    await expect(name).toHaveValue(original);
    await expect(apply).toBeHidden();
    await name.fill("Studio");
    await name.press("Tab");
    expect(await gui.sent()).toEqual([]);
    await name.press("Enter");
    expect(JSON.parse((await gui.last("audio.priority-edit")).Value)).toEqual({
      list: "mics",
      op: "set",
      index: 0,
      value: "Studio",
      field: "name",
    });
  });
}
