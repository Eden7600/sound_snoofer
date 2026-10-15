// Isolated-world Meet adapter. DOM observations and the current runtime port are
// its only inputs; no page-world events, private APIs, labels or meeting content.
(() => {
  if (location.origin !== "https://meet.google.com" || window.top !== window || globalThis.__snooferMeet === 1) return;
  globalThis.__snooferMeet = 1;
  const { meetCounter, meetBounded, meetCommand, backoff } = globalThis.SnooferLogic;
  const api = globalThis.browser ?? chrome;
  const selector = { mic: 'button[jsname="hw0c9"]', camera: 'button[jsname="psRWwc"]' };
  let port = null, enabled = false, instance = "", stopped = false, retry = 0, attempts = 0, changed = 0;
  let root = null, leave = null, call = 0n, sample = 0n;
  const controls = { mic: null, camera: null };
  const revisions = { mic: 0n, camera: 0n };
  const identities = { mic: 0n, camera: 0n };
  const identityNodes = { mic: null, camera: null };
  const pending = new Map();
  const limit = 18446744073709551615n;

  function post(message) {
    try { port?.postMessage(message); } catch { disconnect(); }
  }
  function finish(entry, status, reason) {
    if (pending.get(entry.command.control) !== entry) return;
    pending.delete(entry.command.control);
    clearTimeout(entry.timer);
    post({ type: "meet-result", ...entry.command, status, reason });
  }
  function cancel(reason) {
    for (const entry of [...pending.values()]) finish(entry, "unknown", reason);
    clearTimeout(changed); changed = 0;
  }
  function disconnect() {
    const old = port;
    port = null; enabled = false; instance = "";
    cancel("disconnected");
    try { old?.disconnect(); } catch { /* An invalidated extension context is already closed. */ }
    if (!stopped && !retry) retry = setTimeout(connect, backoff(attempts++));
  }
  function next(value) {
    if (value >= limit) { stopped = true; disconnect(); throw new Error("Meet counter exhausted"); }
    return value + 1n;
  }
  function button(scope, name) {
    const found = scope.querySelectorAll(selector[name]);
    return found.length === 1 && found[0].isConnected ? found[0] : null;
  }
  function inspect(node, name, joined) {
    const muted = node?.getAttribute("data-is-muted");
    const known = muted === "true" || muted === "false";
    const value = known ? (name === "mic" ? muted === "true" : muted === "false") : null;
    const visible = node && [...node.getClientRects()].some((r) => r.width > 0 && r.height > 0);
    const available = joined && known && node.disabled === false && !node.matches(":disabled") && node.getAttribute("aria-disabled") !== "true" && !!visible;
    return { node, value, available: !!available, reason: available ? "" : !joined ? "not-in-call" : known ? "unavailable" : "unknown" };
  }
  function observe() {
    if (!enabled || !port) return null;
    invalidateMutations(observer.takeRecords());
    if (!enabled || !port) return null;
    const roots = document.querySelectorAll('[role="region"][jsname="vNB5le"]');
    const candidate = roots.length === 1 && roots[0].isConnected ? roots[0] : null;
    const leaves = candidate?.querySelectorAll('button[jsname="CQylAd"]') || [];
    const joined = !!candidate && leaves.length === 1 && leaves[0].isConnected;
    const currentRoot = joined ? candidate : null;
    const currentLeave = joined ? leaves[0] : null;
    if (root !== currentRoot || leave !== currentLeave) {
      root = currentRoot; leave = currentLeave;
      if (root) call = next(call);
    }
    const nodes = { mic: button(root || document, "mic"), camera: button(root || document, "camera") };
    const prejoin = !joined && roots.length <= 1 && leaves.length === 0 && nodes.mic && nodes.camera;
    const state = joined ? "joined" : prejoin ? "prejoin" : "unknown";
    const result = { state, call: joined ? String(call) : "", sample: String(sample = next(sample)) };
    for (const name of ["mic", "camera"]) {
      const current = inspect(nodes[name], name, joined);
      if (state === "unknown") { current.value = null; current.available = false; current.reason = "unknown"; }
      const old = controls[name];
      if (identities[name] === 0n || identityNodes[name] !== current.node) {
        identities[name] = next(identities[name]);
        identityNodes[name] = current.node;
      }
      if (!old || ["node", "value", "available", "reason"].some((key) => old[key] !== current[key])) revisions[name] = next(revisions[name]);
      controls[name] = current;
      result[name] = { value: current.value, available: current.available, reason: current.reason, revision: String(revisions[name]), identity: String(identities[name]) };
    }
    post({ type: "meet-state", report: result });
    for (const entry of [...pending.values()]) {
      const current = controls[entry.command.control];
      if (root !== entry.root || current.node !== entry.node || result.call !== entry.command.call) finish(entry, "unknown", "not-confirmed");
      else if (current.available && current.value === entry.command.value) finish(entry, "observed", "");
    }
    return result;
  }
  function command(message) {
    if (!meetBounded(message, 2048)) return;
    const request = meetCommand(message);
    if (!request) return;
    const reply = (reason) => post({ type: "meet-result", ...request, status: "rejected", reason });
    if ([...pending.values()].some((p) => p.command.id === request.id)) return;
    if (!enabled) return reply("disabled");
    if (request.instance !== instance) return reply("stale");
    if (pending.has(request.control)) return reply("busy");
    const capturedPort = port;
    const state = observe();
    if (!enabled || port !== capturedPort || request.instance !== instance) return reply("disabled");
    if (!state || state.call !== request.call || state[request.control].revision !== request.revision) return reply("stale");
    const current = controls[request.control];
    if (!current.available) return reply("unavailable");
    if (current.value === request.value) return post({ type: "meet-result", ...request, status: "observed", reason: "" });
    const entry = { command: request, node: current.node, root, leave };
    pending.set(request.control, entry);
    entry.timer = setTimeout(() => { observe(); finish(entry, "unknown", "not-confirmed"); }, 3000);
    try { current.node.click(); } catch { finish(entry, "unknown", "not-confirmed"); }
    observe();
  }
  function connect() {
    retry = 0;
    if (stopped || port) return;
    try {
      const current = api.runtime.connect({ name: "meet" });
      port = current;
      current.onMessage.addListener((message) => {
        if (port !== current || !message || typeof message !== "object") return;
        if (message.type === "meet-config" && typeof message.enabled === "boolean" && meetCounter(message.instance)) {
          cancel("stopped");
          enabled = message.enabled; instance = message.instance; attempts = 0;
          if (enabled) { post({ type: "meet-ready", instance }); observe(); }
        } else if (message.type === "meet-observe") observe();
        else if (message.type === "meet-command") command(message);
      });
      current.onDisconnect.addListener(() => { if (port === current) disconnect(); });
    } catch { disconnect(); }
  }
  function invalidateMutations(records) {
    // A removal/reinsertion in one batch must still retire captured node identity.
    for (const record of records) for (const removed of record.removedNodes || []) {
      for (const entry of [...pending.values()]) if ([entry.root, entry.leave, entry.node].some((node) => removed === node || removed.contains(node))) finish(entry, "unknown", "not-confirmed");
      if (root && (removed === root || removed.contains(root))) root = null;
      if (leave && (removed === leave || removed.contains(leave))) leave = null;
      for (const name of ["mic", "camera"]) if (identityNodes[name] && (removed === identityNodes[name] || removed.contains(identityNodes[name]))) identityNodes[name] = undefined;
      for (const name of ["mic", "camera"]) if (controls[name]?.node && (removed === controls[name].node || removed.contains(controls[name].node))) controls[name].node = null;
    }
    // Attribute records retain evidence of a change away and back between reads.
    for (const record of records) if (record.type === "attributes" && ["data-is-muted", "disabled", "aria-disabled", "hidden"].includes(record.attributeName) && record.oldValue !== record.target.getAttribute(record.attributeName)) {
      for (const name of ["mic", "camera"]) if (controls[name]?.node === record.target) controls[name] = null;
    }
  }
  const observer = new MutationObserver((records) => {
    invalidateMutations(records);
    if (enabled && !changed) changed = setTimeout(() => { changed = 0; observe(); }, 50);
  });
  const observerOptions = { childList: true, subtree: true, attributes: true, attributeOldValue: true, attributeFilter: ["data-is-muted", "disabled", "aria-disabled", "hidden", "style", "class", "jsname", "role"] };
  observer.observe(document, observerOptions);
  window.addEventListener("pagehide", () => { stopped = true; clearTimeout(retry); observer.disconnect(); disconnect(); });
  window.addEventListener("pageshow", (event) => {
    if (!event.persisted) return;
    stopped = false; root = null; leave = null;
    observer.observe(document, observerOptions);
    connect(); // A restored document gets a fresh port, never an old activation.
  });
  connect();
})();
