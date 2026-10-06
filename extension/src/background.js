// Background script (Chrome service worker, Firefox event page): keeps one
// WebSocket to Snoofer on this computer, merges frame reports into sessions,
// sends artwork as 64 px PNGs once per change, carries out Snoofer's
// commands and answers the popup.
if (typeof importScripts === "function" && !globalThis.SnooferLogic) importScripts("logic.js");
const { PROTOCOL, DEFAULT_PORT, backoff, browserName, route, sessions, validPort } = globalThis.SnooferLogic;
const api = globalThis.browser ?? chrome;

const VERSION = api.runtime.getManifest().version;
const BROWSER = browserName(navigator.userAgentData && navigator.userAgentData.brands, !!navigator.brave, navigator.userAgent);
const frames = new Map(); // "tab:frame" to {tabId, port, report}.
const tabs = new Map(); // Tab ID to {title, url, muted, favIconUrl}.
const sentArt = new Map(); // Session ID to the artwork key last sent.
let port = DEFAULT_PORT;
let socket = null;
let connected = false;
let refused = ""; // Snoofer's reason for refusing this extension, if any.
let attempts = 0;
let retry = 0;
let sendTimer = 0;

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
  schedule();
});

api.runtime.onConnect.addListener((p) => {
  if (p.name !== "media" || !p.sender || !p.sender.tab) return;
  const tabId = p.sender.tab.id;
  const key = tabId + ":" + (p.sender.frameId || 0);
  rememberTab(p.sender.tab);
  frames.set(key, { tabId, port: p, report: null });
  p.onMessage.addListener((message) => {
    const frame = frames.get(key);
    if (!frame) return;
    try {
      frame.report = JSON.parse(message.report);
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
  const response = await fetch(url);
  if (!response.ok) throw new Error(String(response.status));
  const bitmap = await createImageBitmap(await response.blob());
  const canvas = new OffscreenCanvas(64, 64);
  const scale = Math.min(64 / bitmap.width, 64 / bitmap.height);
  const w = Math.max(1, Math.round(bitmap.width * scale));
  const h = Math.max(1, Math.round(bitmap.height * scale));
  canvas.getContext("2d").drawImage(bitmap, (64 - w) / 2, (64 - h) / 2, w, h);
  const bytes = new Uint8Array(await (await canvas.convertToBlob({ type: "image/png" })).arrayBuffer());
  let binary = "";
  for (let i = 0; i < bytes.length; i += 0x8000) binary += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
  return btoa(binary);
}

function schedule() {
  if (sendTimer) return;
  sendTimer = setTimeout(() => {
    sendTimer = 0;
    send().catch(() => {});
  }, 200);
}

async function send() {
  if (!connected) return;
  const list = sessions(frames, tabs);
  for (const s of list) {
    if (s.artKey && sentArt.get(s.id) !== s.artKey) {
      try {
        s.art = await thumbnail(s.artKey);
      } catch {
        s.art = ""; // Unreadable artwork: Snoofer shows the play symbol.
      }
      sentArt.set(s.id, s.artKey);
    }
  }
  for (const id of sentArt.keys()) if (!list.some((s) => s.id === id)) sentArt.delete(id);
  if (connected) socket.send(JSON.stringify({ type: "sessions", sessions: list }));
}

function connect() {
  clearTimeout(retry);
  const ws = new WebSocket("ws://127.0.0.1:" + port + "/nowplaying");
  socket = ws;
  ws.onopen = () => {
    connected = true;
    attempts = 0;
    refused = ""; // A refusal, if any, arrives right after hello.
    sentArt.clear(); // Snoofer starts with no artwork for this connection.
    ws.send(JSON.stringify({ type: "hello", protocol: PROTOCOL, version: VERSION, browser: BROWSER }));
    send().catch(() => {});
  };
  ws.onmessage = (event) => {
    let message;
    try {
      message = JSON.parse(event.data);
    } catch {
      return;
    }
    if (message.type === "refused") {
      refused = String(message.reason || "Snoofer refused this extension");
      return;
    }
    if (message.type !== "command") return;
    const action = route(message, frames, tabs);
    if (action.kind === "mute") api.tabs.update(action.tabId, { muted: action.muted });
    if (action.kind === "frame") action.frame.port.postMessage(action.message);
  };
  ws.onclose = () => {
    if (socket !== ws) return; // Replaced after a port change.
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

api.tabs.query({}).then((all) => all.forEach(rememberTab));
api.storage.local.get("port").then((stored) => {
  port = validPort(stored.port) || DEFAULT_PORT;
  connect();
});
