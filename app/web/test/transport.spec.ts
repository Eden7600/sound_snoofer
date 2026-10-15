import { expect, test, type Gui } from "./host.ts";
import { dot } from "./fixtures/manual.ts";
import type { MediaView } from "../src/screens/media-types.ts";
import type { PriorityView } from "../src/screens/routing-types.ts";
import type { ArtworkFrame } from "../src/types.ts";

declare global {
  interface Window {
    releaseAnimation?: () => void;
  }
}

test.beforeEach(async ({ page }) => {
  await page.clock.install();
  await page.clock.pauseAt(new Date(Date.now() + 1000));
});

async function freezeTelemetry(gui: Gui) {
  await gui.update(() => {
    window.liveMeters = false;
    for (const control of window.fixture.Controls) if (control.Meter) control.Meter.At = new Date().toISOString();
  });
  await gui.page.clock.runFor(200);
}

test("artwork is cached across idle polls, telemetry and navigation, then replaced and cleared", async ({ gui }) => {
  await gui.open();
  await freezeTelemetry(gui);
  await gui.update(() => {
    window.animations = {};
  });
  await gui.screen("Soundboard");
  await gui.page.getByRole("searchbox", { name: "Search Clips" }).fill("sad");
  await gui.page.clock.runFor(200);
  const clip = gui.page.locator("[data-part=clip]:visible");
  const original = await gui.page.evaluate(() => window.ctl("soundboard.clip-sadge").Artwork);
  await expect(clip.locator("img")).toHaveAttribute("src", "data:image/png;base64," + original);
  const before = await gui.page.evaluate(() => ({ ...window.transport }));
  await gui.page.clock.runFor(600);
  expect(await gui.page.evaluate(() => window.transport.unchanged)).toBeGreaterThan(before.unchanged);
  await gui.update(() => {
    window.ctl("soundboard.clip-sadge").Value = "Playing";
  });
  await gui.page.clock.runFor(200);
  await expect(clip).toHaveAttribute("data-state", "playing");
  expect(await gui.page.evaluate(() => window.transport.artwork)).toBe(before.artwork);
  await gui.screen("Audio");
  await gui.screen("Soundboard");
  await gui.page.getByRole("searchbox", { name: "Search Clips" }).fill("sad");
  await expect(clip.locator("img")).toHaveAttribute("src", "data:image/png;base64," + original);
  expect(await gui.page.evaluate(() => window.transport.artwork)).toBe(before.artwork);
  await gui.update((replacement) => {
    window.ctl("soundboard.clip-sadge").Artwork = replacement;
  }, dot);
  await gui.page.clock.runFor(200);
  await expect(clip.locator("img")).toHaveAttribute("src", "data:image/png;base64," + dot);
  await gui.update(() => {
    for (const control of window.fixture.Controls) control.Artwork = "";
  });
  await gui.page.clock.runFor(200);
  await expect(clip.locator("img")).toHaveCount(0);
  expect(await gui.sent()).toEqual([]);
});

test("hydration leaves received controls and fixture artwork immutable", async ({ gui }) => {
  await gui.open();
  await freezeTelemetry(gui);
  await gui.update(() => {
    const desktop = window.go?.app?.Desktop;
    if (!desktop) throw new Error("Desktop missing");
    const state = desktop.State.bind(desktop);
    desktop.State = async (...args) => {
      const next = await state(...args);
      if (next) for (const control of next.State.Controls || []) Object.freeze(control);
      return next;
    };
    window.ctl("soundboard.clip-sadge").Value = "Playing";
  });
  await gui.page.clock.runFor(200);
  await gui.screen("Soundboard");
  await gui.page.getByRole("searchbox", { name: "Search Clips" }).fill("sad");
  await expect(gui.page.locator("[data-part=clip]:visible img")).toHaveCount(1);
  await expect(gui.page.locator("#notice")).toBeHidden();
  const shapes = await gui.page.evaluate(async () => {
    const wire = await window.go?.app?.Desktop?.State(0);
    return {
      inline: wire?.State.Controls?.find((c) => c.ID === "soundboard.clip-sadge")?.Artwork,
      fixture: window.ctl("soundboard.clip-sadge").Artwork,
    };
  });
  expect(shapes.inline).toBe("");
  expect(shapes.fixture).toBeTruthy();
});

test("null polls still expire meters and pending requests", async ({ gui }) => {
  await gui.open();
  await freezeTelemetry(gui);
  await gui.page.clock.runFor(600);
  await expect(gui.page.locator("[data-part=strip]").first().getByText("LEVEL N/A")).toBeVisible();
  await gui.update(() => {
    const desktop = window.go?.app?.Desktop;
    if (!desktop) throw new Error("Desktop missing");
    desktop.Send = async (action) => {
      window.sent.push(action);
    };
  });
  await gui.page.locator("[data-part=tape]").getByRole("button", { name: "Record", exact: true }).click();
  await expect(gui.page.locator("#content")).toHaveAttribute("aria-busy", "true");
  const before = await gui.page.evaluate(() => window.transport.unchanged);
  await gui.page.clock.runFor(2700);
  await expect(gui.page.locator("#content")).toHaveAttribute("aria-busy", "false");
  expect(await gui.page.evaluate(() => window.transport.unchanged)).toBeGreaterThan(before);
  expect(await gui.sent()).toHaveLength(1);
});

test("null polls advance the local media clock", async ({ gui }) => {
  await gui.open();
  await freezeTelemetry(gui);
  await gui.update(() => {
    const session = window.ctl<MediaView>("nowplaying.status").ViewData?.Sessions?.[0];
    if (!session) throw new Error("session missing");
    Object.assign(session, { Status: "Playing", PositionMs: 10000, AtMs: Date.now(), Rate: 1 });
  });
  await gui.page.clock.runFor(200);
  await gui.screen("Media");
  const seek = gui.page.locator("[data-part=media-session]").first().locator("[data-part=seek]");
  const initial = Number(await seek.inputValue());
  const before = await gui.page.evaluate(() => window.transport.unchanged);
  await gui.page.clock.runFor(2000);
  expect(Number(await seek.inputValue())).toBeGreaterThanOrEqual(initial + 1000);
  expect(await gui.page.evaluate(() => window.transport.unchanged)).toBeGreaterThan(before);
});

test("null polls finish a deferred structural rebuild after blur", async ({ gui }) => {
  await gui.open();
  await freezeTelemetry(gui);
  await gui.screen("Routing");
  const pattern = gui.page.locator("[data-part=list-playback]").getByLabel("Playback 2 pattern");
  const initial = await pattern.inputValue();
  await pattern.focus();
  await gui.update(() => {
    const row = window.ctl<PriorityView>("audio.priorities").ViewData?.Lists?.playback?.[1];
    if (!row) throw new Error("priority row missing");
    row.Pattern = "(?i)changed-source";
    window.ctl("audio.priority-edit").Value = "updated configuration";
  });
  await gui.page.clock.runFor(250);
  await expect(pattern).toHaveValue(initial);
  const before = await gui.page.evaluate(() => window.transport.unchanged);
  await pattern.evaluate((node) => node.blur());
  await gui.page.clock.runFor(250);
  await expect(pattern).toHaveValue("(?i)changed-source");
  expect(await gui.page.evaluate(() => window.transport.unchanged)).toBeGreaterThan(before);
  expect(await gui.sent()).toEqual([]);
});

test("animation generations replace later frames and reject a delayed old lookup", async ({ gui }) => {
  await gui.open();
  await freezeTelemetry(gui);
  await gui.update(() => {
    const desktop = window.go?.app?.Desktop;
    if (!desktop) throw new Error("Desktop missing");
    const animation = desktop.Animation.bind(desktop);
    let held = false;
    desktop.Animation = async (id) => {
      const frames = await animation(id);
      if (id !== "soundboard.clip-sadge" || held) return frames;
      held = true;
      return new Promise<ArtworkFrame[] | null>((resolve) => {
        window.releaseAnimation = () => resolve(frames);
      });
    };
  });
  await gui.screen("Soundboard");
  await gui.page.getByRole("searchbox", { name: "Search Clips" }).fill("sad");
  const image = gui.page.locator("[data-part=clip]:visible img");
  const still = await gui.page.evaluate(() => window.ctl("soundboard.clip-sadge").Artwork);
  const before = await gui.page.evaluate(() => ({ ...window.transport }));
  await gui.update(() => {
    const artwork = window.ctl("soundboard.clip-sadge").Artwork;
    if (!artwork || !window.releaseAnimation) throw new Error("initial animation not held");
    window.animations["soundboard.clip-sadge"] = [
      { Artwork: artwork, Delay: 100e6 },
      { Artwork: artwork, Delay: 200e6 },
    ];
  });
  await gui.page.clock.runFor(500);
  await expect(image).toHaveAttribute("src", "data:image/png;base64," + still);
  expect(await gui.page.evaluate(() => window.transport.animations)).toBeGreaterThan(before.animations);
  expect(await gui.page.evaluate(() => window.transport.artwork)).toBe(before.artwork);
  await gui.update(() => {
    window.releaseAnimation?.();
  });
  await gui.page.clock.runFor(800);
  await expect(image).toHaveAttribute("src", "data:image/png;base64," + still);
  await gui.update(() => {
    delete window.animations["soundboard.clip-sadge"];
  });
  await gui.page.clock.runFor(500);
  const afterRemoval = await gui.page.evaluate(() => window.transport.animations);
  await gui.page.clock.runFor(500);
  await expect(image).toHaveAttribute("src", "data:image/png;base64," + still);
  expect(await gui.page.evaluate(() => window.transport.animations)).toBe(afterRemoval);
  expect(await gui.page.evaluate(() => window.transport.artwork)).toBe(before.artwork);
});
