import { defineConfig } from "@playwright/test";

// The GUI suite. Run from the repository root:
//   npx playwright test -c app/web/test
export default defineConfig({
  testDir: ".",
  globalSetup: "./global-setup.ts",
  timeout: 15_000,
  fullyParallel: true,
  forbidOnly: true,
  reporter: [["list"]],
  outputDir: "../../../.local/gui-test-results",
  use: {
    // The installed Chrome only: the suite never downloads a browser.
    channel: "chrome",
    headless: true,
    viewport: { width: 1280, height: 820 },
    screenshot: "only-on-failure",
  },
});
