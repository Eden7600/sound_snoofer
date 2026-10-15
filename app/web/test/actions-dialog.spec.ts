import { expect, test } from "./host.ts";
import { holdActions } from "./action-host.ts";

test("host confirmation failures remain visible inside the still-open dialog", async ({ gui }) => {
  await gui.open((fixture) => {
    fixture.Confirmation = "";
    fixture.ProposalID = "proposal-one";
  });
  await gui.screen("Diagnostics");
  await gui.update(() => {
    window.fixture.Confirmation = "Enable Soundboard?";
  });
  await holdActions(gui);
  const dialog = gui.page.getByRole("dialog");
  await dialog.getByRole("button", { name: "Apply & restart", exact: true }).click();
  await gui.update(() => {
    const action = window.sent.at(-1);
    if (!action) throw new Error("Missing confirmation");
    window.actionReplies[action.ID]?.({ ID: action.ID, Status: "Failed", Detail: "Configuration could not be saved" });
  });
  await expect(dialog).toContainText("Configuration could not be saved");
  await expect(dialog.getByRole("button", { name: "Cancel", exact: true })).toBeEnabled();
});

test("an unrelated action does not disable a local captured confirmation", async ({ gui }) => {
  await gui.open();
  await holdActions(gui);
  await gui.page.getByRole("button", { name: "Mic Mute", exact: true }).click();
  await gui.screen("Stream Deck");
  await gui.page.getByText("Page Options", { exact: true }).click();
  await gui.page.getByRole("button", { name: "Delete Page", exact: true }).click();
  const dialog = gui.page.getByRole("dialog");
  await expect(dialog.getByRole("button", { name: "Delete Page", exact: true })).toBeEnabled();
  await dialog.getByRole("button", { name: "Delete Page", exact: true }).click();
  expect((await gui.sent()).map((action) => action.Request?.ID)).toEqual(["audio.mic-mute", "streamdeck.delete"]);
});
