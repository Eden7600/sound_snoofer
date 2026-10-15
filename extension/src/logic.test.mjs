import { test } from "node:test";
import assert from "node:assert/strict";
import "./logic.js";

const { PROTOCOL, DEFAULT_PORT, backoff, browserName, route, sessions, validPort } = globalThis.SnooferLogic;
const { meetCounter, meetState, meetCommand, meetResult } = globalThis.SnooferLogic;

test("protocol and port match Snoofer", () => {
  assert.equal(PROTOCOL, 2);
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
    ["7:0", { tabId: 7, instance: "f7", report: { present: true, title: "", artist: "Band", state: "playing", positionMs: 1000, durationMs: 9000, at: 5, canNext: true, artwork: "" } }],
    ["7:3", { tabId: 7, instance: "f7", report: { present: false } }],
    ["9:0", { tabId: 9, instance: "f9", report: null }],
    ["8:0", { tabId: 8, instance: "f8", report: { present: true, title: "Song", state: "paused", artwork: "https://cdn/cover.jpg" } }],
  ]);
  const tabs = new Map([
    [7, { title: "Tab title", url: "https://www.youtube.com/watch?v=x", muted: true, favIconUrl: "https://youtube.com/favicon.ico" }],
    [8, { title: "Other", url: "not a url", muted: false }],
  ]);
  const list = sessions(frames, tabs);
  assert.deepEqual(list.map((s) => s.id), ["7:0", "8:0"]);
  assert.equal(list[0].instance, "f7");
  assert.equal(list[0].title, "Tab title", "falls back to the tab title");
  assert.equal(list[0].site, "youtube.com");
  assert.equal(list[0].muted, true);
  assert.equal(list[0].artKey, "https://youtube.com/favicon.ico", "favicon when the page has no artwork");
  assert.equal(list[1].artKey, "https://cdn/cover.jpg");
  assert.equal(list[1].site, "");
  assert.equal(list[1].state, "paused");
});

test("sessions coerce page-supplied fields and keep only web artwork", () => {
  const frames = new Map([
    ["1:0", { tabId: 1, instance: "f1", report: { present: true, title: {}, artist: 5, positionMs: "x", rate: -2, artwork: "file:///C:/secret.png" } }],
    ["2:0", { tabId: 2, instance: "f2", report: { present: true, artwork: "javascript:alert(1)" } }],
  ]);
  const tabs = new Map([[1, { title: "Tab" }], [2, { favIconUrl: "https://site/favicon.ico" }]]);
  const [first, second] = sessions(frames, tabs);
  assert.equal(first.title, "Tab");
  assert.equal(first.artist, "");
  assert.equal(first.positionMs, 0);
  assert.equal(first.rate, 1);
  assert.equal(first.artKey, "");
  assert.equal(second.artKey, "https://site/favicon.ico");
});

test("commands route to frames or the tabs API", () => {
  const port = {};
  const frames = new Map([["7:0", { tabId: 7, instance: "f7", port }]]);
  const tabs = new Map([[7, { muted: false }]]);
  assert.deepEqual(route({ id: "7:0", instance: "f7", op: "mute", value: 1 }, frames, tabs), { kind: "mute", tabId: 7, muted: true });
  const seek = route({ id: "7:0", instance: "f7", op: "seek", value: 4000 }, frames, tabs);
  assert.equal(seek.kind, "frame");
  assert.equal(seek.frame.port, port);
  assert.deepEqual(seek.message, { op: "seek", value: 4000 });
  assert.equal(route({ id: "1:0", op: "toggle" }, frames, tabs).kind, "missing");
  assert.equal(route({ id: "7:0", instance: "f7", op: "close" }, frames, tabs).kind, "invalid");
});

test("reconnect backoff is bounded", () => {
  assert.equal(backoff(0), 1000);
  assert.equal(backoff(3), 8000);
  assert.equal(backoff(50), 30000);
});

test("a reused frame ID cannot accept the old document command", () => {
  const frames = new Map([["7:0", { tabId: 7, instance: "new", port: {} }]]);
  const tabs = new Map([[7, { muted: false }]]);
  for (const op of ["toggle", "mute", "next", "seek"]) {
    assert.equal(route({ id: "7:0", instance: "old", op, value: 1 }, frames, tabs).kind, "stale");
  }
});

test("mute carries the captured desired value even if tab state changes", () => {
  const frames = new Map([["7:0", { tabId: 7, instance: "current", port: {} }]]);
  const tabs = new Map([[7, { muted: true }]]);
  assert.deepEqual(route({ id: "7:0", instance: "current", op: "mute", value: 1 }, frames, tabs), { kind: "mute", tabId: 7, muted: true });
  assert.deepEqual(route({ id: "7:0", instance: "current", op: "mute", value: 0 }, frames, tabs), { kind: "mute", tabId: 7, muted: false });
  assert.equal(route({ id: "7:0", instance: "current", op: "toggle" }, frames, tabs).kind, "invalid");
});

test("ratings require verified sender identity, bounded fields and current track", () => {
  const frame = { tabId: 7, instance: "f7", youtubeMusic: true, report: { present: true, track: "1:song", rating: "like", canRate: true } };
  const frames = new Map([["7:0", frame]]);
  const command = { id: "7:0", instance: "f7", op: "rate", track: "1:song", rating: "none" };
  assert.deepEqual(route(command, frames).message, { op: "rate", track: "1:song", rating: "none" });
  assert.equal(route({ ...command, track: "old" }, frames).kind, "stale");
  for (const extra of [{ rating: "unknown" }, { rating: 1 }, { track: "x".repeat(129) }, { track: "" }, { track: "x\n" }]) {
    assert.equal(route({ ...command, ...extra }, frames).kind, "invalid");
  }
  for (const fields of [{ rating: "unknown" }, { canRate: false }, { track: "new" }, { track: "x".repeat(129) }, { rating: "LIKE" }, { canRate: "true" }, { present: false }, { present: "true" }]) {
    const altered = new Map([["7:0", { ...frame, report: { ...frame.report, ...fields } }]]);
    assert.equal(route(command, altered).kind, "stale");
  }
  frame.youtubeMusic = false;
  assert.equal(sessions(frames, new Map())[0].rating, undefined);
  assert.equal(route(command, frames).kind, "stale");
});

test("unknown differs from old companion and reload blocks only page commands", () => {
  const frame = { tabId: 7, instance: "f7", youtubeMusic: true, report: { present: true } };
  const frames = new Map([["7:0", frame]]);
  assert.equal(sessions(frames, new Map())[0].rating, undefined);
  frame.report.rating = "unknown"; frame.report.canRate = false;
  assert.equal(sessions(frames, new Map())[0].rating, "unknown");
  Object.assign(frame.report, { reloadRequired: true, canNext: true, canPrev: true, canSeek: true, rating: "like", track: "1:song", canRate: true });
  const state = sessions(frames, new Map())[0];
  assert.equal(state.reloadRequired, true); assert.equal(state.canRate, false);
  for (const flag of ["canNext", "canPrev", "canSeek"]) assert.equal(state[flag], false);
  for (const op of ["toggle", "next", "prev", "seek", "rate"]) assert.equal(route({ id: "7:0", instance: "f7", op, value: 1, track: "1:song", rating: "none" }, frames).kind, "stale");
  assert.equal(route({ id: "7:0", instance: "f7", op: "mute", value: 1 }, frames).kind, "mute");
});

test("Meet wire counters, booleans, reasons and exact result echo are bounded", () => {
  assert.equal(meetCounter("18446744073709551615"), true);
  for (const value of ["0", "01", "-1", "1.0", "18446744073709551616", 1]) assert.equal(meetCounter(value), false);
  const c = { value: true, revision: "1", available: true, reason: "" };
  const state = { sample: "1", call: "1", state: "joined", mic: c, camera: { ...c, value: false } };
  assert.deepEqual(meetState(state), state);
  for (const mic of [{ ...c, value: undefined }, { ...c, value: null }, { ...c, reason: "permission denied" }, { ...c, available: "true" }]) assert.equal(meetState({ ...state, mic }), null);
  const command = { id: "1", tab: 0, instance: "1", call: "1", control: "mic", revision: "1", value: false };
  assert.deepEqual(meetCommand(command), command);
  assert.equal(meetCommand({ ...command, value: 0 }), null);
  const result = { ...command, status: "observed", reason: "" };
  assert.deepEqual(meetResult(result, command), result);
  assert.equal(meetResult({ ...result, revision: "2" }, command), null);
  assert.equal(meetResult({ ...result, status: "unknown", reason: "" }, command), null);
});
