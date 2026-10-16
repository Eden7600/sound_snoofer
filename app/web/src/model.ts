import type { Control, ViewState, Meter, Connection } from "./types.ts";
export function controlsByID(state: ViewState) {
  return new Map((state.Controls || []).map((c) => [c.ID, c]));
}
export function compatible(control: Partial<Control>, dial: boolean) {
  if (control.Kind === "text") return false; // Text entry has no deck position.
  if (control.Kind === "status") return !dial; // Display-only keys never act.
  const ops = control.Operations || [];
  return dial ? ops.includes("adjust") : ops.includes("press") || ops.includes("set");
}
export function tone(control?: Partial<Control>) {
  if (!control || !control.Available) return "muted";
  const status = control.Status || "";
  if (/stalled/i.test(control.Value || "") || control.Value === "Error") return "critical";
  if (/failed|error/i.test(status)) return "critical";
  if (/pending|wait|fallback|override/i.test(status) || control.Value === "Wait") return "attention";
  if (control.Value === "On" && /mute/.test(control.ID || "")) return "critical";
  if (control.Value === "On" || control.Value === "Playing" || control.Value === "Loop" || control.Value === "Active")
    return "active";
  return "";
}
export function meterValue(meter: Meter | undefined, now = Date.now()) {
  if (
    !meter?.Present ||
    !meter.Known ||
    !Number.isFinite(meter.DB) ||
    now - Date.parse(meter.At) > 500 ||
    !Number.isFinite(Date.parse(meter.At))
  )
    return null;
  return Math.min(0, Math.max(-60, meter.DB));
}
export function display(control: Control) {
  if (!control.Available) return control.Value === "N/A" ? "N/A" : "Unavailable";
  return control.OptionLabels?.[control.Value] ?? control.Value ?? "";
}
export function gridMove(index: number, key: string, columns: number, count: number) {
  const steps: Record<string, number> = { ArrowLeft: -1, ArrowRight: 1, ArrowUp: -columns, ArrowDown: columns };
  const step = steps[key];
  if (step === undefined) return index;
  return Math.max(0, Math.min(count - 1, index + step));
}

// numericValue reads the number in a dial value such as "62%", "Sync 62%" or "4000K".
export function numericValue(text?: string) {
  const match = /(-?\d+(?:\.\d+)?)/.exec(text || "");
  return match ? Number(match[1]) : null;
}
// sceneRoom recovers the room from a scene Label of the form "<Room> <Scene>".
export function sceneRoom(control?: Partial<Control>) {
  const label = control?.Label || "",
    name = control?.ShortLabel || "";
  return label.endsWith(name) ? label.slice(0, label.length - name.length).trim() : "";
}
// Connection reports (Kind "connection"). Go zero times marshal as year 0001.
export function timeValue(iso?: string) {
  const t = Date.parse(iso || "");
  return Number.isFinite(t) && t > 0 ? t : null;
}
export function relativeTime(iso: string | undefined, now = Date.now()) {
  const t = timeValue(iso);
  if (t === null) return "";
  const s = Math.max(0, Math.round((now - t) / 1000));
  if (s < 2) return "just now";
  if (s < 60) return s + "s ago";
  if (s < 3600) return Math.floor(s / 60) + "m ago";
  if (s < 86400) return Math.floor(s / 3600) + "h ago";
  return Math.floor(s / 86400) + "d ago";
}
export function detail(conn: Partial<Connection> | undefined, label: string) {
  return (conn?.Details || []).find((d) => d.Label === label)?.Value ?? "";
}
// intervalMs parses a provider "Interval" detail such as "1s", "100 ms" or "2 min".
export function intervalMs(text?: string) {
  const m = /^\s*([\d.]+)\s*(ms|s|sec|m|min)\s*$/i.exec(text || "");
  if (!m?.[2]) return null;
  const n = Number(m[1]),
    unit = m[2].toLowerCase();
  return unit === "ms" ? n : unit === "s" || unit === "sec" ? n * 1000 : n * 60000;
}
export function stale(conn: Partial<Connection> | undefined, now = Date.now()) {
  const interval = intervalMs(detail(conn, "Interval")),
    last = timeValue(conn?.LastActivity);
  return (
    interval !== null &&
    last !== null &&
    ["connected", "ready", "attention"].includes(conn?.State || "") &&
    now - last > 3 * interval
  );
}
export function connectionTone(conn?: Partial<Connection>) {
  switch (conn?.State) {
    case "connected":
    case "ready":
      return "active";
    case "connecting":
    case "attention":
      return "attention";
    case "error":
      return "critical";
    case "disconnected":
      return detail(conn, "Required") === "Yes" ? "critical" : "";
    default:
      return "muted";
  }
}
export function connectionText(control: Omit<Partial<Control>, "Connection"> & { Connection?: Partial<Connection> }) {
  const c = control.Connection || {},
    lines = [control.Label + " (" + (control.Group || "") + ")", "State: " + (control.Value || c.State || "")];
  if (c.Endpoint) lines.push("Endpoint: " + c.Endpoint);
  for (const [label, iso] of [
    ["Since", c.Since],
    ["Last activity", c.LastActivity],
  ]) {
    const time = timeValue(iso);
    if (time !== null) lines.push(label + ": " + new Date(time).toISOString());
  }
  const errorTime = timeValue(c.LastErrorAt);
  if (c.LastError)
    lines.push(
      "Last error: " + c.LastError + (errorTime !== null ? " (" + new Date(errorTime).toISOString() + ")" : ""),
    );
  for (const d of c.Details || []) lines.push(d.Label + ": " + d.Value);
  return lines.join("\n");
}
// meterBallistics mirrors the deck dial: instant attack, 24 dB/s release and a
// 1.5 s peak hold falling at 18 dB/s. A null reading resets the state, so an
// expired meter never decays into fake silence.
export type MeterState = { level: null; peak: null } | { level: number; peak: number; at: number; peakFrom: number };
export function meterBallistics(state: MeterState | null, db: number | null, now: number): MeterState {
  if (db === null) return { level: null, peak: null };
  if (state?.level == null) return { level: db, peak: db, at: now, peakFrom: now };
  const dt = (now - state.at) / 1000;
  const level = db >= state.level ? db : Math.max(db, state.level - 24 * dt);
  let { peak, peakFrom } = state;
  if (level >= peak) {
    peak = level;
    peakFrom = now;
  } else if (now - peakFrom > 1500) peak = Math.max(level, peak - 18 * dt);
  return { level, peak, at: now, peakFrom };
}
// gainPosition places a displayed value within its range: dB gain over
// -60…+12 (with the 0 dB mark) or a percentage. Unparsed values have none.
export function gainPosition(value?: string) {
  const db = /^(-?\d+(?:\.\d+)?) dB$/.exec(value || "");
  if (db) return { position: Math.min(1, Math.max(0, (Number(db[1]) + 60) / 72)), zero: 60 / 72 };
  const pct = /(\d+(?:\.\d+)?)%/.exec(value || "");
  if (pct) return { position: Math.min(1, Math.max(0, Number(pct[1]) / 100)), zero: null };
  return null;
}

// progressAt interpolates a session's position (ms) at now from its sample:
// PositionMs at AtMs, advancing at Rate while playing, clamped to the length.
export function progressAt(
  s: Partial<{ Status: string; PositionMs: number; DurationMs: number; AtMs: number; Rate: number }> | undefined,
  now = Date.now(),
) {
  let position = s?.PositionMs || 0;
  if (s?.Status === "Playing" && s.AtMs) position += (now - s.AtMs) * (s.Rate !== undefined && s.Rate > 0 ? s.Rate : 1);
  if (s?.DurationMs !== undefined && s.DurationMs > 0) position = Math.min(position, s.DurationMs);
  return Math.max(0, Math.round(position));
}
