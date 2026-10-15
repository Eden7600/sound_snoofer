import { expect, test } from "./host.ts";
import type { MediaView } from "../src/screens/media-types.ts";

for (const viewport of [
  { width: 1280, height: 820 },
  { width: 800, height: 600 },
]) {
  test(`Windows media errors use explicit error data at ${viewport.width}`, async ({ gui, page }) => {
    await page.setViewportSize(viewport);
    await gui.open();
    await gui.screen("Media");
    await gui.update(() => {
      const view = window.ctl<MediaView>("nowplaying.status").ViewData;
      if (!view) throw new Error("Media view missing");
      Object.assign(view, { Windows: "Connected", WindowsError: "Connected" });
      view.Bridge.Error = "";
      view.Bridge.Refused = "";
    });
    const problem = page
      .locator("[data-part=extension]")
      .getByText("Windows media sessions: Connected", { exact: true });
    await expect(problem).toBeVisible();
    await gui.update(() => {
      const view = window.ctl<MediaView>("nowplaying.status").ViewData;
      if (!view) throw new Error("Media view missing");
      delete view.WindowsError;
      view.Windows = "Connected";
    });
    await expect(problem).toHaveCount(0);
    expect(await gui.sent()).toEqual([]);
  });
}
