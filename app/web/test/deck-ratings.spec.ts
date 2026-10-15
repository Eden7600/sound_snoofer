import { expect, test } from "./host";
import type { DeckView } from "../src/screens/deck-types.ts";

test("rating bindings have distinct thumb previews and retain their action IDs", async ({ gui }) => {
  await gui.open((fixture) => {
    const view = fixture.Controls.find((c) => c.ID === "streamdeck.preview")?.ViewData as DeckView | undefined;
    if (!view) throw new Error("fixture has no deck view");
    for (const [index, id, label, glyph] of [
      [7, "like", "Like", "media-like"],
      [8, "dislike", "Dislike", "media-dislike"],
    ] as const) {
      const slot = view.Keys[index];
      if (!slot) throw new Error("fixture has no rating key position");
      fixture.Controls.push({
        ID: "nowplaying." + id,
        Label: label + " Track",
        ShortLabel: label,
        Kind: "toggle",
        Icon: glyph,
        Value: id === "like" ? "On" : "Off",
        Available: true,
        Revision: 1,
        Operations: ["press"],
      });
      Object.assign(slot, { Control: "nowplaying." + id, Label: label, Source: "", Region: -1 });
    }
  });
  await gui.screen("Stream Deck");
  const binding = gui.page.getByLabel("Binding", { exact: true });
  const shapes: string[] = [];
  for (const [number, id, label, glyph] of [
    [8, "like", "Like", "thumbs-up"],
    [9, "dislike", "Dislike", "thumbs-down"],
  ] as const) {
    const key = gui.page.getByRole("button", { name: `Key ${number}: ${label} Track`, exact: true });
    await expect(key).toContainText(label);
    const icon = key.locator(`svg[data-icon="${glyph}"]`);
    await expect(icon).toHaveAttribute("aria-hidden", "true");
    await expect(icon.locator("path")).toHaveCount(2);
    shapes.push((await icon.innerHTML()).trim());
    await expect(binding.locator(`option[value="nowplaying.${id}"]`)).toHaveText(`${label} Track · Now Playing`);
  }
  expect(shapes[0]).not.toBe(shapes[1]);
  expect(await gui.sent(), "rendering the ratings changed the layout").toEqual([]);
  await binding.selectOption("nowplaying.dislike");
  expect(await gui.last()).toMatchObject({ ID: "streamdeck.binding", Operation: "set", Value: "nowplaying.dislike" });
  await gui.settled();
});
