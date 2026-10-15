import { ui, el, button, panel, badge, withIcon, empty } from "../dom.ts";
import { appState, root, updaters, c } from "../state.ts";
import { connectionText, connectionTone, timeValue, relativeTime, stale } from "../model.ts";

import { copyText } from "../clipboard.ts";
import { viewKey } from "../view-state.ts";
import { pluginStateText, pluginStates } from "../plugin-gate.ts";

function reportCard(id: string, parent: HTMLElement) {
  const initial = c(id);
  if (!initial) return;
  const card = el(
      "section",
      "min-w-0 rounded-[9px] border border-l-[3px] border-seam border-l-line bg-sidebar px-4 py-3.5 data-[tone=active]:border-l-active data-[tone=attention]:border-l-attention data-[tone=critical]:border-l-critical",
    ),
    head = el("div", "mb-2 flex items-center justify-between gap-2.5"),
    name = el("h3", "text-sm font-bold", initial.Label),
    state = badge(""),
    copied = el("small", "ml-1.5 text-active"),
    copy = withIcon(
      button("Copy Details", () => copyText(connectionText(c(id) || initial), copied), ui.small),
      "copy",
    );
  viewKey(copy, id, "copy");
  card.dataset.part = "app-card";
  const endpoint = el("code", "mb-1.5 block font-mono text-xs break-words text-muted"),
    times = el("p", "mb-1.5 text-xs text-muted"),
    error = el("p", "mb-2 text-xs break-words"),
    list = el("dl", "my-2 mb-2.5 grid grid-cols-[max-content_minmax(0,1fr)] gap-x-3 gap-y-[3px] text-xs");
  endpoint.dataset.part = "endpoint";
  times.dataset.part = "app-times";
  error.dataset.part = "app-error";
  list.dataset.part = "app-details";
  head.append(name, state);
  card.append(head, endpoint, times, error, list);
  const foot = el("div", ui.actions);
  foot.append(copy, copied);
  card.append(foot);
  parent.append(card);
  updaters.push(() => {
    const item = c(id);
    if (!item) return;
    const conn = item.Connection,
      now = Date.now(),
      accent = connectionTone(conn);
    state.textContent = item.Value || conn?.State || "";
    state.dataset.tone = accent;
    card.dataset.tone = accent;
    endpoint.textContent = conn?.Endpoint || "";
    endpoint.hidden = !conn?.Endpoint;
    const parts = [];
    if (timeValue(conn?.Since) !== null) parts.push("Since " + relativeTime(conn?.Since, now));
    if (timeValue(conn?.LastActivity) !== null)
      parts.push("Last activity " + relativeTime(conn?.LastActivity, now) + (stale(conn, now) ? " · stale" : ""));
    times.textContent = parts.join(" · ");
    times.hidden = !parts.length;
    times.className = "mb-1.5 text-xs " + (stale(conn, now) ? "text-attention" : "text-muted");
    const current =
      ["error", "attention", "connecting"].includes(conn?.State || "") ||
      (conn?.State === "disconnected" && accent === "critical");
    error.textContent = conn?.LastError
      ? (current ? "" : "Last error: ") +
        conn?.LastError +
        (timeValue(conn?.LastErrorAt) !== null ? " · " + relativeTime(conn?.LastErrorAt, now) : "")
      : "";
    error.hidden = !conn?.LastError;
    error.className = "mb-2 text-xs break-words " + (current ? "text-critical" : "text-muted");
    const rows = (conn?.Details || []).filter((d) => d.Label !== "Required");
    const key = JSON.stringify(rows);
    if (list.dataset.key !== key) {
      list.dataset.key = key;
      list.replaceChildren();
      for (const d of rows) list.append(el("dt", "text-muted", d.Label), el("dd", "break-words", d.Value));
    }
    list.hidden = !rows.length;
  });
}
export function buildApps() {
  const reports = [...appState.controls.values()]
    .filter((v) => v.Kind === "connection")
    .sort((a, b) => (a.Group || "").localeCompare(b.Group || "") || a.Label.localeCompare(b.Label));
  const bar = el("div", ui.toolbar + " justify-between"),
    counts = el("div", "flex flex-1 flex-wrap gap-2"),
    copied = el("small", "ml-1.5 text-active");
  counts.dataset.part = "summary";
  bar.append(
    counts,
    withIcon(
      viewKey(
        button(
          "Copy All",
          () => copyText(reports.map((r) => connectionText(c(r.ID) || r)).join("\n\n"), copied),
          ui.small,
        ),
        "apps",
        "copy-all",
      ),
      "copy",
    ),
    copied,
  );
  root.append(bar);
  updaters.push(() => {
    const tally = { active: 0, attention: 0, critical: 0, other: 0 };
    for (const r of reports) {
      const t = connectionTone(c(r.ID)?.Connection);
      if (t === "active" || t === "attention" || t === "critical") tally[t]++;
      else tally.other++;
    }
    counts.replaceChildren(
      ...(
        [
          ["active", "OK"],
          ["attention", "Attention"],
          ["critical", "Problems"],
          ["other", "Idle"],
        ] as const
      ).map(([k, label]) => badge(tally[k] + " " + label, k === "other" ? "" : k)),
    );
  });
  if (!reports.length) empty(root, "No third-party apps are reporting. Enable plugins in Plugins.");
  for (const group of [...new Set(reports.map((r) => r.Group || "Other"))]) {
    const card = panel(group, root, "mb-[18px]"),
      grid = el("div", "grid grid-cols-[repeat(auto-fill,minmax(300px,1fr))] gap-3");
    card.append(grid);
    for (const r of reports.filter((v) => (v.Group || "Other") === group)) reportCard(r.ID, grid);
  }
  const idle = Object.entries(pluginStates())
    .filter(([, state]) => state !== "running")
    .sort();
  if (idle.length) {
    const card = panel("Not monitored", root, "mb-[18px]"),
      list = el("ul", "list-disc pl-[18px] text-[13px] text-muted");
    list.dataset.part = "not-monitored";
    card.append(list);
    for (const [id] of idle)
      list.append(el("li", "", (appState.state.PluginLabels?.[id] || id) + " — " + pluginStateText(id)));
  }
}
