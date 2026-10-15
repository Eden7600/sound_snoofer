import { expect, test } from "./host.ts";
import { holdActions, reply } from "./action-host.ts";
import type { MediaView } from "../src/screens/media-types.ts";

test("a global media alias keeps its initiating card when focus changes before a delayed failure", async ({ gui }) => {
  await gui.open((fixture) => {
    const view = fixture.Controls.find((control) => control.ID === "nowplaying.status")?.ViewData as MediaView;
    for (const session of view.Sessions || []) session.Focused = session.ID === "nowplaying.s-spotify";
  });
  await gui.screen("Media");
  await holdActions(gui);
  const first = gui.page.locator("[data-part=media-session]").nth(0);
  const second = gui.page.locator("[data-part=media-session]").nth(1);
  await second.getByRole("button", { name: "Next", exact: true }).click();
  await expect(second.locator('[data-action-feedback="nowplaying.next"]')).toContainText("Sending");
  await expect(first.locator('[data-action-feedback="nowplaying.next"]')).toBeHidden();
  await gui.update(() => {
    const view = window.ctl<MediaView>("nowplaying.status").ViewData;
    for (const session of view?.Sessions || []) session.Focused = session.ID === "nowplaying.s-tab";
  });
  await expect(first).toHaveAttribute("data-focused", "true");
  await reply(gui, "nowplaying.next", "Failed", "Spotify could not advance");
  await expect(second).toContainText("Spotify could not advance");
  await expect(first).not.toContainText("Spotify could not advance");
});

test("unavailable media targets disable their exact controls despite advertised view capabilities", async ({ gui }) => {
  await gui.open();
  await gui.screen("Media");
  await gui.update(() => {
    window.ctl("nowplaying.s-tab").Available = false;
    for (const id of ["nowplaying.prev", "nowplaying.next", "nowplaying.mute"]) window.ctl(id).Available = false;
  });
  const card = gui.page.locator("[data-part=media-session]").first();
  await expect(card.locator("[data-part=play]")).toBeDisabled();
  await expect(card.locator("[data-part=seek]")).toBeDisabled();
  await expect(card.getByRole("button", { name: "Next", exact: true })).toBeDisabled();
  await expect(card.locator("[data-part=mute]")).toBeDisabled();
  expect(await gui.sent()).toEqual([]);
});
