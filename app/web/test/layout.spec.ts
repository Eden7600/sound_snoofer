import { expect, test } from "./host";

// Runs in the browser so measurements include the real generated CSS.
function layoutIssues(): string[] {
  const issues: string[] = [];
  const tolerance = 1;
  const name = (element: Element): string => {
    const label =
      element.getAttribute("aria-label") ||
      element.id ||
      element.getAttribute("data-screen") ||
      element.textContent?.trim().replace(/\s+/g, " ").slice(0, 60) ||
      element.getAttribute("type") ||
      "unlabelled";
    return `${element.tagName.toLowerCase()}[${label}]`;
  };
  if (document.documentElement.scrollWidth > innerWidth + tolerance) issues.push("page: horizontal overflow");
  const controls = [
    ...document.querySelectorAll<HTMLElement>("button, input, select, textarea, a[href], summary, [role=button]"),
  ]
    .filter((element) => element.checkVisibility({ visibilityProperty: true }))
    .map((element) => ({ element, rect: element.getBoundingClientRect(), name: name(element) }));
  for (const control of controls) {
    const { rect, element } = control;
    if (rect.left < -tolerance || rect.right > innerWidth + tolerance)
      issues.push(`${control.name}: outside viewport width`);
    for (let parent = element.parentElement; parent; parent = parent.parentElement) {
      const style = getComputedStyle(parent);
      const bounds = parent.getBoundingClientRect();
      const left = bounds.left + parent.clientLeft;
      const top = bounds.top + parent.clientTop;
      const clipsX =
        ["hidden", "clip"].includes(style.overflowX) &&
        (rect.left < left - tolerance || rect.right > left + parent.clientWidth + tolerance);
      const clipsY =
        ["hidden", "clip"].includes(style.overflowY) &&
        (rect.top < top - tolerance || rect.bottom > top + parent.clientHeight + tolerance);
      if (clipsX || clipsY)
        issues.push(`${control.name}: clipped by ${name(parent)} (${clipsX ? "x" : ""}${clipsY ? "y" : ""})`);
    }
  }
  for (let i = 0; i < controls.length; i++) {
    for (const other of controls.slice(i + 1)) {
      const control = controls[i]!;
      const width = Math.min(control.rect.right, other.rect.right) - Math.max(control.rect.left, other.rect.left);
      const height = Math.min(control.rect.bottom, other.rect.bottom) - Math.max(control.rect.top, other.rect.top);
      if (width > tolerance && height > tolerance) issues.push(`${control.name}: overlaps ${other.name}`);
    }
  }
  return issues.sort();
}

const screens = [
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
];
// gui-layout-polish owns audit item 13: the sticky save bar covers the dials at the top of the page.
const knownDeckOverlaps = [
  "button[Dial 5: Empty]: overlaps button[Discard]",
  "button[Dial 5: Empty]: overlaps button[Save Layout]",
];

for (const viewport of [
  { width: 1280, height: 820 },
  { width: 800, height: 600 },
]) {
  test.describe(`Layout at ${viewport.width}×${viewport.height}`, () => {
    test.use({ viewport });
    for (const screen of screens) {
      test(`${screen} has no new layout defects`, async ({ gui }, testInfo) => {
        await gui.open();
        await gui.screen(screen);
        await gui.page.evaluate(() => document.fonts.ready);
        await gui.page.evaluate(() => window.scrollTo(0, 0));
        const issues = await gui.page.evaluate(layoutIssues);
        const screenshot = testInfo.outputPath(`${screen}-${viewport.width}x${viewport.height}.png`);
        await gui.page.screenshot({ path: screenshot, fullPage: true });
        await testInfo.attach(`${screen}-${viewport.width}x${viewport.height}`, {
          path: screenshot,
          contentType: "image/png",
        });
        const known = screen === "Stream Deck" && viewport.width === 800 ? knownDeckOverlaps : [];
        if (known.length)
          testInfo.annotations.push({
            type: "known-layout",
            description: "gui-layout-polish: audit item 13, sticky save bar overlaps Dial 5 at scrollY=0",
          });
        expect(issues, "Unexpected or resolved layout issue; update only the exact owned entries").toEqual(known);
      });
    }
  });
}

test("geometry checks detect overflow, clipping and overlap", async ({ page }) => {
  await page.setContent(
    '<!doctype html><button style="position:absolute;left:10px;top:10px;width:100px;height:30px">First</button><button style="position:absolute;left:120px;top:10px;width:100px;height:30px">Second</button><div id="clip" style="position:absolute;left:10px;top:50px;width:120px;height:40px;overflow:hidden"><input aria-label="Clipped" style="width:100px;height:20px"></div>',
  );
  expect(await page.evaluate(layoutIssues)).toEqual([]);
  await page
    .locator("button")
    .nth(1)
    .evaluate((element) => {
      element.style.left = "80px";
    });
  await page.locator("input").evaluate((element) => {
    element.style.width = "200px";
  });
  expect(await page.evaluate(layoutIssues)).toEqual([
    "button[First]: overlaps button[Second]",
    "input[Clipped]: clipped by div[clip] (x)",
  ]);
  await page.locator("input").evaluate((element) => {
    element.style.height = "60px";
  });
  expect(await page.evaluate(layoutIssues)).toEqual([
    "button[First]: overlaps button[Second]",
    "input[Clipped]: clipped by div[clip] (xy)",
  ]);
  await page
    .locator("button")
    .nth(1)
    .evaluate((element) => {
      element.style.left = `${innerWidth}px`;
    });
  expect(await page.evaluate(layoutIssues)).toEqual([
    "button[Second]: outside viewport width",
    "input[Clipped]: clipped by div[clip] (xy)",
    "page: horizontal overflow",
  ]);
});
