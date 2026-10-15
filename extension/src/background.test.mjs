import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import vm from "node:vm";

const event = () => {
  const listeners = [];
  return { addListener: (f) => listeners.push(f), emit: (...args) => listeners.forEach((f) => f(...args)) };
};

async function background(download) {
  const sockets = [], timers = [];
  const api = {
    runtime: { getManifest: () => ({ version: "test" }), onConnect: event(), onMessage: event(), onInstalled: event() },
    tabs: { onUpdated: event(), onRemoved: event(), query: async () => [], update: async () => {} },
    storage: { onChanged: event(), local: { get: async () => ({}) } },
  };
  class Socket {
    sent = [];
    constructor() { sockets.push(this); }
    send(message) { this.sent.push(JSON.parse(message)); }
    close() { this.onclose?.(); }
  }
  const context = vm.createContext({ chrome: api, navigator: { userAgent: "Chrome" }, WebSocket: Socket, URL,
    setTimeout: (f) => timers.push(f), clearTimeout: () => {}, setInterval: () => {} });
  for (const file of ["logic.js", "background.js"]) vm.runInContext(readFileSync(new URL(file, import.meta.url), "utf8"), context);
  if (download) { context.download = download; vm.runInContext("thumbnail = download", context); }
  await Promise.resolve();
  const port = (sender = {}) => ({ name: "media", sender: { tab: { id: 7 }, frameId: 0, ...sender }, onMessage: event(), onDisconnect: event(), sent: [], postMessage(message) { this.sent.push(message); } });
  const report = (p, title, fields = {}) => p.onMessage.emit({ report: JSON.stringify({ present: true, title, state: "paused", ...fields }) });
  const flush = async () => { await new Promise(setImmediate); while (timers.length) timers.shift()(); await new Promise(setImmediate); };
  return { api, sockets, port, report, flush };
}

test("replacement port owns its report and rejects commands for the old document", async () => {
  const h = await background();
  const ws = h.sockets[0];
  ws.onopen();
  const old = h.port();
  h.api.runtime.onConnect.emit(old);
  h.report(old, "Old");
  await h.flush();
  const oldSession = ws.sent.at(-1).sessions[0];
  const current = h.port();
  h.api.runtime.onConnect.emit(current);
  h.report(current, "Current");
  h.report(old, "Late old report");
  old.onDisconnect.emit();
  await h.flush();
  const session = ws.sent.at(-1).sessions[0];
  assert.equal(session.title, "Current");
  assert.notEqual(session.instance, oldSession.instance);
  ws.onmessage({ data: JSON.stringify({ type: "command", id: session.id, instance: oldSession.instance, op: "toggle", value: 1 }) });
  assert.equal(current.sent.length, 0);
  ws.onmessage({ data: JSON.stringify({ type: "command", id: session.id, instance: session.instance, op: "toggle", value: 0 }) });
  assert.equal(current.sent.length, 1);
  assert.equal(current.sent[0].value, 0);
});

test("obsolete socket callbacks cannot command or replace the current connection", async () => {
  const h = await background();
  const old = h.sockets[0];
  old.onopen();
  const p = h.port();
  h.api.runtime.onConnect.emit(p);
  h.report(p, "Same document");
  await h.flush();
  const session = old.sent.at(-1).sessions[0];
  h.api.storage.onChanged.emit({ port: { newValue: 50000 } }, "local");
  const current = h.sockets[1];
  current.onopen();
  const command = { data: JSON.stringify({ type: "command", id: session.id, instance: session.instance, op: "next", value: 0 }) };
  old.onopen();
  old.onmessage(command);
  old.onclose();
  assert.equal(p.sent.length, 0);
  current.onmessage(command);
  assert.equal(p.sent.length, 1);
});

test("ratings trust runtime sender URL and top frame, never tab URL or page claims", async () => {
  const h = await background(); const ws = h.sockets[0]; ws.onopen();
  for (const [sender, allowed] of [
    [{ url: "https://music.youtube.com/watch" }, true],
    [{ url: "https://music.youtube.com/watch", frameId: 3 }, false],
    [{ url: "https://evil.example/", tab: { id: 7, url: "https://music.youtube.com" } }, false],
    [{ tab: { id: 7, url: "https://music.youtube.com" } }, false],
    [{ url: "https://music.youtube.com/watch", origin: "null" }, false],
  ]) {
    const p = h.port(sender); h.api.runtime.onConnect.emit(p);
    h.report(p, "Song", { origin: "https://music.youtube.com", rating: "none", track: "1:song", canRate: true }); await h.flush();
    const session = ws.sent.at(-1).sessions.find((s) => s.id === "7:" + (sender.frameId || 0));
    assert.equal(session.canRate === true, allowed);
    ws.onmessage({ data: JSON.stringify({ type: "command", id: session.id, instance: session.instance, op: "rate", track: "1:song", rating: "like" }) });
    assert.equal(p.sent.length, allowed ? 1 : 0);
    p.onDisconnect.emit(); await h.flush();
  }
});

test("slow artwork sends current paused rating immediately and cannot roll it back", async () => {
  let finish; let downloads = 0;
  const h = await background(() => { downloads++; return new Promise((resolve) => { finish = resolve; }); });
  const ws = h.sockets[0]; ws.onopen();
  const p = h.port({ url: "https://music.youtube.com" }); h.api.runtime.onConnect.emit(p);
  const fields = { artwork: "https://cdn/art", track: "1:song", rating: "none", canRate: true };
  h.report(p, "Song", fields); await h.flush();
  assert.equal(ws.sent.at(-1).sessions[0].rating, "none");
  h.report(p, "Song", { ...fields, rating: "like" }); await h.flush();
  assert.equal(ws.sent.at(-1).sessions[0].rating, "like"); assert.equal(downloads, 1);
  finish("png"); await h.flush();
  const current = ws.sent.at(-1).sessions[0]; assert.equal(current.rating, "like"); assert.equal(current.art, "png");
  h.report(p, "Song", { ...fields, rating: "dislike" }); await h.flush();
  assert.equal(ws.sent.at(-1).sessions[0].art, undefined); assert.equal(downloads, 1);
});

test("obsolete artwork cannot revive a replaced frame or cross a socket reconnect", async () => {
  const finishes = [];
  const h = await background(() => new Promise((resolve) => finishes.push(resolve)));
  const ws = h.sockets[0]; ws.onopen();
  const old = h.port(); h.api.runtime.onConnect.emit(old); h.report(old, "Old", { artwork: "https://cdn/old" }); await h.flush();
  const current = h.port(); h.api.runtime.onConnect.emit(current); h.report(current, "Current", { artwork: "https://cdn/new" }); await h.flush();
  finishes[0]("old-png"); await h.flush();
  assert.equal(ws.sent.at(-1).sessions[0].title, "Current"); assert.equal(ws.sent.at(-1).sessions[0].art, undefined);
  h.api.storage.onChanged.emit({ port: { newValue: 50000 } }, "local"); const replacement = h.sockets[1]; replacement.onopen();
  finishes[1]("old-connection-png"); await h.flush();
  assert.equal(replacement.sent.at(-1).sessions[0].art, undefined);
  finishes[2]("current-png"); await h.flush();
  assert.equal(replacement.sent.at(-1).sessions[0].art, "current-png");
});
