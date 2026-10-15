import { expect, test } from "./host.ts";
import type { OutputSlotView } from "../src/screens/routing-types.ts";

for (const size of [
  { width: 1280, height: 820 },
  { width: 800, height: 600 },
]) {
  test(`output device detail and editing retain slot identity at ${size.width}px`, async ({ gui }) => {
    await gui.page.setViewportSize(size);
    await gui.open();
    await gui.screen("Routing");
    const device = await gui.page.evaluate(() => {
      const slot = window.ctl<OutputSlotView>("audio.slot-1");
      return { status: slot.Status, detail: slot.ViewData };
    });
    expect(device.status).toBe("");
    expect(device.detail).toMatchObject({ ID: "music", Device: "Speakers (SteelSeries Arena 7)" });
    const slot = gui.page.locator("[data-part=output-slot]");
    await expect(slot).toContainText("In use");
    await expect(slot.getByLabel("Music device")).toHaveValue("current");
    await expect(
      slot.getByLabel("Music device").getByRole("option", { name: "Speakers (SteelSeries Arena 7)", exact: true }),
    ).toHaveCount(1);
    await slot.getByLabel("Music device").selectOption({ label: "Speakers (Realtek Audio)" });
    const request = await gui.last("audio.output-edit");
    expect(JSON.parse(request.Value)).toEqual({
      op: "device",
      id: "music",
      value: JSON.stringify({ ID: "{realtek}", Name: "Speakers (Realtek Audio)" }),
    });
    await gui.noHorizontalScroll();
  });
}
