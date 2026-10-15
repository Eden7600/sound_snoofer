import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import vm from "node:vm";

const event = () => { const listeners = []; return { addListener: (f) => listeners.push(f), emit: (...args) => listeners.forEach((f) => f(...args)) }; };
const control = (value, revision = "1") => ({ value, revision, available: true, reason: "" });
const state = (sample = "1") => ({ state: "joined", call: "1", sample, mic: control(true), camera: control(false) });
async function background(download = async () => "png") {
  let now = 0, serial = 0;
  const timers = new Map(), sockets = [];
  const api = { runtime: { getManifest: () => ({ version: "test" }), onConnect: event(), onMessage: event(), onInstalled: event() },
    tabs: { onUpdated: event(), onRemoved: event(), query: async () => [], update: async () => {} },
    storage: { onChanged: event(), local: { get: async () => ({}) } } };
  class Socket {
    sent = []; constructor() { sockets.push(this); }
    send(raw) { this.sent.push(JSON.parse(raw)); } close() { this.onclose?.(); }
  }
  const timeout = (fn, delay, repeat = false) => { const id = ++serial; timers.set(id, { fn, at: now + delay, repeat: repeat ? delay : 0 }); return id; };
  const context = vm.createContext({ chrome: api, navigator: { userAgent: "Chrome" }, URL, TextEncoder, WebSocket: Socket, performance: { now: () => now }, download,
    setTimeout: timeout, clearTimeout: (id) => timers.delete(id), setInterval: (fn, delay) => timeout(fn, delay, true) });
  for (const file of ["logic.js", "background.js"]) vm.runInContext(readFileSync(new URL(file, import.meta.url), "utf8"), context);
  vm.runInContext("thumbnail = download", context);
  await Promise.resolve();
  const h = { api, sockets, context };
  h.advance = (ms) => { const end = now + ms; for (;;) { const next = [...timers].filter(([, t]) => t.at <= end).sort((a, b) => a[1].at - b[1].at)[0]; if (!next) break; now = next[1].at; timers.delete(next[0]); if (next[1].repeat) timers.set(next[0], { ...next[1], at: now + next[1].repeat }); next[1].fn(); } now = end; };
  h.port = (tab = 7, sender = {}, name = "meet") => {
    const p = { name, sender: { tab: { id: tab }, frameId: 0, url: "https://meet.google.com/fixture", documentId: "doc-" + tab, ...sender }, sent: [], onMessage: event(), onDisconnect: event(),
      postMessage(m) { if (this.fail) throw new Error("lost port"); this.sent.push(JSON.parse(JSON.stringify(m))); }, disconnect() { this.closed = true; this.onDisconnect.emit(); } };
    api.runtime.onConnect.emit(p); return p;
  };
  h.message = (message, ws = sockets.at(-1)) => ws.onmessage({ data: JSON.stringify(message) });
  h.caps = (meetEnabled = true, mediaEnabled = true) => h.message({ type: "capabilities", meet: 1, meetEnabled, mediaEnabled });
  h.ready = (p) => { const instance = p.sent.findLast((m) => m.type === "meet-config").instance; p.onMessage.emit({ type: "meet-ready", instance }); return instance; };
  h.report = (p, report = state()) => p.onMessage.emit({ type: "meet-state", report });
  h.calls = () => sockets.at(-1).sent.findLast((m) => m.type === "meet-state")?.calls;
  h.command = (p, id = "1", control = "mic", value = false) => ({ type: "meet-command", id, tab: p.sender.tab.id, instance: p.sent.findLast((m) => m.type === "meet-config").instance, call: "1", control, revision: "1", value });
  h.results = () => sockets.at(-1).sent.filter((m) => m.type === "meet-result");
  sockets[0].onopen(); return h;
}

test("Meet requires host negotiation and admitted top-frame sender", async () => {
  const h = await background(); const p = h.port();
  assert.equal(h.sockets[0].sent[0].meet, 1); assert.equal(p.sent[0].enabled, false);
  h.ready(p); h.report(p); h.advance(1500); assert.equal(h.calls(), undefined);
  h.message(h.command(p)); assert.equal(h.results().length, 0, "old host receives no Meet envelopes");
  h.message({ type: "capabilities", meet: 99, mediaEnabled: true, meetEnabled: true });
  h.message(h.command(p)); assert.equal(h.results().length, 0, "unknown Meet version cannot command");
  for (const sender of [{ frameId: 1 }, { url: "https://evil.example", tab: { id: 9, url: "https://meet.google.com" } }, { origin: "null" }, { tab: { id: -1 } }]) assert.equal(h.port(9, sender).closed, true);
  h.caps(); h.report(p); h.advance(200); assert.deepEqual(h.calls(), [], "requires fresh enable acknowledgement");
  h.ready(p); h.report(p); h.advance(200); assert.equal(h.calls()[0].mic.value, true);
});

test("samples expire independently; cached/lower samples never renew freshness", async () => {
  const h = await background(); h.caps(); const a = h.port(7), b = h.port(8); h.ready(a); h.ready(b); h.report(a); h.report(b); h.advance(200);
  for (let n = 2; n <= 5; n++) { h.advance(800); h.report(a, state("1")); h.report(b, state(String(n))); h.advance(200); }
  assert.deepEqual(h.calls().map((call) => call.tab), [8]);
  h.report(a, state("2")); h.advance(200); assert.equal(h.calls().length, 2);
  h.report(a, state("1")); h.advance(3000); assert.equal(h.calls().some((call) => call.tab === 7), false);
});

test("results correlate exact socket, port and intent, settle once, and ignore duplicate pending IDs", async () => {
  const h = await background(); h.caps(); const p = h.port(); h.ready(p); h.report(p);
  const c = h.command(p); h.message(c); h.message(c);
  assert.equal(p.sent.filter((m) => m.type === "meet-command").length, 1);
  h.message(h.command(p, "2")); assert.equal(h.results().at(-1).reason, "busy");
  h.message(h.command(p, "3", "camera", true)); assert.equal(p.sent.filter((m) => m.type === "meet-command").length, 2);
  const result = { ...c, type: "meet-result", status: "observed", reason: "" };
  p.onMessage.emit({ ...result, value: true }); assert.equal(h.results().filter((m) => m.id === c.id).length, 0);
  p.onMessage.emit(result); p.onMessage.emit(result); assert.equal(h.results().filter((m) => m.id === c.id).length, 1);
  h.advance(5000); assert.equal(h.results().find((m) => m.id === "3").status, "unknown");
});

test("disable and replacement socket cancel adapters and reject old results", async () => {
  const h = await background(); h.caps(); const p = h.port(); h.ready(p); h.report(p);
  const first = h.command(p); h.message(first); h.caps(false);
  assert.equal(p.sent.at(-1).enabled, false); assert.equal(h.results().at(-1).reason, "stopped");
  h.caps(); h.message(h.command(p, "2")); assert.equal(h.results().at(-1).reason, "stale");
  h.ready(p); h.report(p, state("2")); const next = h.command(p, "3"); h.message(next);
  const old = h.sockets[0]; h.api.storage.onChanged.emit({ port: { newValue: 50000 } }, "local");
  assert.equal(p.sent.at(-1).enabled, false);
  const current = h.sockets.at(-1); current.onopen(); h.caps(); h.ready(p); h.report(p, state("3"));
  p.onMessage.emit({ ...next, type: "meet-result", status: "observed", reason: "" });
  h.message(next, old); assert.equal(h.results().length, 0);
  assert.equal(p.sent.filter((m) => m.type === "meet-command").length, 2, "reconnect sends no command");
});

test("replaced or mutated document ports cannot publish or settle current work", async () => {
  const h = await background(); h.caps(); const old = h.port(); h.ready(old); h.report(old); const c = h.command(old); h.message(c);
  const p = h.port(); h.ready(p); h.report(p); h.advance(200);
  h.report(old, { ...state("99"), call: "99" }); old.onDisconnect.emit(); h.advance(200);
  assert.equal(h.calls()[0].call, "1"); assert.notEqual(h.calls()[0].instance, c.instance);
  p.sender.documentId = "replaced"; h.report(p, state("2")); h.advance(3000);
  assert.deepEqual(h.calls(), []);
});

test("Meet outstanding requests are bounded to eight and malformed reports are ignored", async () => {
  const h = await background(); h.caps(); const ports = [];
  for (let n = 0; n < 5; n++) { const p = h.port(n); h.ready(p); h.report(p); ports.push(p); }
  for (let n = 0; n < 8; n++) h.message(h.command(ports[Math.floor(n / 2)], String(n + 1), n % 2 ? "camera" : "mic", n % 2 === 1));
  h.message(h.command(ports[4], "9")); assert.equal(h.results().at(-1).reason, "busy");
  h.report(ports[4], { ...state("2"), sample: "18446744073709551616" });
  h.report(ports[4], { ...state("2"), privateText: "x".repeat(8192) });
  h.advance(200); assert.equal(h.calls().find((c) => c.tab === 4).sample, "1");
});

test("Media enable transitions refresh artwork without disconnecting Meet", async () => {
  let downloads = 0;
  const h = await background(async () => { downloads++; return "png"; });
  h.caps(true, false);
  const meet = h.port(); h.ready(meet); h.report(meet);
  const media = h.port(8, { url: "https://music.youtube.com" }, "media");
  media.onMessage.emit({ report: JSON.stringify({ present: true, title: "Song", artwork: "https://cdn/art" }) }); h.advance(200);
  assert.equal(downloads, 0);
  h.caps(true, true); h.advance(200); await new Promise(setImmediate); h.advance(200); assert.equal(downloads, 1);
  assert.equal(h.sockets[0].sent.findLast((m) => m.type === "sessions").sessions[0].art, "png");
  h.caps(true, true); h.advance(200); assert.equal(downloads, 1);
  h.caps(true, false); h.caps(true, true); h.advance(200); await new Promise(setImmediate); h.advance(200); assert.equal(downloads, 2);
  assert.equal(h.sockets.length, 1); assert.equal(meet.sent.filter((m) => m.type === "meet-config").length, 1);
});

test("exact Meet-origin Media ports are refused while other sites retain Media", async () => {
  const h = await background();
  for (const sender of [{ url: "https://meet.google.com/call" }, { url: "about:blank", origin: "https://meet.google.com" }]) {
    const p = h.port(7, sender, "media"); assert.equal(p.closed, true);
    p.onMessage.emit({ report: JSON.stringify({ present: true, title: "Must not leave Meet" }) });
  }
  const other = h.port(8, { url: "https://music.youtube.com" }, "media");
  other.onMessage.emit({ report: JSON.stringify({ present: true, title: "Music" }) }); h.advance(200);
  assert.deepEqual(h.sockets[0].sent.findLast((m) => m.type === "sessions").sessions.map((s) => s.title), ["Music"]);
});

test("install injection uses only isolated Meet files on Meet top frame", async () => {
  const h = await background(); const injected = [];
  h.api.tabs.query = async () => [{ id: 7, url: "https://meet.google.com/call" }, { id: 8, url: "https://music.youtube.com/" }];
  h.api.scripting = { executeScript: async (request) => injected.push(JSON.parse(JSON.stringify(request))) };
  await vm.runInContext("injectOpenTabs()", h.context);
  assert.deepEqual(injected.filter((i) => i.target.tabId === 7), [{ target: { tabId: 7, frameIds: [0] }, files: ["logic.js", "meet.js"] }]);
  assert.deepEqual(injected.filter((i) => i.target.tabId === 8).map((i) => i.files[0]), ["page.js", "bridge.js"]);
});
