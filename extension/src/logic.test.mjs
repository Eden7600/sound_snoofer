import { test } from "node:test";
import assert from "node:assert/strict";
import "./logic.js";

const { PROTOCOL, DEFAULT_PORT, backoff, browserName, route, sessions, validPort } = globalThis.SnooferLogic;

test("protocol and port match Snoofer", () => {
  assert.equal(PROTOCOL, 1);
  assert.equal(DEFAULT_PORT, 47815);
  assert.equal(validPort("50000"), 50000);
  assert.equal(validPort(80), 0);
  assert.equal(validPort("x"), 0);
  assert.equal(validPort(70000), 0);
});

test("browser names pair with Windows sessions", () => {
  assert.equal(browserName([{ brand: "Chromium" }, { brand: "Brave" }], false, ""), "Brave");
  assert.equal(browserName([{ brand: "Chromium" }], true, ""), "Brave");
  assert.equal(browserName([{ brand: "Microsoft Edge" }, { brand: "Chromium" }], false, ""), "Edge");
  assert.equal(browserName([{ brand: "Google Chrome" }], false, ""), "Chrome");
  assert.equal(browserName(undefined, false, "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:140.0) Gecko/20100101 Firefox/140.0"), "Firefox");
  assert.equal(browserName(undefined, false, ""), "Chrome");
});

test("sessions merge frame reports with tab details", () => {
  const frames = new Map([
    ["7:0", { tabId: 7, report: { present: true, title: "", artist: "Band", state: "playing", positionMs: 1000, durationMs: 9000, at: 5, canNext: true, artwork: "" } }],
    ["7:3", { tabId: 7, report: { present: false } }],
    ["9:0", { tabId: 9, report: null }],
    ["8:0", { tabId: 8, report: { present: true, title: "Song", state: "paused", artwork: "https://cdn/cover.jpg" } }],
  ]);
  const tabs = new Map([
    [7, { title: "Tab title", url: "https://www.youtube.com/watch?v=x", muted: true, favIconUrl: "https://youtube.com/favicon.ico" }],
    [8, { title: "Other", url: "not a url", muted: false }],
  ]);
  const list = sessions(frames, tabs);
  assert.deepEqual(list.map((s) => s.id), ["7:0", "8:0"]);
  assert.equal(list[0].title, "Tab title", "falls back to the tab title");
  assert.equal(list[0].site, "youtube.com");
  assert.equal(list[0].muted, true);
  assert.equal(list[0].artKey, "https://youtube.com/favicon.ico", "favicon when the page has no artwork");
  assert.equal(list[1].artKey, "https://cdn/cover.jpg");
  assert.equal(list[1].site, "");
  assert.equal(list[1].state, "paused");
});

test("commands route to frames or the tabs API", () => {
  const port = {};
  const frames = new Map([["7:0", { tabId: 7, port }]]);
  const tabs = new Map([[7, { muted: false }]]);
  assert.deepEqual(route({ id: "7:0", op: "mute" }, frames, tabs), { kind: "mute", tabId: 7, muted: true });
  const seek = route({ id: "7:0", op: "seek", value: 4000 }, frames, tabs);
  assert.equal(seek.kind, "frame");
  assert.equal(seek.frame.port, port);
  assert.deepEqual(seek.message, { op: "seek", value: 4000 });
  assert.equal(route({ id: "1:0", op: "toggle" }, frames, tabs).kind, "missing");
  assert.equal(route({ id: "7:0", op: "close" }, frames, tabs).kind, "invalid");
});

test("reconnect backoff is bounded", () => {
  assert.equal(backoff(0), 1000);
  assert.equal(backoff(3), 8000);
  assert.equal(backoff(50), 30000);
});
