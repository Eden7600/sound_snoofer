// Pure helpers for the background script, free of extension APIs so Node can
// test them. A classic script: Chrome's service worker loads it with
// importScripts, Firefox lists it before background.js, and both read
// globalThis.SnooferLogic.
(() => {
  // PROTOCOL is the Snoofer bridge message format this extension speaks.
  const PROTOCOL = 1;
  const DEFAULT_PORT = 47815;

  // browserName reports the browser Snoofer pairs with its Windows session.
  function browserName(brands, isBrave, userAgent) {
    if (isBrave) return "Brave";
    if (/\bFirefox\//.test(userAgent || "")) return "Firefox";
    const names = (brands || []).map((b) => b.brand);
    if (names.includes("Brave")) return "Brave";
    if (names.includes("Microsoft Edge")) return "Edge";
    if (names.includes("Opera")) return "Opera";
    if (names.includes("Vivaldi")) return "Vivaldi";
    return "Chrome";
  }

  function site(url) {
    try {
      return new URL(url).hostname.replace(/^www\./, "");
    } catch {
      return "";
    }
  }

  // Reports come from page scripts, so every field is coerced to its type.
  const text = (v) => (typeof v === "string" ? v : "");
  const count = (v) => (typeof v === "number" && Number.isFinite(v) && v >= 0 ? v : 0);

  // webUrl keeps http(s) URLs only: artwork is fetched, and other schemes must not be.
  function webUrl(url) {
    try {
      const protocol = new URL(url).protocol;
      return protocol === "http:" || protocol === "https:" ? url : "";
    } catch {
      return "";
    }
  }

  // sessions builds Snoofer's session list. frames maps "tab:frame" to
  // {tabId, report}; tabs maps tab IDs to {title, url, muted, favIconUrl}.
  // artKey names the artwork; the background adds the image only when the key
  // changes.
  function sessions(frames, tabs) {
    const out = [];
    for (const [id, frame] of frames) {
      const r = frame.report;
      if (!r || !r.present) continue;
      const tab = tabs.get(frame.tabId) || {};
      out.push({
        id,
        tab: frame.tabId,
        site: site(tab.url || ""),
        title: text(r.title) || text(tab.title),
        artist: text(r.artist),
        album: text(r.album),
        artKey: webUrl(text(r.artwork)) || webUrl(text(tab.favIconUrl)),
        state: r.state === "playing" ? "playing" : "paused",
        positionMs: count(r.positionMs),
        durationMs: count(r.durationMs),
        updatedMs: count(r.at),
        rate: count(r.rate) || 1,
        canNext: !!r.canNext,
        canPrev: !!r.canPrev,
        canSeek: !!r.canSeek,
        muted: !!tab.muted,
      });
    }
    out.sort((a, b) => a.id.localeCompare(b.id));
    return out.slice(0, 64);
  }

  // route decides how a command from Snoofer is carried out: tab mute through
  // the tabs API, everything else by the frame that reported the session.
  function route(command, frames, tabs) {
    const frame = frames.get(command.id);
    if (!frame) return { kind: "missing" };
    if (command.op === "mute") {
      return { kind: "mute", tabId: frame.tabId, muted: !(tabs.get(frame.tabId) || {}).muted };
    }
    if (!["toggle", "next", "prev", "seek"].includes(command.op)) return { kind: "invalid" };
    return { kind: "frame", frame, message: { op: command.op, value: command.value || 0 } };
  }

  // backoff is the reconnect delay after n failed attempts.
  function backoff(n) {
    return Math.min(30000, 1000 * 2 ** Math.min(n, 5));
  }

  // validPort accepts the ports Snoofer's settings allow.
  function validPort(value) {
    const port = Number(value);
    return Number.isInteger(port) && port >= 1024 && port <= 65535 ? port : 0;
  }

  globalThis.SnooferLogic = { PROTOCOL, DEFAULT_PORT, browserName, sessions, route, backoff, validPort };
})();
