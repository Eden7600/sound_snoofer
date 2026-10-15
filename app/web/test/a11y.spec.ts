import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "./host";
import { knownA11y } from "./known-a11y";

const screens = ["Audio", "Routing", "Soundboard", "Lights", "App Audio", "Meetings", "Media", "Stream Deck", "Plugins", "Third-Party Apps", "Diagnostics"];
const wcagTags = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"];

test("accessibility exceptions name a scanned screen and owning change", () => {
  for (const entry of knownA11y) {
    expect(screens).toContain(entry.screen);
    expect(entry.owner).toMatch(/^gui-[a-z]+(?:-[a-z]+)*$/);
    expect(entry.rule).not.toBe("");
    expect(entry.target.length).toBeGreaterThan(0);
  }
});

test("accessibility scanner detects an unlabeled control", async ({ gui }) => {
  await gui.open();
  await gui.page.locator("#content").evaluate((content) => {
    const input = document.createElement("input");
    input.id = "a11y-regression";
    content.prepend(input);
  });
  const results = await new AxeBuilder({ page: gui.page }).withTags(wcagTags).analyze();
  expect(results.violations.find((violation) => violation.id === "label")?.nodes.map((node) => node.target))
    .toContainEqual(["#a11y-regression"]);
});

for (const screen of screens) {
  test(`${screen} has no new or obsolete accessibility exceptions`, async ({ gui }, testInfo) => {
    await gui.open();
    await gui.screen(screen);
    await gui.page.locator("#content details").evaluateAll((details) => {
      for (const detail of details) detail.setAttribute("open", "");
    });
    const results = await new AxeBuilder({ page: gui.page })
      .withTags(wcagTags)
      .analyze();
    await testInfo.attach("accessibility", { body: JSON.stringify(results, null, 2), contentType: "application/json" });
    const actual = results.violations.flatMap((violation) =>
      violation.nodes.map((node) => JSON.stringify({ rule: violation.id, target: node.target })),
    ).sort();
    const expected = knownA11y.filter((entry) => entry.screen === screen)
      .map(({ rule, target }) => JSON.stringify({ rule, target })).sort();
    expect(actual, "Update an exact exception only for a reviewed existing defect; remove exceptions when fixed").toEqual(expected);
  });
}
