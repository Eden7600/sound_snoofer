import { expect, test } from "./host.ts";

for (const viewport of [
  { width: 1280, height: 820 },
  { width: 800, height: 600 },
]) {
  test(`Audio cards and live labels stay compact at ${viewport.width}px`, async ({ gui }, testInfo) => {
    await gui.page.setViewportSize(viewport);
    await gui.open();
    const geometry = await gui.page.evaluate(() => {
      const section = (title: string) =>
        [...document.querySelectorAll<HTMLElement>("section")].find(
          (s) => s.querySelector("h2")?.textContent === title,
        )!;
      const recording = section("Recording"),
        live = section("Live Controls");
      const last = recording.lastElementChild!.getBoundingClientRect();
      const gaps = ["Mic Stack", "Mic Mute", "Playback Mute"].map((label) => {
        const button = live.querySelector(`[aria-label="${label}"]`)!;
        const text = button.closest("[data-part=row]")!.querySelector("label")!;
        return button.getBoundingClientRect().left - text.getBoundingClientRect().right;
      });
      const iface = [...live.querySelectorAll("label")].find((label) => label.textContent === "Interface")!;
      return {
        bottomPadding: recording.getBoundingClientRect().bottom - last.bottom,
        gaps,
        interfaceBottom: iface.getBoundingClientRect().bottom,
        stripTop: live.querySelector("[data-part=strip]")!.getBoundingClientRect().top,
      };
    });
    expect(geometry.bottomPadding).toBeLessThanOrEqual(22);
    for (const gap of geometry.gaps) expect(gap).toBeLessThanOrEqual(20);
    expect(geometry.interfaceBottom).toBeLessThan(geometry.stripTop);
    await gui.page.screenshot({ path: testInfo.outputPath(`Audio-${viewport.width}.png`), fullPage: true });
    expect(await gui.sent()).toEqual([]);
  });

  test(`Audio and App Audio meter labels do not collide at ${viewport.width}px`, async ({ gui }) => {
    await gui.page.setViewportSize(viewport);
    await gui.open();
    for (const screen of ["Audio", "App Audio"]) {
      await gui.screen(screen);
      const collisions = await gui.page.locator("[data-part=meter] + div").evaluateAll((scales) =>
        scales.flatMap((scale) => {
          const marks = [...scale.querySelectorAll("span")]
            .filter((mark) => mark.checkVisibility())
            .map((mark) => ({ label: mark.textContent, rect: mark.getBoundingClientRect() }));
          return marks
            .slice(1)
            .flatMap((mark, i) =>
              marks[i]!.rect.right > mark.rect.left - 6 ? [`${marks[i]!.label}/${mark.label}`] : [],
            );
        }),
      );
      expect(collisions, screen).toEqual([]);
      if (screen === "Audio" && viewport.width === 1280)
        await expect(gui.page.locator("[data-part=strip]").first().getByText("-3", { exact: true })).toBeVisible();
    }
  });
}
