import { expect, test, type Gui } from "./host.ts";
import type { AppAudioView } from "../src/screens/appaudio-types.ts";

async function rebuild(gui: Gui) {
  const old = await gui.page.locator("#content > :first-child").elementHandle();
  if (!old) throw new Error("content missing");
  await gui.update(() => {
    window.ctl("aec.strength").Label += " updated";
  });
  await expect.poll(() => old.evaluate((node) => node.isConnected)).toBe(false);
}

async function pageName(gui: Gui) {
  await gui.open();
  await gui.screen("Stream Deck");
  await gui.page.getByText("Page Options", { exact: true }).click();
  return gui.page.getByRole("textbox", { name: "Name", exact: true });
}

test("unfocused semantic text retains its draft, selection and original revision through rebuild", async ({ gui }) => {
  const name = await pageName(gui);
  const revision = await gui.page.evaluate(() => window.ctl("streamdeck.name").Revision);
  await name.fill("My page draft");
  await name.evaluate((node: HTMLInputElement) => node.setSelectionRange(3, 7, "backward"));
  await name.blur();
  await gui.update(() => {
    window.ctl("streamdeck.name").Revision++;
  });
  await rebuild(gui);
  await expect(name).toHaveValue("My page draft");
  expect(
    await name.evaluate((node: HTMLInputElement) => [node.selectionStart, node.selectionEnd, node.selectionDirection]),
  ).toEqual([3, 7, "backward"]);
  expect(await gui.sent()).toEqual([]);
  await name.focus();
  await name.press("Enter");
  expect(await gui.last()).toMatchObject({ ID: "streamdeck.name", Revision: revision, Value: "My page draft" });
});

test("queue acknowledgment retains text until observation, and Escape discards a draft", async ({ gui }) => {
  const name = await pageName(gui);
  await gui.update(() => {
    const desktop = window.go?.app?.Desktop;
    if (!desktop) throw new Error("desktop missing");
    desktop.Send = async (action) => {
      window.sent.push(action);
      return { ID: action.ID, Status: "Submitted", Detail: "Queued" };
    };
  });
  await name.fill("Queued name");
  await name.press("Enter");
  await gui.update(() => {
    window.ctl("streamdeck.name").Revision++;
  });
  await gui.settled();
  await rebuild(gui);
  await expect(name).toHaveValue("Queued name");
  const polls = await gui.update(() => {
    window.ctl("streamdeck.name").Value = "Queued name";
    return window.transport.polls;
  });
  await expect.poll(() => gui.page.evaluate(() => window.transport.polls)).toBeGreaterThan(polls + 1);
  await gui.update(() => {
    window.ctl("streamdeck.name").Value = "Observed later";
  });
  await expect(name).toHaveValue("Observed later");
  await name.fill("Cancel me");
  await name.press("Escape");
  await rebuild(gui);
  await expect(name).toHaveValue("Observed later");
  expect(await gui.sent()).toHaveLength(1);
});

for (const context of ["page", "profile"]) {
  test(`semantic text does not transfer to a different Deck ${context}`, async ({ gui }) => {
    const name = await pageName(gui);
    await name.fill("Wrong target");
    await name.blur();
    await gui.update((context) => {
      window.ctl("streamdeck." + context).Value += "-other";
      window.ctl("streamdeck.name").Value = "Other target";
    }, context);
    await rebuild(gui);
    await gui.page.getByText("Page Options", { exact: true }).click();
    await expect(name).toHaveValue("Other target");
    expect(await gui.sent()).toEqual([]);
  });
}

test("page-name drafts survive slot browsing but clear on workspace departure", async ({ gui }) => {
  const name = await pageName(gui);
  await name.fill("Keep this page");
  await name.blur();
  await gui.page.getByRole("button", { name: /^Key 2:/ }).click();
  await gui.settled();
  await expect(name).toHaveValue("Keep this page");
  await gui.screen("Audio");
  await gui.screen("Stream Deck");
  await gui.page.getByText("Page Options", { exact: true }).click();
  await expect(name).toHaveValue("Home");
  expect(await gui.sent()).toHaveLength(1);
});

test("remembered-deck text does not transfer to another serial", async ({ gui }) => {
  await pageName(gui);
  const name = gui.page.getByRole("textbox", { name: "Deck Name", exact: true });
  await name.fill("Old hardware");
  await name.blur();
  await gui.update(() => {
    window.ctl("streamdeck.deck").Value = "other-serial";
    window.ctl("streamdeck.deck-name").Value = "Other hardware";
  });
  await expect(name).toHaveValue("Other hardware");
  expect(await gui.sent()).toEqual([]);
});

for (const label of ["Home", " Home "]) {
  test(`New Page ${JSON.stringify(label)} clears only after a new exact matching page is observed`, async ({ gui }) => {
    await pageName(gui);
    const input = gui.page.getByRole("textbox", { name: "New Page", exact: true });
    await gui.update(() => {
      const desktop = window.go?.app?.Desktop;
      if (!desktop) throw new Error("desktop missing");
      desktop.Send = async (action) => {
        window.sent.push(action);
        return { ID: action.ID, Status: "Submitted", Detail: "Queued" };
      };
    });
    await input.fill(label);
    await input.press("Enter");
    await gui.update(() => {
      window.ctl("streamdeck.add").Revision++;
    });
    await gui.settled();
    await rebuild(gui);
    await expect(input).toHaveValue(label);
    await gui.update((label) => {
      const page = window.ctl("streamdeck.page");
      (page.Options ??= []).push("page-new");
      (page.OptionLabels ??= {})["page-new"] = label;
      page.Value = "page-new";
    }, label);
    await expect(gui.page.locator("#field-streamdeck\\.add")).toHaveValue("");
    expect(await gui.sent()).toHaveLength(1);
  });
}

test("App Audio Rename and Combine survive unrelated changes with the captured edit revision", async ({ gui }) => {
  await gui.open();
  await gui.screen("App Audio");
  const app = gui.page.locator("[data-part=app]", {
    has: gui.page.getByRole("heading", { name: "Discord", exact: true }),
  });
  await app.locator("summary").click();
  const rename = gui.page.getByRole("textbox", { name: "Rename Discord", exact: true });
  const combine = gui.page.getByRole("combobox", { name: "Combine Discord into", exact: true });
  const revision = await gui.page.evaluate(() => window.ctl("appaudio.edit").Revision);
  await rename.fill("Calls");
  await rename.blur();
  await combine.selectOption("Spotify");
  await combine.blur();
  await gui.update(() => {
    window.ctl("appaudio.edit").Revision++;
  });
  await rebuild(gui);
  await expect(rename).toHaveValue("Calls");
  await expect(combine).toHaveValue("Spotify");
  expect(await gui.sent()).toEqual([]);
  await app.getByRole("button", { name: "Rename", exact: true }).click();
  expect(await gui.last()).toMatchObject({
    ID: "appaudio.edit",
    Revision: revision,
    Value: JSON.stringify({ op: "rename", app: "Discord", value: "Calls" }),
  });
});

test("a renamed app does not inherit the old payload's draft", async ({ gui }) => {
  await gui.open();
  await gui.screen("App Audio");
  const app = gui.page.locator("[data-part=app]").first();
  await app.locator("summary").click();
  await gui.page.getByRole("textbox", { name: "Rename Discord", exact: true }).fill("Wrong target");
  await gui.page.getByRole("textbox", { name: "Rename Discord", exact: true }).blur();
  await gui.update(() => {
    window.ctl("appaudio.app-discord").Label = "Chat";
    const view = window.ctl<AppAudioView>("appaudio.status").ViewData;
    const app = view?.Apps?.find((app) => app.ID === "appaudio.app-discord");
    if (!app) throw new Error("app missing");
    app.Name = "Chat";
  });
  await expect(gui.page.getByRole("heading", { name: "Chat", exact: true })).toBeVisible();
  await app.locator("summary").click();
  await expect(gui.page.getByRole("textbox", { name: "Rename Chat", exact: true })).toHaveValue("");
  expect(await gui.sent()).toEqual([]);
});

test("a removed Combine choice clears instead of selecting its neighbor", async ({ gui }) => {
  await gui.open();
  await gui.screen("App Audio");
  const app = gui.page.locator("[data-part=app]").first();
  await app.locator("summary").click();
  const combine = gui.page.getByRole("combobox", { name: "Combine Discord into", exact: true });
  await combine.selectOption("Google Chrome");
  await combine.blur();
  await gui.update(() => {
    const view = window.ctl<AppAudioView>("appaudio.status").ViewData;
    if (!view) throw new Error("view missing");
    view.Apps = (view.Apps || []).filter((app) => app.Name !== "Google Chrome");
  });
  await expect(combine.locator("option[value='Google Chrome']")).toHaveCount(0);
  await expect(combine).toHaveValue("");
  await expect(app.getByRole("button", { name: "Combine", exact: true })).toBeDisabled();
  expect(await gui.sent()).toEqual([]);
});

test("the Excluded Add draft survives rebuild and clears only when its pattern is observed", async ({ gui }) => {
  await gui.open();
  await gui.screen("App Audio");
  const input = gui.page.getByRole("textbox", { name: "Program to Exclude", exact: true });
  await input.fill("steam.exe");
  await input.blur();
  await rebuild(gui);
  await expect(input).toHaveValue("steam.exe");
  const apply = gui.page.getByRole("button", { name: "Exclude", exact: true });
  await expect(apply).toBeEnabled();
  await apply.click();
  await gui.settled();
  await expect(input).toHaveValue("steam.exe");
  await gui.update(() => {
    const view = window.ctl<AppAudioView>("appaudio.status").ViewData;
    if (!view) throw new Error("view missing");
    (view.Exclude ??= []).push("steam.exe");
  });
  await expect(input).toHaveValue("");
  await expect(apply).toBeDisabled();
  expect(await gui.sent()).toHaveLength(1);
});
