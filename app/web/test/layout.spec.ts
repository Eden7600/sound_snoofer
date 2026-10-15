import { test } from "./host";

test.describe("Layout at 800×600", () => {
  test.use({ viewport: { width: 800, height: 600 } });

  for (const screen of ["Audio", "Stream Deck", "Lights", "Meetings", "Third-Party Apps"]) {
    test(`${screen} has no horizontal scroll`, async ({ gui }) => {
      await gui.open();
      await gui.screen(screen);
      await gui.noHorizontalScroll();
    });
  }
});
