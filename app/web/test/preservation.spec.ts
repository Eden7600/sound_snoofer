import { expect, test, type Gui } from "./host.ts";

// Changing an unrelated provider label forces a real automatic rebuild on every screen.
async function rebuild(gui: Gui) {
  const old = await gui.page.locator("#content > :first-child").elementHandle();
  if (!old) throw new Error("content missing");
  await gui.update(() => {
    window.ctl("aec.strength").Label += " updated";
  });
  await expect.poll(() => old.evaluate((node) => node.isConnected)).toBe(false);
}

test("Soundboard search and filtering survive an automatic rebuild", async ({ gui }) => {
  await gui.open();
  await gui.screen("Soundboard");
  const search = gui.page.getByRole("searchbox", { name: "Search Clips" });
  await search.fill("sad");
  await search.blur();
  await rebuild(gui);
  await expect(search).toHaveValue("sad");
  await expect(gui.page.locator("[data-part=clip]:visible")).toHaveCount(1);
  expect(await gui.sent()).toEqual([]);
  await gui.screen("Audio");
  await gui.screen("Soundboard");
  await expect(search).toHaveValue("");
});

test("App Audio More stays expanded and its same action keeps focus", async ({ gui }) => {
  await gui.open();
  await gui.screen("App Audio");
  const discord = gui.page.locator("[data-part=app]", {
    has: gui.page.getByRole("heading", { name: "Discord", exact: true }),
  });
  await discord.locator("summary").click();
  const mute = discord.getByRole("button", { name: "Mute", exact: true });
  await mute.focus();
  await rebuild(gui);
  await expect(discord.locator("details")).toHaveAttribute("open", "");
  await expect(mute).toBeFocused();
  expect(await gui.sent()).toEqual([]);
});

test("Lights Other Rooms stays expanded through catalogue updates", async ({ gui }) => {
  await gui.open();
  await gui.screen("Lights");
  const details = gui.page.locator("details", { has: gui.page.getByText("Other Rooms", { exact: true }) });
  await details.locator("summary").click();
  await rebuild(gui);
  await expect(details).toHaveAttribute("open", "");
  await expect(details.locator("summary")).toBeFocused();
  expect(await gui.sent()).toEqual([]);
});

for (const viewport of [
  { width: 1280, height: 820 },
  { width: 800, height: 600 },
]) {
  test(`Deck search, details and scroll survive rebuild at ${viewport.width}x${viewport.height}`, async ({ gui }) => {
    await gui.page.setViewportSize(viewport);
    await gui.open();
    await gui.screen("Stream Deck");
    await gui.update(() => {
      window.liveMeters = false;
    });
    const pageOptions = gui.page.locator("details", { has: gui.page.getByText("Page Options", { exact: true }) });
    await pageOptions.locator("summary").click();
    const search = gui.page.getByRole("searchbox", { name: "Find a Binding" });
    await search.fill("audio");
    await search.blur();
    const binding = gui.page.getByLabel("Binding", { exact: true });
    const options = await binding.locator("option").allTextContents();
    await binding.evaluate((node) => {
      node.scrollTop = 100;
    });
    await gui.page.evaluate(() => window.scrollTo(0, 250));
    const before = {
      document: await gui.page.evaluate(() => scrollY),
      binding: await binding.evaluate((node) => node.scrollTop),
    };
    expect(before.document).toBeGreaterThan(0);
    expect(before.binding).toBeGreaterThan(0);
    await rebuild(gui);
    await expect(search).toHaveValue("audio");
    await expect(pageOptions).toHaveAttribute("open", "");
    expect(await binding.locator("option").allTextContents()).toEqual(options);
    expect(await gui.page.evaluate(() => scrollY)).toBe(before.document);
    await expect.poll(() => binding.evaluate((node) => node.scrollTop)).toBe(before.binding);
    const unchanged = await gui.page.evaluate(() => window.transport.unchanged);
    await expect.poll(() => gui.page.evaluate(() => window.transport.unchanged)).toBeGreaterThan(unchanged + 1);
    expect(await binding.evaluate((node) => node.scrollTop)).toBe(before.binding);
    expect(await gui.sent()).toEqual([]);
  });
}

test("removed focus targets fall back to content without moving to a replacement", async ({ gui }) => {
  await gui.open();
  await gui.screen("Soundboard");
  const clip = gui.page.locator("[data-part=clip]", { has: gui.page.getByText("sadge", { exact: true }) });
  await clip.focus();
  await gui.update(() => {
    const old = window.ctl("soundboard.clip-sadge");
    window.fixture.Controls = window.fixture.Controls.filter((control) => control.ID !== old.ID);
    window.fixture.Controls.push({ ...old, ID: "soundboard.clip-replacement", Label: "Replacement" });
  });
  await expect(clip).toHaveCount(0);
  await expect(gui.page.locator("#content")).toBeFocused();
  expect(await gui.sent()).toEqual([]);
});

test("a surviving semantic action retains focus without dispatch", async ({ gui }) => {
  await gui.open();
  const mute = gui.page.getByRole("button", { name: "Mic Mute", exact: true });
  await mute.focus();
  await rebuild(gui);
  await expect(mute).toBeFocused();
  expect(await gui.sent()).toEqual([]);
});

test("a host dialog keeps focus when content rebuilds underneath it", async ({ gui }) => {
  await gui.open();
  await gui.update(() => {
    window.fixture.Confirmation = "Enable media and restart Snoofer?";
  });
  const cancel = gui.page.getByRole("dialog").getByRole("button", { name: "Cancel", exact: true });
  await expect(cancel).toBeFocused();
  await rebuild(gui);
  await expect(cancel).toBeFocused();
  expect(await gui.sent()).toEqual([]);
});

test("an unavailable surviving action falls back to content", async ({ gui }) => {
  await gui.open();
  const mute = gui.page.getByRole("button", { name: "Mic Mute", exact: true });
  await mute.focus();
  await gui.update(() => {
    window.ctl("audio.mic-mute").Available = false;
    window.ctl("aec.strength").Label += " updated";
  });
  await expect(mute).toBeDisabled();
  await expect(gui.page.locator("#content")).toBeFocused();
  expect(await gui.sent()).toEqual([]);
});

test("Media focus follows the session ID when a session is inserted before it", async ({ gui }) => {
  await gui.open();
  await gui.screen("Media");
  const song = gui.page.locator("[data-part=media-session]", {
    has: gui.page.getByRole("heading", { name: "Song", exact: true }),
  });
  const play = song.getByRole("button", { name: "Play Song", exact: true });
  await play.focus();
  await gui.update(() => {
    const old = window.ctl("nowplaying.s-spotify");
    window.fixture.Controls.push({ ...old, ID: "nowplaying.s-new", Label: "New session", Order: -1 });
  });
  await expect(gui.page.locator("[data-part=media-session]")).toHaveCount(3);
  await expect(play).toBeFocused();
  expect(await gui.sent()).toEqual([]);
});

test("every screen has unique preservation keys through an actual rebuild", async ({ gui }) => {
  await gui.open();
  for (const screen of [
    "Audio",
    "Routing",
    "Soundboard",
    "Lights",
    "App Audio",
    "Meetings",
    "Media",
    "Stream Deck",
    "Plugins",
    "Third-Party Apps",
    "Diagnostics",
  ]) {
    await gui.screen(screen);
    await rebuild(gui);
    const keys = await gui.page
      .locator("#content [data-view-key]")
      .evaluateAll((nodes) => nodes.map((node) => (node as HTMLElement).dataset.viewKey));
    expect(new Set(keys).size, screen).toBe(keys.length);
    await expect(gui.page.locator("#notice")).not.toContainText("Duplicate view key");
  }
  expect(await gui.sent()).toEqual([]);
});
