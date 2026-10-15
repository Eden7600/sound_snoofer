import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import vm from "node:vm";

test("updated bridge uses only versioned page events in already-open tabs", () => {
  const listeners = new Map(), reports = [], commands = [];
  let receive;
  const context = vm.createContext({ __snooferBridge: 2,
    chrome: { runtime: { connect: () => ({ onMessage: { addListener: (f) => { receive = f; } },
      onDisconnect: { addListener() {} }, postMessage: (m) => reports.push(m) }) } },
    window: { addEventListener: (n, f) => listeners.set(n, f), dispatchEvent: (e) => commands.push(e) },
    CustomEvent: class { constructor(type, options) { this.type = type; this.detail = options.detail; } } });
  vm.runInContext(readFileSync(new URL("bridge.js", import.meta.url), "utf8"), context);
  assert.equal(listeners.has("snoofer-media:report"), false);
  assert.equal(listeners.has("snoofer-media:report-v2"), false);
  listeners.get("snoofer-media:report-v3")({ detail: JSON.stringify({ present: true }) });
  assert.equal(reports.length, 1);
  receive({ op: "toggle", value: 0 });
  assert.equal(commands[0].type, "snoofer-media:command-v3");
  assert.equal(JSON.parse(commands[0].detail).value, 0);
});

function replacedPort() {
  const listeners = new Map(), commands = [], ports = [];
  const context = vm.createContext({
    chrome: { runtime: { connect: () => {
      const port = { sent: [], fail: false,
        onMessage: { addListener(f) { port.receive = f; } },
        onDisconnect: { addListener(f) { port.disconnect = f; } },
        postMessage(message) { if (port.fail) throw new Error("Port disconnected"); port.sent.push(message); } };
      ports.push(port); return port;
    } } },
    window: { addEventListener: (n,f) => listeners.set(n,f), dispatchEvent: (e) => commands.push(e) },
    CustomEvent: class { constructor(type,options) { this.type=type; this.detail=options.detail; } } });
  vm.runInContext(readFileSync(new URL("bridge.js",import.meta.url),"utf8"),context);
  const report=() => listeners.get("snoofer-media:report-v3")({ detail: JSON.stringify({present:true}) });
  report();
  const old=ports[0]; old.fail=true;
  report(); // A disconnected send fails before the queued disconnect callback runs.
  report(); // The next report opens a replacement in this content world.
  assert.equal(ports.length,2);
  return {old,current:ports[1],ports,report,commands};
}

test("late content-port commands do not cross a replacement",()=>{
  const h=replacedPort();
  h.old.receive({op:"toggle",value:1});
  assert.equal(h.commands.length,0,"old port forwarded a command after replacement");
  h.current.receive({op:"toggle",value:0});
  assert.equal(h.commands.length,1);
  assert.equal(JSON.parse(h.commands[0].detail).value,0);
});

test("late content-port disconnect cannot clear its replacement",()=>{
  const h=replacedPort();
  h.old.disconnect();
  h.report();
  assert.equal(h.ports.length,2,"old disconnect caused an unnecessary new port lifetime");
  assert.equal(h.current.sent.length,2);
});
