// Background script (Chrome service worker, Firefox event page): keeps one
// WebSocket to Snoofer on this computer, merges frame reports into sessions,
// sends artwork as 128 px PNGs once per change, carries out Snoofer's
// commands and answers the popup.
if (typeof importScripts === "function" && !globalThis.SnooferLogic) importScripts("logic.js");
const { PROTOCOL, DEFAULT_PORT, backoff, browserName, route, sessions, validPort, meetBounded, meetOrigin, meetSender, meetState, meetCommand, meetResult } = globalThis.SnooferLogic;
const api = globalThis.browser ?? chrome;

const VERSION = api.runtime.getManifest().version;
const BROWSER = browserName(navigator.userAgentData && navigator.userAgentData.brands, !!navigator.brave, navigator.userAgent);
const frames = new Map(); // "tab:frame" to {tabId, instance, port, report}.
let nextFrame = 0n;
const tabs = new Map(); // Tab ID to {title, url, muted, favIconUrl}.
const sentArt = new Map(); // Session ID to current instance/key download and send state.
let port = DEFAULT_PORT;
let socket = null;
let connected = false;
let refused = ""; // Snoofer's reason for refusing this extension, if any.
let attempts = 0;
let retry = 0;
let sendTimer = 0;
let capabilitiesSeen = false, mediaEnabled = true, meetNegotiated = false, meetEnabled = false, meetTimer = 0;
let nextMeet = 0n;
const meetFrames = new Map(); // Verified top-frame ports; no page titles or call URLs.
const meetPending = new Map(); // Request ID to exact socket/port/intent and bounded timer.

function meetReply(ws, command, status, reason) {
  if (connected && socket === ws) ws.send(JSON.stringify({ type: "meet-result", ...command, status, reason }));
}
function finishMeet(entry, status, reason) {
  if (meetPending.get(entry.command.id) !== entry) return;
  meetPending.delete(entry.command.id);
  clearTimeout(entry.timer);
  meetReply(entry.ws, entry.command, status, reason);
}
function removeMeet(frame) {
  if (meetFrames.get(frame.tab) !== frame) return;
  meetFrames.delete(frame.tab);
  clearTimeout(frame.expiry);
  for (const entry of [...meetPending.values()]) if (entry.frame === frame) finishMeet(entry, "unknown", "disconnected");
  scheduleMeet();
}
function postMeet(frame, message) {
  try { frame.port.postMessage(message); return true; }
  catch { removeMeet(frame); return false; }
}
function enableMeet(value) {
  if (meetEnabled === value) return;
  meetEnabled = value;
  for (const entry of [...meetPending.values()]) finishMeet(entry, "unknown", "stopped");
  for (const frame of meetFrames.values()) {
    clearTimeout(frame.expiry);
    frame.report = null; frame.ready = false;
    postMeet(frame, { type: "meet-config", instance: frame.instance, enabled: value });
  }
  scheduleMeet();
}
function sendMeet() {
  if (!connected || !meetEnabled) return;
  const now = performance.now();
  const calls = [];
  for (const frame of meetFrames.values()) {
    if (frame.report && now - frame.at >= 3000) {
      frame.report = null;
      for (const entry of [...meetPending.values()]) if (entry.frame === frame) finishMeet(entry, "unknown", "disconnected");
    }
    if (frame.report) calls.push({ tab: frame.tab, instance: frame.instance, ...frame.report });
  }
  socket.send(JSON.stringify({ type: "meet-state", calls }));
}
function scheduleMeet() {
  if (!meetEnabled || meetTimer) return;
  meetTimer = setTimeout(() => { meetTimer = 0; sendMeet(); }, 200);
}
function admitMeet(p) {
  if (!meetSender(p.sender) || (meetFrames.size >= 64 && !meetFrames.has(p.sender.tab.id)) || nextMeet >= 18446744073709551615n) {
    p.disconnect(); return;
  }
  const tab = p.sender.tab.id;
  const previous = meetFrames.get(tab);
  if (previous) { postMeet(previous, { type: "meet-config", instance: previous.instance, enabled: false }); removeMeet(previous); }
  const frame = { tab, instance: String(++nextMeet), port: p, document: p.sender.documentId, report: null, ready: false, sample: 0n, at: 0 };
  meetFrames.set(tab, frame);
  p.onMessage.addListener((message) => {
    if (meetFrames.get(tab) !== frame || !meetSender(p.sender) || p.sender.documentId !== frame.document || !message || !meetBounded(message, 8192)) return;
    if (!meetEnabled) return;
    if (message.type === "meet-ready" && message.instance === frame.instance) { frame.ready = true; return; }
    if (!frame.ready) return;
    if (message.type === "meet-state") {
      const report = meetState(message.report);
      if (!report || BigInt(report.sample) <= frame.sample) return;
      frame.sample = BigInt(report.sample); frame.at = performance.now(); frame.report = report;
      clearTimeout(frame.expiry);
      frame.expiry = setTimeout(() => {
        if (meetFrames.get(tab) !== frame) return;
        frame.report = null;
        for (const entry of [...meetPending.values()]) if (entry.frame === frame) finishMeet(entry, "unknown", "disconnected");
        sendMeet();
      }, 3000);
      scheduleMeet();
    } else if (message.type === "meet-result" && meetBounded(message, 2048)) {
      const entry = meetPending.get(message.id);
      if (!entry || entry.frame !== frame || entry.ws !== socket) return;
      const result = meetResult(message, entry.command);
      if (result) finishMeet(entry, result.status, result.reason);
    }
  });
  p.onDisconnect.addListener(() => removeMeet(frame));
  postMeet(frame, { type: "meet-config", instance: frame.instance, enabled: meetEnabled });
}
function routeMeet(message, ws) {
  if (!meetBounded(message, 2048)) return;
  const command = meetCommand(message);
  if (!command || meetPending.has(command.id)) return;
  const reject = (reason) => meetReply(ws, command, "rejected", reason);
  if (!meetEnabled) return reject("disabled");
  const frame = meetFrames.get(command.tab);
  if (!frame || !frame.ready || !meetSender(frame.port.sender) || frame.port.sender.documentId !== frame.document || frame.instance !== command.instance || !frame.report || performance.now() - frame.at >= 3000 || frame.report.call !== command.call || frame.report[command.control]?.revision !== command.revision) return reject("stale");
  if (!frame.report[command.control].available) return reject("unavailable");
  if (meetPending.size >= 8 || [...meetPending.values()].some((p) => p.frame === frame && p.command.call === command.call && p.command.control === command.control)) return reject("busy");
  const entry = { command, frame, ws };
  meetPending.set(command.id, entry);
  entry.timer = setTimeout(() => finishMeet(entry, "unknown", "not-confirmed"), 5000);
  postMeet(frame, { type: "meet-command", ...command });
}

function rememberTab(tab) {
  if (!tab || tab.id === undefined) return;
  tabs.set(tab.id, { title: tab.title || "", url: tab.url || "", muted: !!(tab.mutedInfo && tab.mutedInfo.muted), favIconUrl: tab.favIconUrl || "" });
}

api.tabs.onUpdated.addListener((_id, change, tab) => {
  rememberTab(tab);
  if (change.mutedInfo || change.title) schedule();
});
api.tabs.onRemoved.addListener((id) => {
  tabs.delete(id);
  for (const [key, frame] of frames) if (frame.tabId === id) frames.delete(key);
  const meet = meetFrames.get(id);
  if (meet) removeMeet(meet);
  schedule();
});

api.runtime.onConnect.addListener((p) => {
  if (p.name === "meet") { admitMeet(p); return; }
  if (p.name !== "media" || !p.sender || !p.sender.tab) return;
  if (meetOrigin(p.sender)) { p.disconnect(); return; }
  const tabId = p.sender.tab.id;
  const key = tabId + ":" + (p.sender.frameId || 0);
  let youtubeMusic = false;
  try {
    youtubeMusic = p.sender.frameId === 0 && new URL(p.sender.url).origin === "https://music.youtube.com" &&
      (p.sender.origin === undefined || p.sender.origin === "https://music.youtube.com");
  } catch { /* Missing or invalid sender identity cannot grant ratings. */ }
  rememberTab(p.sender.tab);
  frames.set(key, { tabId, instance: String(++nextFrame), youtubeMusic, port: p, report: null });
  p.onMessage.addListener((message) => {
    const frame = frames.get(key);
    if (!frame || frame.port !== p) return;
    try {
      if (typeof message?.report !== "string" || message.report.length > 16384) return;
      const report = JSON.parse(message.report);
      if (!report || typeof report !== "object" || Array.isArray(report)) return;
      frame.report = report;
    } catch {
      return;
    }
    schedule();
  });
  p.onDisconnect.addListener(() => {
    if (frames.get(key) && frames.get(key).port === p) frames.delete(key);
    schedule();
  });
});

// The popup asks for status; reply synchronously.
api.runtime.onMessage.addListener((message, _sender, reply) => {
  if (!message || message.type !== "status") return false;
  reply({ connected, refused, port, browser: BROWSER, playing: sessions(frames, tabs).length });
  return false;
});

// thumbnail draws artwork into a 64 px square PNG, as base64.
async function thumbnail(url) {
  // Page-chosen URLs: never send the user's cookies with them.
  const response = await fetch(url, { credentials: "omit" });
  if (!response.ok) throw new Error(String(response.status));
  const bitmap = await createImageBitmap(await response.blob());
  const canvas = new OffscreenCanvas(128, 128);
  const scale = Math.min(128 / bitmap.width, 128 / bitmap.height);
  const w = Math.max(1, Math.round(bitmap.width * scale));
  const h = Math.max(1, Math.round(bitmap.height * scale));
  canvas.getContext("2d").drawImage(bitmap, (128 - w) / 2, (128 - h) / 2, w, h);
  const bytes = new Uint8Array(await (await canvas.convertToBlob({ type: "image/png" })).arrayBuffer());
  let binary = "";
  for (let i = 0; i < bytes.length; i += 0x8000) binary += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
  return btoa(binary);
}

function schedule() {
  if (sendTimer) return;
  sendTimer = setTimeout(() => {
    sendTimer = 0;
    send();
  }, 200);
}

function send() {
  if (!connected || !mediaEnabled) return;
  const ws = socket;
  const list = sessions(frames, tabs);
  for (const s of list) {
    let cached = sentArt.get(s.id);
    if (!cached || cached.instance !== s.instance || cached.key !== s.artKey) {
      cached = { instance: s.instance, key: s.artKey, ready: false, sent: false };
      sentArt.set(s.id, cached);
      if (s.artKey) {
        const current = cached;
        thumbnail(s.artKey).catch(() => "").then((art) => {
          if (!connected || socket !== ws || sentArt.get(s.id) !== current) return;
          current.art = art;
          current.ready = true;
          schedule(); // Rebuild from current reports; never resend this old snapshot.
        });
      }
    }
    if (cached.ready && !cached.sent) {
      s.art = cached.art;
      cached.sent = true;
    }
  }
  for (const id of sentArt.keys()) if (!list.some((s) => s.id === id)) sentArt.delete(id);
  if (connected && socket === ws) ws.send(JSON.stringify({ type: "sessions", sessions: list }));
}

function connect() {
  clearTimeout(retry);
  enableMeet(false);
  capabilitiesSeen = false; mediaEnabled = true; meetNegotiated = false;
  const ws = new WebSocket("ws://127.0.0.1:" + port + "/nowplaying");
  socket = ws;
  ws.onopen = () => {
    if (socket !== ws) return;
    connected = true;
    attempts = 0;
    refused = ""; // A refusal, if any, arrives right after hello.
    sentArt.clear(); // Snoofer starts with no artwork for this connection.
    ws.send(JSON.stringify({ type: "hello", protocol: PROTOCOL, version: VERSION, browser: BROWSER, meet: 1 }));
    send();
  };
  ws.onmessage = (event) => {
    if (socket !== ws) return;
    let message;
    try {
      message = JSON.parse(event.data);
    } catch {
      return;
    }
    if (!message || typeof message !== "object") return;
    if (message.type === "capabilities") {
      if (typeof message.mediaEnabled !== "boolean" || typeof message.meetEnabled !== "boolean") return;
      const refresh = message.mediaEnabled && (!capabilitiesSeen || !mediaEnabled);
      capabilitiesSeen = true; mediaEnabled = message.mediaEnabled;
      meetNegotiated = message.meet === 1;
      enableMeet(meetNegotiated && message.meetEnabled);
      if (refresh) { sentArt.clear(); schedule(); }
      return;
    }
    if (message.type === "refused") {
      refused = String(message.reason || "Snoofer refused this extension");
      return;
    }
    if (message.type === "meet-command") { if (meetNegotiated) routeMeet(message, ws); return; }
    if (message.type !== "command" || !mediaEnabled) return;
    const action = route(message, frames);
    if (action.kind === "mute") api.tabs.update(action.tabId, { muted: action.muted });
    if (action.kind === "frame") action.frame.port.postMessage(action.message);
  };
  ws.onclose = () => {
    if (socket !== ws) return; // Replaced after a port change.
    enableMeet(false);
    connected = false;
    socket = null;
    // A refusal will not change until something is updated: retry slowly.
    retry = setTimeout(connect, refused ? 30000 : backoff(attempts++));
  };
}

// Traffic every 20 s keeps the connection and Chrome's service worker alive.
setInterval(() => {
  if (connected) socket.send(JSON.stringify({ type: "ping" }));
}, 20000);
setInterval(() => {
  if (!connected || !meetEnabled) return;
  sendMeet();
  for (const frame of meetFrames.values()) postMeet(frame, { type: "meet-observe" });
}, 1000);

api.storage.onChanged.addListener((changes, area) => {
  if (area !== "local" || !changes.port) return;
  port = validPort(changes.port.newValue) || DEFAULT_PORT;
  refused = "";
  attempts = 0;
  const old = socket;
  connected = false;
  connect();
  if (old) old.close();
});

// Content scripts reach only pages loaded after install or update, so inject
// them into tabs that are already open. page.js ignores a second run.
async function injectOpenTabs() {
  const open = await api.tabs.query({ url: ["http://*/*", "https://*/*"] });
  for (const tab of open) {
    const target = { tabId: tab.id, allFrames: true };
    try {
      if (meetOrigin({ url: tab.url })) {
        await api.scripting.executeScript({ target: { tabId: tab.id, frameIds: [0] }, files: ["logic.js", "meet.js"] });
      } else {
        await api.scripting.executeScript({ target, files: ["page.js"], world: "MAIN" });
        await api.scripting.executeScript({ target, files: ["bridge.js"] });
      }
    } catch {
      // Pages the browser protects (stores, settings) refuse injection.
    }
  }
}
api.runtime.onInstalled.addListener(() => {
  injectOpenTabs().catch(() => {});
});

api.tabs.query({}).then((all) => all.forEach(rememberTab));
api.storage.local.get("port").then((stored) => {
  port = validPort(stored.port) || DEFAULT_PORT;
  connect();
});
