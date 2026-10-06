import { test } from "node:test";
import assert from "node:assert/strict";
import { existsSync, mkdtempSync, readFileSync, readdirSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { GECKO_ID, build, manifest } from "./build.mjs";

test("manifests differ only where the browsers do", () => {
  const chrome = manifest("chrome"), firefox = manifest("firefox");
  assert.deepEqual(chrome.background, { service_worker: "background.js" });
  assert.deepEqual(firefox.background, { scripts: ["logic.js", "background.js"] });
  assert.equal(firefox.browser_specific_settings.gecko.id, GECKO_ID);
  assert.equal(chrome.browser_specific_settings, undefined);
  for (const m of [chrome, firefox]) {
    assert.equal(m.manifest_version, 3);
    assert.deepEqual(m.permissions, ["tabs", "storage"]);
    assert.equal(m.content_scripts[0].world, "MAIN");
    assert.doesNotMatch(m.content_security_policy.extension_pages, /upgrade-insecure-requests/);
    delete m.background;
    delete m.browser_specific_settings;
    delete m.minimum_chrome_version;
  }
  assert.deepEqual(chrome, firefox);
});

test("build writes complete folders without tests", () => {
  const out = mkdtempSync(join(tmpdir(), "snoofer-media-"));
  try {
    build(out);
    for (const browser of ["chrome", "firefox"]) {
      const files = readdirSync(join(out, browser)).sort();
      assert.deepEqual(files, ["background.js", "bridge.js", "icons", "logic.js", "manifest.json", "page.js", "popup.html", "popup.js"]);
      const m = JSON.parse(readFileSync(join(out, browser, "manifest.json"), "utf8"));
      for (const icon of Object.values(m.icons)) assert.ok(existsSync(join(out, browser, icon)), icon);
    }
  } finally {
    rmSync(out, { recursive: true, force: true });
  }
});
