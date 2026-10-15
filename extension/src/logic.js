// Pure helpers for the background script, free of extension APIs so Node can
// test them. A classic script: Chrome's service worker loads it with
// importScripts, Firefox lists it before background.js, and both read
// globalThis.SnooferLogic.
(() => {
  // PROTOCOL is the Snoofer bridge message format this extension speaks.
  const PROTOCOL = 2;
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
  const validTrack = (v) => typeof v === "string" && /^[\x21-\x7e]{1,128}$/.test(v);
  const ratings = ["none", "like", "dislike"];

  function ratingFields(frame) {
    const r = frame.report || {};
    if (!frame.youtubeMusic || r.present !== true || r.rating === undefined) return {};
    const known = ratings.includes(r.rating);
    return {
      ...(validTrack(r.track) ? { track: r.track } : {}),
      rating: known && typeof r.canRate === "boolean" ? r.rating : "unknown",
      canRate: known && r.canRate === true && validTrack(r.track) && r.reloadRequired !== true,
    };
  }

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
  // {tabId, instance, report}; tabs maps tab IDs to {title, url, muted, favIconUrl}.
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
        instance: frame.instance,
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
        canNext: r.reloadRequired !== true && !!r.canNext,
        canPrev: r.reloadRequired !== true && !!r.canPrev,
        canSeek: r.reloadRequired !== true && !!r.canSeek,
        muted: !!tab.muted,
        ...(r.reloadRequired === true ? { reloadRequired: true } : {}),
        ...ratingFields(frame),
      });
    }
    out.sort((a, b) => a.id.localeCompare(b.id));
    return out.slice(0, 64);
  }

  // route decides how a command from Snoofer is carried out: tab mute through
  // the tabs API, everything else by the frame that reported the session.
  function route(command, frames) {
    if (!command || typeof command !== "object") return { kind: "invalid" };
    const frame = frames.get(command.id);
    if (!frame) return { kind: "missing" };
    if (typeof command.instance !== "string" || !command.instance || command.instance.length > 64 || command.instance !== frame.instance) return { kind: "stale" };
    if (["toggle", "mute"].includes(command.op) && command.value !== 0 && command.value !== 1) return { kind: "invalid" };
    if (command.op === "mute") {
      return { kind: "mute", tabId: frame.tabId, muted: command.value === 1 };
    }
    if (frame.report?.reloadRequired === true) return { kind: "stale" };
    if (command.op === "rate") {
      if (!validTrack(command.track) || !ratings.includes(command.rating)) return { kind: "invalid" };
      const current = ratingFields(frame);
      if (!current.canRate || current.track !== command.track) return { kind: "stale" };
      return { kind: "frame", frame, message: { op: "rate", track: command.track, rating: command.rating } };
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

  const meetCounter = (v) => typeof v === "string" && /^[1-9][0-9]{0,19}$/.test(v) && BigInt(v) <= 18446744073709551615n;
  const meetBounded = (v, bytes) => new TextEncoder().encode(JSON.stringify(v)).length <= bytes;
  function meetOrigin(sender) {
    if (sender?.origin === "https://meet.google.com") return true;
    try { return new URL(sender?.url).origin === "https://meet.google.com"; } catch { return false; }
  }
  function meetSender(sender) {
    try {
      return sender?.frameId === 0 && Number.isSafeInteger(sender.tab?.id) && sender.tab.id >= 0 &&
        new URL(sender.url).origin === "https://meet.google.com" &&
        (sender.origin === undefined || sender.origin === "https://meet.google.com");
    } catch { return false; }
  }
  function meetState(value) {
    if (!value || !["joined", "prejoin", "unknown"].includes(value.state) || !meetCounter(value.sample) ||
        (value.state === "joined" ? !meetCounter(value.call) : value.call !== "")) return null;
    const result = { sample: value.sample, call: value.call, state: value.state };
    for (const name of ["mic", "camera"]) {
      const c = value[name];
      if (!c || (typeof c.value !== "boolean" && c.value !== null) || typeof c.available !== "boolean" || !meetCounter(c.revision) ||
          !["", "unknown", "unavailable", "not-in-call", "reload-required"].includes(c.reason) ||
          (c.available ? value.state !== "joined" || c.value === null || c.reason !== "" : c.reason === "")) return null;
      if (c.identity !== undefined && c.identity !== "0" && !meetCounter(c.identity)) return null;
      result[name] = { value: c.value, revision: c.revision, available: c.available, reason: c.reason };
      if (c.identity !== undefined) result[name].identity = c.identity;
    }
    return result;
  }
  function meetCommand(value) {
    if (!value || !meetCounter(value.id) || !Number.isSafeInteger(value.tab) || value.tab < 0 ||
        !meetCounter(value.instance) || !meetCounter(value.call) || !meetCounter(value.revision) ||
        !["mic", "camera"].includes(value.control) || typeof value.value !== "boolean") return null;
    const { id, tab, instance, call, control, revision, value: desired } = value;
    return { id, tab, instance, call, control, revision, value: desired };
  }
  function meetResult(value, command) {
    const echo = meetCommand(value);
    if (!echo || Object.keys(echo).some((key) => echo[key] !== command[key])) return null;
    const reasons = { observed: [""], rejected: ["stale", "unavailable", "busy", "disabled"], unknown: ["not-confirmed", "disconnected", "stopped"] };
    if (!Object.hasOwn(reasons, value.status) || !reasons[value.status].includes(value.reason)) return null;
    return { ...echo, status: value.status, reason: value.reason };
  }

  globalThis.SnooferLogic = { PROTOCOL, DEFAULT_PORT, browserName, sessions, route, backoff, validPort, meetCounter, meetBounded, meetOrigin, meetSender, meetState, meetCommand, meetResult };
})();
