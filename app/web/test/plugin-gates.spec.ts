import { expect, test } from "./host.ts";

const screens = [
  ["Audio", "audio", "Audio", "audio.health"],
  ["Routing", "audio", "Audio", "audio.priorities"],
  ["Soundboard", "soundboard", "Soundboard", "soundboard.status"],
  ["Lights", "hue", "Hue", "hue.status"],
  ["App Audio", "appaudio", "App Audio", "appaudio.status"],
  ["Meetings", "discord", "Discord", "discord.channel"],
  ["Media", "nowplaying", "Now Playing", "nowplaying.status"],
  ["Stream Deck", "streamdeck", "Stream Deck", "streamdeck.preview"],
] as const;

for (const [screen, id, label, publication] of screens) {
  test(`${screen} distinguishes lifecycle states and first publication`, async ({ gui, page }) => {
    await gui.open((fixture) => {
      fixture.PluginStates[id] = "disabled";
      fixture.Enabled[id] = false;
    });
    await gui.screen(screen);
    const gate = page.locator("[data-part=plugin-gate]");
    await expect(gate.getByRole("heading", { name: label, exact: true })).toBeVisible();
    await expect(gate.getByText("Off", { exact: true })).toBeVisible();
    await gate.getByRole("button", { name: "Enable " + label, exact: true }).click();
    expect((await gui.sent()).at(-1)).toMatchObject({ Kind: "selection", Plugin: id, Enable: true });
    await gui.update((id) => {
      window.fixture.PluginStates[id] = "missing";
      window.fixture.Plugins[id] = "Running";
    }, id);
    await expect(gate.getByText("Not part of this build", { exact: true })).toBeVisible();
    await expect(gate.getByRole("button", { name: /^Enable/ })).toHaveCount(0);
    await gui.update((id) => {
      window.fixture.PluginStates[id] = "starting";
    }, id);
    await expect(gate.getByText("Starting", { exact: true })).toBeVisible();
    await expect(gate.getByRole("button", { name: "Plugins", exact: true })).toBeVisible();
    await gui.update((id) => {
      window.fixture.PluginStates[id] = "failed";
    }, id);
    await expect(gate.getByText("Could not start", { exact: true })).toBeVisible();
    await expect(gate.getByText("Running", { exact: true })).toBeVisible();
    await gui.update((id) => {
      window.fixture.PluginStates[id] = "running";
      window.fixture.Enabled[id] = true;
    }, id);
    await expect(gate).toHaveCount(0);
    await gui.update((publication) => {
      window.fixture.Controls = window.fixture.Controls.filter((control) => control.ID !== publication);
    }, publication);
    await expect(gate.getByText("Starting", { exact: true })).toBeVisible();
    expect(await gui.sent()).toHaveLength(1);
  });
}

test("failed plugin recovery opens Plugins and retries only on request", async ({ gui, page }) => {
  await gui.open((fixture) => {
    fixture.PluginStates.nowplaying = "failed";
    fixture.Plugins.nowplaying = "Disabled";
    fixture.PluginStates.media = "missing";
  });
  await gui.screen("Media");
  await page.locator("[data-part=plugin-gate]").getByRole("button", { name: "Plugins", exact: true }).click();
  await expect(page.locator("#title")).toHaveText("Plugins");
  expect(await gui.sent()).toEqual([]);
  const retry = page.getByRole("button", { name: "Retry Plugins", exact: true });
  await expect(retry).toBeEnabled();
  const missing = page.locator("[data-part=plugin]", {
    has: page.getByRole("heading", { name: "Windows Media", exact: true }),
  });
  await expect(missing.getByRole("button")).toHaveCount(0);
  await gui.update(() => {
    const desktop = window.go?.app?.Desktop;
    if (!desktop) throw new Error("Desktop missing");
    desktop.Send = (action) => {
      window.sent.push(action);
      return new Promise(() => {});
    };
  });
  await retry.click();
  expect((await gui.sent()).at(-1)).toMatchObject({ Kind: "retry" });
  await expect(retry).toBeDisabled();
  await gui.update(() => {
    window.fixture.PluginStates.nowplaying = "running";
    window.fixture.Plugins.nowplaying = "Running";
  });
  await expect(retry).toBeDisabled();
});

test("a focused editor cannot defer its plugin disappearing", async ({ gui, page }) => {
  await gui.open();
  await gui.screen("Routing");
  const name = page.locator("[data-part=outputs] input[type=text]").first();
  await name.fill("Unsent output name");
  await gui.update(() => {
    window.fixture.PluginStates.audio = "failed";
    window.fixture.Plugins.audio = "Device failure";
  });
  await expect(page.locator("[data-part=plugin-gate]")).toContainText("Device failure");
  await expect(name).toHaveCount(0);
  await expect(page.locator("#content")).toBeFocused();
  expect(await gui.sent()).toEqual([]);
});

test("Enable keeps host confirmation and cancellation separate from observed lifecycle", async ({ gui, page }) => {
  await gui.open((fixture) => {
    fixture.PluginStates.hue = "disabled";
    fixture.Enabled.hue = false;
  });
  await gui.screen("Lights");
  const gate = page.locator("[data-part=plugin-gate]");
  const enable = gate.getByRole("button", { name: "Enable Hue", exact: true });
  for (const choice of ["Cancel", "Apply & restart"]) {
    await enable.click();
    await gui.update(() => {
      window.fixture.Confirmation = "Enable Hue and restart Snoofer? Recording will stop.";
    });
    const dialog = page.getByRole("dialog");
    await expect(dialog).toContainText("Recording will stop.");
    await dialog.getByRole("button", { name: choice, exact: true }).click();
    await gui.update(() => {
      window.fixture.Confirmation = "";
      window.fixture.Notice += " resolved";
    });
    await expect(dialog).toBeHidden();
    await expect(gate.getByText("Off", { exact: true })).toBeVisible();
  }
  expect((await gui.sent()).map(({ Kind, Plugin, Enable }) => ({ Kind, Plugin, Enable }))).toEqual([
    { Kind: "selection", Plugin: "hue", Enable: true },
    { Kind: "cancel", Plugin: undefined, Enable: undefined },
    { Kind: "selection", Plugin: "hue", Enable: true },
    { Kind: "confirm", Plugin: undefined, Enable: undefined },
  ]);
  await gui.update(() => {
    window.fixture.Enabled.hue = true;
    window.fixture.PluginStates.hue = "running";
  });
  await expect(gate).toHaveCount(0);
});

test("disconnection disables lifecycle gates and retry without replay", async ({ gui, page }) => {
  await gui.open((fixture) => {
    fixture.PluginStates.hue = "disabled";
    fixture.PluginStates.aec = "failed";
  });
  await gui.screen("Lights");
  await gui.update(() => {
    const desktop = window.go?.app?.Desktop;
    if (!desktop) throw new Error("Desktop missing");
    desktop.State = async () => {
      throw new Error("Disconnected");
    };
  });
  await expect(page.locator("#connection")).toBeVisible();
  await expect(page.getByRole("button", { name: "Enable Hue", exact: true })).toBeDisabled();
  await gui.screen("Plugins");
  await expect(page.getByRole("button", { name: "Retry Plugins", exact: true })).toBeDisabled();
  expect(await gui.sent()).toEqual([]);
});

test("empty ready screens stay usable and an optional AEC failure does not gate Audio", async ({ gui, page }) => {
  await gui.open((fixture) => {
    fixture.PluginStates.aec = "failed";
    fixture.Plugins.aec = "Running";
    fixture.Controls = fixture.Controls.filter(
      (control) => !["soundboard.clips", "nowplaying.sessions", "appaudio.apps"].includes(control.Collection || ""),
    );
    for (const control of fixture.Controls) {
      if (control.ID === "nowplaying.status") Object.assign(control.ViewData as object, { Sessions: [], Browsers: [] });
      if (control.ID === "appaudio.status") Object.assign(control.ViewData as object, { Apps: [] });
    }
  });
  for (const name of ["Audio", "Soundboard", "Media", "App Audio", "Stream Deck"]) {
    await gui.screen(name);
    await expect(page.locator("[data-part=plugin-gate]")).toHaveCount(0);
  }
  await gui.screen("Third-Party Apps");
  await expect(page.locator("[data-part=not-monitored]")).toContainText(
    "Echo Cancellation — Could not start — Running",
  );
  expect(await gui.sent()).toEqual([]);
});

for (const [screen, publication] of [
  ["Routing", "audio.priorities"],
  ["App Audio", "appaudio.status"],
  ["Media", "nowplaying.status"],
  ["Stream Deck", "streamdeck.preview"],
]) {
  test(`${screen} waits for its structured view`, async ({ gui, page }) => {
    await gui.open((fixture) => {
      const control = fixture.Controls.find((control) => control.ID === publication);
      if (!control) throw new Error("Publication missing");
      delete control.ViewData;
    });
    await gui.screen(screen!);
    await expect(page.locator("[data-part=plugin-gate]").getByText("Starting", { exact: true })).toBeVisible();
    expect(await gui.sent()).toEqual([]);
  });
}

test("a local confirmation stays invalid after plugin failure and recovery", async ({ gui, page }) => {
  await gui.open((fixture) => {
    const deletion = fixture.Controls.find((control) => control.ID === "streamdeck.delete");
    if (!deletion) throw new Error("Missing page deletion");
    deletion.Available = true;
  });
  await gui.screen("Stream Deck");
  await page.getByText("Page Options", { exact: true }).click();
  await page.getByRole("button", { name: "Delete Page", exact: true }).click();
  const dialog = page.getByRole("dialog");
  const accept = dialog.getByRole("button", { name: "Delete Page", exact: true });
  await expect(accept).toBeEnabled();
  await gui.update(() => {
    window.fixture.PluginStates.streamdeck = "failed";
    window.fixture.Plugins.streamdeck = "Deck failed";
  });
  await expect(accept).toBeDisabled();
  await expect(dialog).toContainText("Selection changed; cancel and try again.");
  await gui.update(() => {
    window.fixture.PluginStates.streamdeck = "running";
  });
  await expect(page.locator("[data-part=plugin-gate]")).toHaveCount(0);
  await expect(accept).toBeDisabled();
  await page.keyboard.press("Escape");
  await expect(dialog).toBeHidden();
  expect(await gui.sent()).toEqual([]);
});

declare global {
  interface Window {
    drawnSelection: HTMLButtonElement;
  }
}

for (const screen of ["Lights", "Plugins"]) {
  test(`${screen} configuration-only updates refresh selections without retargeting an old callback`, async ({
    gui,
    page,
  }) => {
    await gui.open((fixture) => {
      fixture.ConfigRevision = "drawn-config";
      fixture.PluginStates.hue = "disabled";
      fixture.Enabled.hue = false;
    });
    await gui.screen(screen);
    const enable =
      screen === "Lights"
        ? page.getByRole("button", { name: "Enable Hue", exact: true })
        : page
            .locator("[data-part=plugin]", { has: page.getByRole("heading", { name: "Hue", exact: true }) })
            .getByRole("button", { name: "Enable", exact: true });
    await enable.evaluate((node) => {
      window.drawnSelection = node as HTMLButtonElement;
    });
    const polls = await page.evaluate(() => window.transport.polls);
    await gui.update(() => {
      window.fixture.ConfigRevision = "new-config";
      const desktop = window.go?.app?.Desktop;
      if (!desktop) throw new Error("Desktop missing");
      desktop.Send = async (action) => {
        window.sent.push(action);
        return { ID: action.ID, Status: "Rejected", Detail: "Configuration changed; review the current selection." };
      };
    });
    await expect.poll(() => page.evaluate(() => window.transport.polls)).toBeGreaterThan(polls + 1);
    await expect.poll(() => enable.evaluate((node) => node !== window.drawnSelection)).toBe(true);
    await gui.update(() => window.drawnSelection.click());
    expect((await gui.sent()).at(-1)).toMatchObject({
      Kind: "selection",
      Plugin: "hue",
      Enable: true,
      ConfigRevision: "drawn-config",
    });
    await expect(page.locator("#content")).toContainText("Configuration changed; review the current selection.");
    expect(await gui.sent()).toHaveLength(1);
    await gui.settled();
    await enable.click();
    expect((await gui.sent()).at(-1)).toMatchObject({ ConfigRevision: "new-config" });
    expect(await gui.sent()).toHaveLength(2);
  });
}

test("lifecycle reservation survives configuration and notice updates without blocking navigation", async ({
  gui,
  page,
}) => {
  await gui.open((fixture) => {
    fixture.PluginStates.hue = "disabled";
    fixture.Enabled.hue = false;
    fixture.PluginStates.aec = "failed";
  });
  await gui.screen("Lights");
  await gui.update(() => {
    const desktop = window.go?.app?.Desktop;
    if (!desktop) throw new Error("Desktop missing");
    desktop.Send = (action) => {
      window.sent.push(action);
      return new Promise(() => {});
    };
  });
  await page.getByRole("button", { name: "Enable Hue", exact: true }).click();
  await expect(page.getByRole("button", { name: "Enable Hue", exact: true })).toBeDisabled();
  await gui.update(() => {
    window.fixture.ConfigRevision = "later";
    window.fixture.Notice = "Unrelated notice";
  });
  await gui.screen("Plugins");
  await expect(page.getByRole("button", { name: "Retry Plugins", exact: true })).toBeDisabled();
  await expect(page.locator("[data-part=plugin] button:enabled")).toHaveCount(0);
  expect(await gui.sent()).toHaveLength(1);
});
