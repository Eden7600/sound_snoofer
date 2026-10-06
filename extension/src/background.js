// Service worker: keeps one WebSocket to Snoofer, merges frame reports into
// sessions, sends artwork as 64 px PNGs once per change, and carries out
// Snoofer's commands.
import config from "./config.js";
import { backoff, browserName, route, sessions } from "./logic.js";

const VERSION = chrome.runtime.getManifest().version;
const BROWSER = browserName(navigator.userAgentData && navigator.userAgentData.brands, !!navigator.brave);
const frames = new Map(); // "tab:frame" to {tabId, port, report}.
const tabs = new Map(); // Tab ID to {title, url, muted, favIconUrl}.
const sentArt = new Map(); // Session ID to the artwork key last sent.
let socket = null;
let attempts = 0;
let sendTimer = 0;

function rememberTab(tab) {
  if (!tab || tab.id === undefined) return;
  tabs.set(tab.id, { title: tab.title || "", url: tab.url || "", muted: !!(tab.mutedInfo && tab.mutedInfo.muted), favIconUrl: tab.favIconUrl || "" });
}

chrome.tabs.onUpdated.addListener((_id, change, tab) => {
  rememberTab(tab);
  if (change.mutedInfo || change.title) schedule();
});
chrome.tabs.onRemoved.addListener((id) => {
  tabs.delete(id);
  for (const [key, frame] of frames) if (frame.tabId === id) frames.delete(key);
  schedule();
});

chrome.runtime.onConnect.addListener((port) => {
  if (port.name !== "media" || !port.sender || !port.sender.tab) return;
  const tabId = port.sender.tab.id;
  const key = tabId + ":" + (port.sender.frameId || 0);
  rememberTab(port.sender.tab);
  frames.set(key, { tabId, port, report: null });
  port.onMessage.addListener((message) => {
    try {
      const frame = frames.get(key);
      if (frame) frame.report = JSON.parse(message.report);
    } catch {
      return;
    }
    schedule();
  });
  port.onDisconnect.addListener(() => {
    if (frames.get(key) && frames.get(key).port === port) frames.delete(key);
    schedule();
  });
});

// thumbnail draws artwork into a 64 px square PNG, as base64.
async function thumbnail(url) {
  const response = await fetch(url);
  if (!response.ok) throw new Error(String(response.status));
  const bitmap = await createImageBitmap(await response.blob());
  const canvas = new OffscreenCanvas(64, 64);
  const scale = Math.min(64 / bitmap.width, 64 / bitmap.height);
  const w = Math.max(1, Math.round(bitmap.width * scale)), h = Math.max(1, Math.round(bitmap.height * scale));
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
  if (!socket || socket.readyState !== WebSocket.OPEN) return;
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
  socket.send(JSON.stringify({ type: "sessions", sessions: list }));
}

function connect() {
  socket = new WebSocket("ws://127.0.0.1:" + config.port + "/nowplaying");
  socket.onopen = () => {
    attempts = 0;
    sentArt.clear(); // Snoofer starts with no artwork for this connection.
    socket.send(JSON.stringify({ type: "hello", token: config.token, version: VERSION, browser: BROWSER }));
    send().catch(() => {});
  };
  socket.onmessage = (event) => {
    let command;
    try {
      command = JSON.parse(event.data);
    } catch {
      return;
    }
    if (command.type !== "command") return;
    const action = route(command, frames, tabs);
    if (action.kind === "mute") chrome.tabs.update(action.tabId, { muted: action.muted });
    if (action.kind === "frame") action.frame.port.postMessage(action.message);
  };
  socket.onclose = () => {
    socket = null;
    setTimeout(connect, backoff(attempts++));
  };
}

// Traffic every 20 s keeps the connection and this worker alive.
setInterval(() => {
  if (socket && socket.readyState === WebSocket.OPEN) socket.send(JSON.stringify({ type: "ping" }));
}, 20000);

chrome.tabs.query({}).then((all) => all.forEach(rememberTab));
connect();
