import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import vm from "node:vm";

test("playback command applies captured intent instead of toggling current state", () => {
  const listeners = new Map();
  const element = { isConnected: true, paused: false, ended: false, duration: 20, plays: 0, pauses: 0,
    play() { this.plays++; return Promise.resolve(); }, pause() { this.pauses++; } };
  const context = vm.createContext({ navigator: {}, document: { addEventListener() {}, querySelectorAll: () => [element] },
    window: { __snooferMedia: true, addEventListener: (name, handler) => listeners.set(name, handler) }, setTimeout: () => 1 });
  vm.runInContext(readFileSync(new URL("page.js", import.meta.url), "utf8"), context);
  assert.equal(context.window.__snooferMedia, 2, "new helper installs over the old version marker");
  assert.equal(listeners.has("snoofer-media:command"), false, "new helper does not share the old toggle event");
  const command = listeners.get("snoofer-media:command-v2");
  command({ detail: JSON.stringify({ op: "toggle", value: 1 }) });
  assert.equal(element.plays, 1);
  assert.equal(element.pauses, 0);
  element.paused = true;
  command({ detail: JSON.stringify({ op: "toggle", value: 0 }) });
  assert.equal(element.pauses, 1);
  command({ detail: JSON.stringify({ op: "toggle" }) });
  assert.equal(element.plays, 1);
  assert.equal(element.pauses, 1);
});
