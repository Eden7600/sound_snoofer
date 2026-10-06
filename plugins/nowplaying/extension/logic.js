// Pure helpers for the service worker, kept free of extension APIs so they
// can be tested with Node.

// browserName reports the browser Snoofer pairs with Windows sessions.
export function browserName(brands, isBrave) {
  if (isBrave) return "Brave";
  const names = (brands || []).map((b) => b.brand);
  if (names.includes("Brave")) return "Brave";
  if (names.includes("Microsoft Edge")) return "Edge";
  if (names.includes("Opera")) return "Opera";
  if (names.includes("Vivaldi")) return "Vivaldi";
  return "Chrome";
}

function site(url) {
  try {
    const host = new URL(url).hostname;
    return host.replace(/^www\./, "");
  } catch {
    return "";
  }
}

// sessions builds Snoofer's session list from frame reports and tab details.
// frames maps "tab:frame" to {tabId, report}; tabs maps tab IDs to
// {title, url, muted, favIconUrl}. artKey names the artwork; the worker adds
// the image itself only when the key changes.
export function sessions(frames, tabs) {
  const out = [];
  for (const [id, frame] of frames) {
    const r = frame.report;
    if (!r || !r.present) continue;
    const tab = tabs.get(frame.tabId) || {};
    out.push({
      id,
      tab: frame.tabId,
      site: site(tab.url || ""),
      title: r.title || tab.title || "",
      artist: r.artist || "",
      album: r.album || "",
      artKey: r.artwork || tab.favIconUrl || "",
      state: r.state === "playing" ? "playing" : "paused",
      positionMs: r.positionMs || 0,
      durationMs: r.durationMs || 0,
      updatedMs: r.at || 0,
      rate: r.rate || 1,
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
export function route(command, frames, tabs) {
  const frame = frames.get(command.id);
  if (!frame) return { kind: "missing" };
  if (command.op === "mute") {
    return { kind: "mute", tabId: frame.tabId, muted: !(tabs.get(frame.tabId) || {}).muted };
  }
  if (!["toggle", "next", "prev", "seek"].includes(command.op)) return { kind: "invalid" };
  return { kind: "frame", frame, message: { op: command.op, value: command.value || 0 } };
}

// backoff is the reconnect delay after n failed attempts.
export function backoff(n) {
  return Math.min(30000, 1000 * 2 ** Math.min(n, 5));
}
