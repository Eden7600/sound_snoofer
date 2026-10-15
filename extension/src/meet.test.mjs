import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import vm from "node:vm";

function adapter({ origin = "https://meet.google.com", top = true } = {}) {
  let now = 0, serial = 0, mutate;
  const timers = new Map(), ports = [], handlers = new Map();
  const button = (name) => ({ name, isConnected: true, disabled: false, value: "true", clicks: 0, visible: true,
    getAttribute(attr) { return attr === "data-is-muted" ? this.value : null; }, matches() { return false; },
    getClientRects() { return this.visible ? [{ width: 40, height: 40 }] : []; },
    click() { this.clicks++; if (!this.delayed) { h.records.push({ type: "attributes", target: this, attributeName: "data-is-muted", oldValue: this.value }); this.value = this.value === "true" ? "false" : "true"; } }, contains(node) { return this === node; } });
  const h = { mic: button("mic"), camera: button("camera"), leave: button("leave"), roots: [], duplicates: {}, ports, records: [] };
  const select = (selector) => selector.includes("CQylAd") ? (h.hasLeave ? [h.leave] : []) : selector.includes("hw0c9") ? [h.mic, ...(h.duplicates.mic || [])] : selector.includes("psRWwc") ? [h.camera] : [];
  h.newRoot = () => ({ isConnected: true, querySelectorAll: select, contains: (node) => [h.mic, h.camera, h.leave].includes(node) });
  h.root = h.newRoot(); h.roots = [h.root]; h.hasLeave = true;
  const event = () => { const listeners = []; return { addListener: (fn) => listeners.push(fn), emit: (...args) => listeners.forEach((fn) => fn(...args)) }; };
  const window = { addEventListener: (type, fn) => handlers.set(type, fn) }; window.top = top ? window : {};
  const context = vm.createContext({ location: { origin }, window, URL, TextEncoder,
    document: { querySelectorAll: (selector) => selector.includes('role="region"') ? h.roots : select(selector) },
    chrome: { runtime: { connect() {
      const p = { sent: [], onMessage: event(), onDisconnect: event(), postMessage(message) { this.sent.push(JSON.parse(JSON.stringify(message))); }, disconnect() { this.onDisconnect.emit(); } };
      ports.push(p); return p;
    } } },
    setTimeout(fn, delay) { const id = ++serial; timers.set(id, { fn, at: now + delay }); return id; }, clearTimeout(id) { timers.delete(id); },
    MutationObserver: class { constructor(fn) { mutate = fn; } observe() {} disconnect() {} takeRecords() { return h.records.splice(0); } } });
  for (const file of ["logic.js", "meet.js"]) vm.runInContext(readFileSync(new URL(file, import.meta.url), "utf8"), context);
  h.advance = (ms) => { const end = now + ms; for (;;) { const next = [...timers].filter(([, t]) => t.at <= end).sort((a, b) => a[1].at - b[1].at)[0]; if (!next) break; now = next[1].at; timers.delete(next[0]); next[1].fn(); } now = end; };
  h.port = () => ports.at(-1);
  h.config = (enabled = true, instance = "1") => h.port().onMessage.emit({ type: "meet-config", enabled, instance });
  h.observe = () => { h.port().onMessage.emit({ type: "meet-observe" }); return h.state(); };
  h.state = () => h.port().sent.findLast((message) => message.type === "meet-state")?.report;
  h.command = (control, value, overrides = {}) => {
    const state = h.state();
    const command = { type: "meet-command", id: String(++serial), tab: 7, instance: "1", call: state.call, control, revision: state[control].revision, value, ...overrides };
    h.port().onMessage.emit(command); return command;
  };
  h.results = () => h.port().sent.filter((message) => message.type === "meet-result");
  h.mutate = (records = h.records.splice(0)) => { mutate(records); h.advance(50); };
  h.install = () => vm.runInContext(readFileSync(new URL("meet.js", import.meta.url), "utf8"), context);
  h.pageEvent = (type, event = {}) => handlers.get(type)(event);
  return h;
}

test("Meet isolated adapter requires exact top origin and one installation", () => {
  assert.equal(adapter({ origin: "https://example.com" }).ports.length, 0);
  assert.equal(adapter({ top: false }).ports.length, 0);
  const h = adapter(); h.install(); assert.equal(h.ports.length, 1);
  assert.equal(h.port().sent.length, 0, "disabled until host negotiation");
  h.config(); const initial = h.state();
  assert.equal(initial.state, "joined"); assert.equal(initial.mic.value, true); assert.equal(initial.camera.value, false);
  const later = h.observe(); assert.notEqual(later.sample, initial.sample); assert.equal(later.mic.revision, initial.mic.revision);
});

test("prejoin, ambiguous and unsupported controls never become actionable", () => {
  const h = adapter(); h.roots = []; h.hasLeave = false; h.config();
  assert.equal(h.state().state, "prejoin"); assert.equal(h.state().call, ""); assert.equal(h.state().mic.available, false);
  h.roots = [h.root]; h.hasLeave = true; h.mic.value = "not a boolean";
  assert.equal(h.observe().mic.value, null); assert.equal(h.state().camera.available, true);
  h.mic.value = "true"; h.mic.disabled = true; assert.equal(h.observe().mic.available, false);
  h.mic.disabled = false; h.mic.visible = false; assert.equal(h.observe().mic.available, false);
  h.mic.visible = true; h.duplicates.mic = [h.mic]; assert.equal(h.observe().mic.available, false);
  h.roots.push(h.newRoot()); assert.equal(h.observe().state, "unknown"); assert.equal(h.state().camera.available, false);
});

test("explicit desired state clicks once and reports observation, including zero-click satisfaction", () => {
  const h = adapter(); h.config();
  h.command("mic", true); assert.equal(h.mic.clicks, 0); assert.equal(h.results().at(-1).status, "observed");
  const request = h.command("mic", false);
  assert.equal(h.mic.clicks, 1); assert.equal(h.state().mic.value, false);
  assert.equal(h.results().at(-1).status, "observed"); assert.equal(h.results().at(-1).revision, request.revision);
  assert.equal(h.port().sent.at(-2).type, "meet-state", "fresh state precedes result");
  h.command("camera", true); assert.equal(h.camera.clicks, 1); assert.equal(h.state().camera.value, true);
});

test("same-document call and control replacement reject captured work", () => {
  const h = adapter(); h.config(); const initial = h.state();
  h.root = h.newRoot(); h.roots = [h.root]; h.observe();
  h.command("mic", false, { call: initial.call, revision: initial.mic.revision }); assert.equal(h.results().at(-1).reason, "stale");
  const before = h.state(); h.mic = { ...h.mic }; h.observe();
  h.command("mic", false, { revision: before.mic.revision }); assert.equal(h.results().at(-1).reason, "stale"); assert.equal(h.mic.clicks, 0);
  const call = h.state().call; h.roots = []; h.hasLeave = false; h.observe(); h.roots = [h.root]; h.hasLeave = true;
  assert.notEqual(h.observe().call, call);
});

test("manual change away and back invalidates the captured revision", () => {
  const h = adapter(); h.config(); const revision = h.state().mic.revision;
  h.mutate([{ type: "attributes", target: h.mic, attributeName: "data-is-muted", oldValue: "true" }, { type: "attributes", target: h.mic, attributeName: "data-is-muted", oldValue: "false" }]);
  h.command("mic", false, { revision }); assert.equal(h.results().at(-1).reason, "stale"); assert.equal(h.mic.clicks, 0);
});

test("independent controls, duplicate IDs, busy rejection and bounded uncertain readback", () => {
  const h = adapter(); h.config(); h.mic.delayed = true; h.camera.delayed = true;
  const mic = h.command("mic", false); h.port().onMessage.emit(mic);
  assert.equal(h.results().length, 0); assert.equal(h.mic.clicks, 1);
  h.command("mic", false); assert.equal(h.results().at(-1).reason, "busy");
  h.command("camera", true); assert.equal(h.camera.clicks, 1);
  h.mic.value = "false"; h.mutate(); assert.equal(h.results().at(-1).status, "observed");
  h.advance(3000); assert.equal(h.results().at(-1).status, "unknown"); assert.equal(h.results().at(-1).reason, "not-confirmed");
  h.observe(); h.advance(10000); assert.equal(h.camera.clicks, 1);
});

test("disable and runtime-port loss cancel pending work without replay", () => {
  const h = adapter(); h.config(); h.mic.delayed = true; h.command("mic", false);
  h.config(false); assert.equal(h.results().at(-1).reason, "stopped");
  h.config(); assert.equal(h.mic.clicks, 1); h.command("mic", false);
  const old = h.port(); old.onDisconnect.emit(); h.advance(1000);
  assert.equal(h.ports.length, 2); assert.equal(h.port().sent.length, 0);
  old.onMessage.emit({ type: "meet-config", enabled: true, instance: "old" }); assert.equal(h.port().sent.length, 0);
  h.config(true, "2"); h.advance(5000); assert.equal(h.mic.clicks, 2); assert.equal(h.results().length, 0);
});

test("node removal and reinsertion cancels an in-flight observation", () => {
  const h = adapter(); h.config(); h.camera.delayed = true; h.command("camera", true);
  h.mutate([{ removedNodes: [h.camera] }]);
  assert.equal(h.results().at(-1).status, "unknown"); assert.equal(h.camera.clicks, 1);
});

test("Leave button replacement or removal ends the captured call lifetime", () => {
  const h = adapter(); h.config(); const initial = h.state();
  h.leave = { ...h.leave }; h.observe(); assert.notEqual(h.state().call, initial.call);
  h.command("mic", false, { call: initial.call }); assert.equal(h.results().at(-1).reason, "stale");
  h.camera.delayed = true; h.command("camera", true); const before = h.state().call;
  h.mutate([{ removedNodes: [h.leave] }]);
  assert.equal(h.results().at(-1).status, "unknown"); assert.notEqual(h.state().call, before);
  assert.equal(h.camera.clicks, 1); assert.equal(h.mic.clicks, 0);
});

test("back-forward restored document reconnects with no pending activation", () => {
  const h = adapter(); h.config(); h.mic.delayed = true; h.command("mic", false);
  h.pageEvent("pagehide"); h.advance(2000); assert.equal(h.ports.length, 1);
  h.pageEvent("pageshow", { persisted: true }); assert.equal(h.ports.length, 2); assert.equal(h.port().sent.length, 0);
  h.config(true, "2"); h.advance(4000); assert.equal(h.mic.clicks, 1); assert.equal(h.results().length, 0);
});

test("port loss while publishing the immediate pre-click read cannot still click", () => {
  const h = adapter(); h.config();
  h.port().postMessage = () => { throw new Error("port closed during read"); };
  h.command("mic", false);
  assert.equal(h.mic.clicks, 0);
  h.advance(1000); assert.equal(h.ports.length, 2); assert.equal(h.port().sent.length, 0);
});

test("a synchronously read click and unrelated styling do not advance revision twice", () => {
  const h = adapter(); h.config(); h.command("mic", false);
  const revision = h.state().mic.revision;
  h.mutate(); assert.equal(h.state().mic.revision, revision);
  h.mutate([{ type: "attributes", target: h.mic, attributeName: "class", oldValue: "normal" }]);
  assert.equal(h.state().mic.revision, revision);
});

test("control identity changes only with node lifetime, including same-batch reinsertion", () => {
  const h = adapter(); h.config(); const initial = h.state();
  assert.equal(initial.mic.identity, "1"); assert.equal(initial.camera.identity, "1");
  h.command("mic", false); assert.equal(h.state().mic.identity, initial.mic.identity);
  h.mic.disabled = true; h.observe(); h.mic.disabled = false; h.observe();
  assert.equal(h.state().mic.identity, initial.mic.identity);
  h.mutate([{ type: "attributes", target: h.mic, attributeName: "data-is-muted", oldValue: "true" }]);
  assert.equal(h.state().mic.identity, initial.mic.identity, "attribute revision must not manufacture a new node");
  h.mic = { ...h.mic, value: "true" }; const replacement = h.observe();
  assert.notEqual(replacement.mic.identity, initial.mic.identity);
  assert.equal(replacement.camera.identity, initial.camera.identity);
  h.mutate([{ removedNodes: [h.mic] }]); assert.notEqual(h.state().mic.identity, replacement.mic.identity);
  const reinserted = h.state().mic.identity;
  h.duplicates.mic = [h.mic]; h.observe(); h.duplicates.mic = []; h.observe();
  assert.notEqual(h.state().mic.identity, reinserted);
});
