import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Gui } from "./host";

const actions = [
  {
    id: "streamdeck.delete",
    name: "Delete Page",
    title: "Delete page?",
    accept: "Delete Page",
    warning: "Delete this page from the draft? Save applies the deletion.",
  },
  {
    id: "streamdeck.deck-forget",
    name: "Forget Deck",
    title: "Forget deck?",
    accept: "Forget Deck",
    warning: "Forget this deck and its own layout?",
  },
  {
    id: "streamdeck.deck-own",
    name: "Own Layout",
    title: "Use model layout?",
    accept: "Use Model Layout",
    warning: "Use the model layout on this deck? Its own layout is deleted.",
  },
] as const;
type DeckAction = (typeof actions)[number];

async function open(gui: Gui, action: DeckAction = actions[0]) {
  const invoker = gui.page.locator("#content").getByRole("button", { name: action.name, exact: true });
  await invoker.click();
  const dialog = gui.page.getByRole("dialog", { name: action.title, exact: true });
  await expect(dialog).toBeVisible();
  await expect(dialog).toContainText(action.warning);
  return { dialog, invoker, accept: dialog.getByRole("button", { name: action.accept, exact: true }) };
}

test.beforeEach(async ({ gui }) => {
  await gui.open((fixture) => {
    const deletion = fixture.Controls.find((control) => control.ID === "streamdeck.delete");
    if (!deletion) throw new Error("missing page deletion");
    deletion.Available = true;
  });
  await gui.screen("Stream Deck");
  await gui.page.getByText("Page Options", { exact: true }).click();
});

for (const action of actions) {
  test(`${action.name} has a local Cancel default and Escape returns focus without dispatch`, async ({ gui }) => {
    const { dialog, invoker } = await open(gui, action);
    await expect(dialog.getByRole("button", { name: "Cancel", exact: true })).toBeFocused();
    await gui.page.keyboard.press("Escape");
    await expect(dialog).toBeHidden();
    await expect(invoker).toBeFocused();
    expect(await gui.sent()).toEqual([]);
    await open(gui, action);
    await gui.page.keyboard.press("Enter");
    await expect(dialog).toBeHidden();
    expect(await gui.sent()).toEqual([]);
  });

  test(`${action.name} accepts its captured request exactly once`, async ({ gui }) => {
    const revision = await gui.page.evaluate((id) => window.ctl(id).Revision, action.id);
    const { dialog, accept } = await open(gui, action);
    await accept.evaluate((button) => {
      (button as HTMLButtonElement).click();
      (button as HTMLButtonElement).click();
    });
    await expect(dialog).toBeHidden();
    expect(await gui.sent()).toHaveLength(1);
    expect(await gui.last()).toMatchObject({ ID: action.id, Operation: "press", Revision: revision });
    if (action.id === "streamdeck.delete")
      expect((await gui.sent()).some((item) => item.Request?.ID === "streamdeck.save")).toBe(false);
  });
}

test("polling and rebuilds continue behind a local dialog without stealing focus", async ({ gui }) => {
  const { dialog, invoker } = await open(gui);
  const oldInvoker = await invoker.elementHandle();
  const polls = await gui.page.evaluate(() => window.transport.polls);
  await gui.update(() => {
    window.ctl("aec.strength").Label += " changed";
  });
  await expect.poll(() => oldInvoker?.evaluate((node) => node.isConnected)).toBe(false);
  await expect.poll(() => gui.page.evaluate(() => window.transport.polls)).toBeGreaterThan(polls + 1);
  await expect(dialog.getByRole("button", { name: "Cancel", exact: true })).toBeFocused();
  await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
  await expect(invoker).toBeFocused();
  expect(await gui.sent()).toEqual([]);
});

for (const change of ["revision", "removed", "unavailable", "operation", "page", "profile", "serial"] as const) {
  test(`a changed ${change} invalidates the captured confirmation`, async ({ gui }) => {
    const action = change === "serial" ? actions[1] : actions[0];
    const { dialog, accept } = await open(gui, action);
    await gui.update(
      ([change, id]) => {
        if (change === "revision") window.ctl(id).Revision++;
        if (change === "removed")
          window.fixture.Controls = window.fixture.Controls.filter((control) => control.ID !== id);
        if (change === "unavailable") window.ctl(id).Available = false;
        if (change === "operation") window.ctl(id).Operations = [];
        if (change === "page") window.ctl("streamdeck.page").Value = "another-page";
        if (change === "profile") window.ctl("streamdeck.profile").Value = "serial:another-deck";
        if (change === "serial") window.ctl("streamdeck.deck").Value = "another-serial";
      },
      [change, action.id] as const,
    );
    await expect(accept).toBeDisabled();
    await expect(dialog).toContainText("Selection changed; cancel and try again.");
    await accept.evaluate((button) => (button as HTMLButtonElement).click());
    expect(await gui.sent()).toEqual([]);
    await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
    await expect(dialog).toBeHidden();
    if (change === "removed") await expect(gui.page.locator("#content")).toBeFocused();
  });
}

test("disconnection permanently invalidates a local ticket without replay on recovery", async ({ gui }) => {
  const { dialog, accept } = await open(gui);
  await gui.update(() => {
    const desktop = window.go?.app?.Desktop;
    if (!desktop) throw new Error("missing desktop");
    const state = desktop.State.bind(desktop);
    desktop.State = async (...args) => {
      if (window.fixture.Notice === "disconnect fixture") throw new Error("fixture disconnected");
      return state(...args);
    };
    window.fixture.Notice = "disconnect fixture";
  });
  await expect(gui.page.locator("#connection")).toBeVisible();
  await expect(accept).toBeDisabled();
  await gui.update(() => {
    window.fixture.Notice = "";
  });
  await expect(gui.page.locator("#connection")).toBeHidden();
  await expect(accept).toBeDisabled();
  expect(await gui.sent()).toEqual([]);
  await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
});

test("host confirmation preempts a local ticket without sending host Cancel", async ({ gui }) => {
  await open(gui);
  await gui.update(() => {
    window.fixture.Confirmation = "Enable plugin and restart?";
  });
  const host = gui.page.getByRole("dialog", { name: "Apply changes?", exact: true });
  await expect(host).toContainText("Enable plugin and restart?");
  await expect(host.getByRole("button", { name: "Cancel", exact: true })).toBeFocused();
  expect(await gui.sent()).toEqual([]);
  await gui.page.keyboard.press("Escape");
  await expect
    .poll(() => gui.sent())
    .toEqual([expect.objectContaining({ ID: expect.any(String), Kind: "cancel", ProposalID: "" })]);
  await gui.update(() => {
    window.fixture.Confirmation = "";
  });
  await expect(gui.page.getByRole("dialog")).toBeHidden();
});

test("enabling Own Layout remains direct", async ({ gui }) => {
  await gui.update(() => {
    window.ctl("streamdeck.deck-own").Value = "Off";
  });
  const own = gui.page.getByRole("button", { name: "Own Layout", exact: true });
  await expect(own).toHaveAttribute("aria-pressed", "false");
  await own.click();
  expect((await gui.last()).ID).toBe("streamdeck.deck-own");
  await expect(gui.page.getByRole("dialog")).toBeHidden();
});

test("explicit navigation closes a local ticket without dispatch", async ({ gui }) => {
  await open(gui);
  await gui.page.locator('[data-screen="audio"]').evaluate((button) => (button as HTMLButtonElement).click());
  await expect(gui.page.getByRole("dialog")).toBeHidden();
  await expect(gui.page.locator("#title")).toHaveText("Audio");
  expect(await gui.sent()).toEqual([]);
});

test("closing the local dialog cancels without dispatch", async ({ gui }) => {
  const { dialog, invoker } = await open(gui);
  await dialog.evaluate((node) => (node as HTMLDialogElement).close());
  await expect(dialog).toBeHidden();
  await expect(invoker).toBeFocused();
  expect(await gui.sent()).toEqual([]);
});

for (const size of [
  { width: 1280, height: 820 },
  { width: 800, height: 600 },
]) {
  test(`local confirmation fits and remains accessible at ${size.width}px`, async ({ gui }, info) => {
    await gui.page.setViewportSize(size);
    const { dialog } = await open(gui, actions[2]);
    const bounds = await dialog.boundingBox();
    expect(bounds).not.toBeNull();
    if (!bounds) throw new Error("dialog not visible");
    expect(bounds.x).toBeGreaterThanOrEqual(0);
    expect(bounds.y).toBeGreaterThanOrEqual(0);
    expect(bounds.x + bounds.width).toBeLessThanOrEqual(size.width);
    expect(bounds.y + bounds.height).toBeLessThanOrEqual(size.height);
    const results = await new AxeBuilder({ page: gui.page })
      .include("#confirm")
      .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"])
      .analyze();
    expect(results.violations).toEqual([]);
    const screenshot = info.outputPath("local-confirmation.png");
    await gui.page.screenshot({ path: screenshot });
    await info.attach("local-confirmation", { path: screenshot, contentType: "image/png" });
  });
}
