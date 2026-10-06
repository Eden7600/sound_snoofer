// Builds the Snoofer Media extension for each browser: dist/chrome and
// dist/firefox (load these unpacked while developing), and with --package a
// zip of each for store upload. Manifests are generated here so the shared
// sources stay identical across browsers.
import { cpSync, mkdirSync, readFileSync, readdirSync, rmSync, writeFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const root = dirname(fileURLToPath(import.meta.url));
const { version } = JSON.parse(readFileSync(join(root, "package.json"), "utf8"));

// GECKO_ID is the Firefox add-on ID. It is permanent once published on
// addons.mozilla.org.
export const GECKO_ID = "snoofer-media@sound-snoofer";

const icons = { 16: "icons/icon-16.png", 32: "icons/icon-32.png", 48: "icons/icon-48.png", 128: "icons/icon-128.png" };

function base() {
  return {
    manifest_version: 3,
    name: "Snoofer Media",
    version,
    description: "Shows and controls the media playing in each tab from Snoofer on this computer.",
    icons,
    action: { default_title: "Snoofer Media", default_popup: "popup.html", default_icon: icons },
    permissions: ["tabs", "storage"],
    host_permissions: ["<all_urls>"],
    // No upgrade-insecure-requests: the only connection is ws://127.0.0.1.
    content_security_policy: { extension_pages: "script-src 'self'; object-src 'self'" },
    content_scripts: [
      { matches: ["<all_urls>"], js: ["page.js"], run_at: "document_start", all_frames: true, world: "MAIN" },
      { matches: ["<all_urls>"], js: ["bridge.js"], run_at: "document_start", all_frames: true },
    ],
  };
}

// manifest returns the browser's manifest.
export function manifest(browser) {
  const m = base();
  if (browser === "chrome") {
    m.background = { service_worker: "background.js" };
    m.minimum_chrome_version = "111"; // Main-world content scripts.
  } else if (browser === "firefox") {
    m.background = { scripts: ["logic.js", "background.js"] };
    m.browser_specific_settings = {
      gecko: { id: GECKO_ID, strict_min_version: "128.0", data_collection_permissions: { required: ["none"] } },
    };
  } else {
    throw new Error("unknown browser " + browser);
  }
  return m;
}

// shipped reports whether a source file belongs in the extension.
export function shipped(name) {
  return !name.endsWith(".test.mjs");
}

// build writes dist/<browser> for each browser into out.
export function build(out = join(root, "dist")) {
  for (const browser of ["chrome", "firefox"]) {
    const dir = join(out, browser);
    rmSync(dir, { recursive: true, force: true });
    mkdirSync(dir, { recursive: true });
    for (const entry of readdirSync(join(root, "src"), { withFileTypes: true })) {
      if (entry.isDirectory() || shipped(entry.name)) cpSync(join(root, "src", entry.name), join(dir, entry.name), { recursive: true });
    }
    writeFileSync(join(dir, "manifest.json"), JSON.stringify(manifest(browser), null, 2) + "\n");
  }
  return out;
}

// pack zips each browser folder with Windows' own bsdtar (tar -a picks zip
// from the extension), so packaging needs no extra tools. Git's GNU tar,
// often first on PATH, cannot write zips.
function pack(out) {
  const tar = process.platform === "win32" ? join(process.env.SystemRoot || "C:/Windows", "System32", "tar.exe") : "bsdtar";
  for (const browser of ["chrome", "firefox"]) {
    const zip = `snoofer-media-${browser}-${version}.zip`;
    rmSync(join(out, zip), { force: true });
    // Name entries explicitly so manifest.json sits at the archive root.
    const entries = readdirSync(join(out, browser)).sort();
    execFileSync(tar, ["-a", "-c", "-f", join("..", zip), ...entries], { cwd: join(out, browser), stdio: "inherit" });
    console.log("Packaged " + join(out, zip));
  }
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const out = build();
  console.log("Built " + join(out, "chrome") + " and " + join(out, "firefox"));
  if (process.argv.includes("--package")) pack(out);
}
