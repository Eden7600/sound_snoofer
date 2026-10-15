import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import vm from "node:vm";

const event = () => {
  const listeners = [];
  return { addListener: (f) => listeners.push(f), emit: (...args) => listeners.forEach((f) => f(...args)) };
};

async function background() {
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
  await Promise.resolve();
  const port = () => ({ name: "media", sender: { tab: { id: 7 }, frameId: 0 }, onMessage: event(), onDisconnect: event(), sent: [], postMessage(message) { this.sent.push(message); } });
  const report = (p, title) => p.onMessage.emit({ report: JSON.stringify({ present: true, title, state: "paused" }) });
  const flush = async () => { while (timers.length) timers.shift()(); await Promise.resolve(); };
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
