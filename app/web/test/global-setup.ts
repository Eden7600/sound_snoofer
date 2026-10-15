import { execSync } from "node:child_process";
import path from "node:path";
import { chromium } from "@playwright/test";

const repository = path.resolve(__dirname, "../../..");

export default async function globalSetup(): Promise<void> {
  // Build both embedded assets so tests exercise the packaged source.
  execSync("npm run --silent build:gui", { cwd: repository, stdio: "inherit" });
  try {
    const browser = await chromium.launch({ channel: "chrome" });
    await browser.close();
  } catch (error) {
    throw new Error(`Install Google Chrome; the GUI suite never downloads a browser.\n${String(error)}`, {
      cause: error,
    });
  }
}
