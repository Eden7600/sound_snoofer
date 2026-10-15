import { expect, test } from "./host.ts";
import type { AppAudioView } from "../src/screens/appaudio-types.ts";
import type { MediaView } from "../src/screens/media-types.ts";

test.beforeEach(async ({ gui, page }) => {
  await page.clock.install();
  await page.clock.pauseAt(new Date(Date.now() + 1000));
  await gui.open();
  await gui.update(() => {
    window.liveMeters = false;
  });
  await page.clock.runFor(200);
});

test("App Audio polling preserves selected Programs while Heard and details update", async ({ gui, page }) => {
  await gui.update(() => {
    const app = window.ctl<AppAudioView>("appaudio.status").ViewData?.Apps?.[0];
    if (!app) throw new Error("missing Discord fixture app");
    app.LastHeard = new Date(Date.now() - 10000).toISOString();
  });
  await page.clock.runFor(200);
  await gui.screen("App Audio");
  const card = page.locator("[data-part=app]").first();
  await card.locator("summary").click();
  const values = card.locator("dl dd"),
    programs = values.nth(0),
    heard = values.nth(5);
  await expect(heard).toHaveText("10s ago");
  const selectedText = await programs.evaluateHandle((node) => {
    const range = document.createRange();
    range.selectNodeContents(node);
    const selection = window.getSelection();
    if (!selection) throw new Error("selection unavailable");
    selection.removeAllRanges();
    selection.addRange(range);
    return node.firstChild;
  });
  const selected = await programs.textContent();
  const unchanged = await page.evaluate(() => window.transport.unchanged);
  await page.clock.runFor(2000);
  await expect(heard).toHaveText("12s ago");
  expect(await page.evaluate(() => window.transport.unchanged)).toBeGreaterThan(unchanged);
  expect(await page.evaluate(() => window.getSelection()?.toString())).toBe(selected);
  expect(await programs.evaluate((node, text) => node.firstChild === text, selectedText)).toBe(true);

  await gui.update(() => {
    const app = window.ctl<AppAudioView>("appaudio.status").ViewData?.Apps?.[0];
    if (!app) throw new Error("missing Discord fixture app");
    Object.assign(app, { Devices: ["USB Speakers", "HDMI Audio"], PIDs: [31, 32], Sessions: 4 });
  });
  await page.clock.runFor(200);
  expect(await values.nth(1).textContent()).toBe("USB Speakers\nHDMI Audio");
  await expect(values.nth(2)).toHaveText("31, 32");
  await expect(values.nth(3)).toHaveText("4");
  expect(await page.evaluate(() => window.getSelection()?.toString())).toBe(selected);
  expect(await programs.evaluate((node, text) => node.firstChild === text, selectedText)).toBe(true);

  await gui.update(() => {
    const app = window.ctl<AppAudioView>("appaudio.status").ViewData?.Apps?.[0];
    if (!app) throw new Error("missing Discord fixture app");
    app.Executables = ["C:\\Discord\\new.exe"];
  });
  await page.clock.runFor(200);
  await expect(programs).toHaveText("C:\\Discord\\new.exe");
  await gui.update(() => {
    const app = window.ctl<AppAudioView>("appaudio.status").ViewData?.Apps?.[0];
    if (!app) throw new Error("missing Discord fixture app");
    Object.assign(app, { Executables: null, Devices: null, PIDs: null, Sessions: 0, LastHeard: undefined });
  });
  await page.clock.runFor(200);
  expect(await values.allTextContents()).toEqual(["—", "—", "—", "0", "None", "Not yet"]);
  expect(await gui.sent()).toEqual([]);
});

test("Media polling retains its icon across metadata updates and changes it with playback", async ({ gui, page }) => {
  await gui.screen("Media");
  const card = page.locator("[data-part=media-session]").first(),
    play = card.locator("[data-part=play]"),
    seek = card.locator("[data-part=seek]");
  await expect(play).toHaveAccessibleName("Pause Video A");
  const playingIcon = await play.locator("svg").elementHandle();
  if (!playingIcon) throw new Error("missing playing icon");
  const pauseShape = await playingIcon.innerHTML();
  const unchanged = await page.evaluate(() => window.transport.unchanged);
  const position = Number(await seek.inputValue());
  await page.clock.runFor(1200);
  expect(await page.evaluate(() => window.transport.unchanged)).toBeGreaterThan(unchanged);
  expect(Number(await seek.inputValue())).toBeGreaterThanOrEqual(position + 1000);
  expect(await playingIcon.evaluate((node) => node.isConnected)).toBe(true);

  await gui.update(() => {
    const session = window.ctl<MediaView>("nowplaying.status").ViewData?.Sessions?.[0];
    if (!session) throw new Error("missing browser fixture session");
    Object.assign(session, { Artist: "New Artist", Album: "New Album", PositionMs: 45000, AtMs: Date.now() });
  });
  await page.clock.runFor(200);
  await expect(card.locator("h3")).toHaveText("Video A");
  await expect(card).toContainText("New Artist · New Album");
  await expect(play).toHaveAccessibleName("Pause Video A");
  expect(await playingIcon.evaluate((node) => node.isConnected)).toBe(true);
  expect(Number(await seek.inputValue())).toBeGreaterThanOrEqual(45000);

  await gui.update(() => {
    const session = window.ctl<MediaView>("nowplaying.status").ViewData?.Sessions?.[0];
    if (!session) throw new Error("missing browser fixture session");
    session.Status = "Paused";
  });
  await page.clock.runFor(200);
  await expect(play).toHaveAccessibleName("Play Video A");
  expect(await playingIcon.evaluate((node) => node.isConnected)).toBe(false);
  const pausedIcon = await play.locator("svg").elementHandle();
  if (!pausedIcon) throw new Error("missing paused icon");
  expect(await pausedIcon.innerHTML()).not.toBe(pauseShape);
  await page.clock.runFor(500);
  expect(await pausedIcon.evaluate((node) => node.isConnected)).toBe(true);
  await play.click();
  await gui.settled();
  await page.clock.runFor(300);
  const sent = await gui.sent();
  expect(sent).toHaveLength(1);
  expect(sent[0]?.Request).toMatchObject({ ID: "nowplaying.s-tab", Operation: "press" });

  await gui.update(() => {
    const session = window.ctl<MediaView>("nowplaying.status").ViewData?.Sessions?.[0];
    if (!session) throw new Error("missing browser fixture session");
    session.Status = "Playing";
  });
  await page.clock.runFor(200);
  await expect(play).toHaveAccessibleName("Pause Video A");
  expect(await pausedIcon.evaluate((node) => node.isConnected)).toBe(false);
  expect(await play.locator("svg").innerHTML()).toBe(pauseShape);

  await gui.update(() => {
    const session = window.ctl<MediaView>("nowplaying.status").ViewData?.Sessions?.[0];
    if (!session) throw new Error("missing browser fixture session");
    session.Title = "Video B";
    Object.assign(window.ctl(session.ID), { Label: session.Title, ShortLabel: session.Title });
  });
  await page.clock.runFor(200);
  await expect(card.locator("h3")).toHaveText("Video B");
  await expect(play).toHaveAccessibleName("Pause Video B");
  expect(await play.locator("svg").innerHTML()).toBe(pauseShape);
});
