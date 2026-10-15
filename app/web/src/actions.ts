import type { ActionResult, UIAction } from "./types.ts";
import { appState, c, root } from "./state.ts";
import { button, el, ui } from "./dom.ts";

interface ActionState {
  action: UIAction;
  sequence: number;
  control: string;
  context?: string;
  label: string;
  screen: string;
  busyKey: string;
  until: number;
  status: ActionResult["Status"] | "Sending";
  detail: string;
  dismissed: boolean;
}
const outstanding = new Map<string, ActionState>();
const latest = new Map<string, ActionState>();
const removed = new Map<string, ActionState>();
const session = Array.from(crypto.getRandomValues(new Uint32Array(3)), (part) => part.toString(36)).join("-");
let sequence = 0;
const feedbackKey = (screen: string, control: string) => screen + "\0" + control;

// Mirrors share a reservation, but their feedback still belongs to the exact
// screen and control the user invoked. A bad mirror cycle remains bounded.
function busyKey(id: string) {
  const seen = new Set<string>();
  while (c(id)?.Mirrors && !seen.has(id)) {
    seen.add(id);
    id = c(id)?.Mirrors || id;
  }
  return id;
}
export function isBusy(id: string) {
  const key = busyKey(id);
  return [...outstanding.values()].some((item) => item.busyKey === key);
}
export function pendingCount(screen?: string) {
  return [...outstanding.values()].filter((item) => !screen || item.screen === screen).length;
}
export function beginAction(input: Omit<UIAction, "ID">, rejection = "", context?: string) {
  const control = input.Request?.ID || "lifecycle";
  const action: UIAction = { ...input, ID: session + "-" + ++sequence };
  const item: ActionState = {
    action,
    sequence,
    control,
    context,
    label: c(control)?.Label || (input.Kind === "selection" ? "Plugin selection" : "Plugin lifecycle"),
    screen: appState.screen,
    busyKey: busyKey(control),
    until: Date.now() + 2500,
    status: "Sending",
    detail: "",
    dismissed: false,
  };
  const problem =
    rejection ||
    (appState.reconnecting ? "Host disconnected; action not sent." : "") ||
    (isBusy(control) ? "Control busy; action not sent." : "") ||
    (outstanding.size >= 16 ? "Too many actions are waiting; try again after one finishes." : "");
  latest.set(feedbackKey(item.screen, control), item);
  if (problem) {
    item.status = "Rejected";
    item.detail = problem;
    return null;
  }
  outstanding.set(action.ID, item);
  return action;
}
function retainRemoved(item: ActionState) {
  const previous = removed.get(item.screen);
  const unresolved = (entry: ActionState) => ["Failed", "Rejected", "Unknown"].includes(entry.status);
  if (
    !previous ||
    (unresolved(item) && !unresolved(previous)) ||
    (unresolved(item) === unresolved(previous) && item.sequence >= previous.sequence)
  )
    removed.set(item.screen, item);
}
export function finishAction(id: string, result: ActionResult) {
  const item = outstanding.get(id);
  if (!item) return;
  if (result?.ID !== id || !["Submitted", "Processed", "Rejected", "Failed", "Unknown"].includes(result.Status)) {
    item.status = "Unknown";
    item.detail = "Outcome unknown; the host returned an unmatched result. Still waiting.";
    return;
  }
  outstanding.delete(id);
  item.status = result.Status;
  item.detail = result.Detail || "";
  if (item.control !== "lifecycle" && !c(item.control)) retainRemoved(item);
}
export function uncertainAction(id: string, error: unknown) {
  const item = outstanding.get(id);
  if (!item) return;
  outstanding.delete(id);
  item.status = "Unknown";
  item.detail = "Outcome unknown; " + String(error);
  if (item.control !== "lifecycle" && !c(item.control)) retainRemoved(item);
}
export function disconnectActions() {
  for (const id of outstanding.keys()) uncertainAction(id, "host disconnected. The action was not retried.");
}
export function actionFeedback(id: string, parent: HTMLElement, context = "") {
  const node = el("div", "mt-1 min-w-0 basis-full text-xs break-words");
  node.dataset.actionFeedback = id;
  node.dataset.actionContext = context;
  node.hidden = true;
  parent.append(node);
  return node;
}
function render(node: HTMLElement, item?: ActionState, includeLabel = false) {
  node.hidden = !item || item.dismissed;
  if (!item || item.dismissed) {
    node.replaceChildren();
    delete node.dataset.signature;
    delete node.dataset.actionState;
    return;
  }
  const message = (includeLabel ? item.label + ": " : "") + item.status + (item.detail ? " · " + item.detail : "");
  const signature = item.action.ID + "\0" + message;
  if (node.dataset.signature === signature) return;
  node.dataset.signature = signature;
  node.dataset.actionState = item.status;
  node.replaceChildren();
  node.classList.toggle("text-critical", item.status === "Failed" || item.status === "Rejected");
  node.classList.toggle("text-attention", item.status === "Sending" || item.status === "Unknown");
  node.classList.toggle("text-muted", item.status === "Submitted" || item.status === "Processed");
  const text = el("span", "", message);
  text.setAttribute("role", "status");
  node.append(text);
  if (item.status !== "Sending") {
    node.append(
      button(
        "Dismiss",
        () => {
          item.dismissed = true;
          renderActionFeedback();
        },
        ui.small + " ml-2",
      ),
    );
  }
}
export function renderActionFeedback() {
  for (const item of outstanding.values()) {
    if (item.status === "Sending" && Date.now() >= item.until) {
      item.status = "Unknown";
      item.detail = "Outcome unknown; still waiting.";
    }
  }
  // Dynamic collections can disappear indefinitely. Keep just one last
  // unresolved result per screen, not an unbounded history of removed IDs.
  for (const [key, item] of latest) {
    if (item.control !== "lifecycle" && !c(item.control)) {
      latest.delete(key);
      retainRemoved(item);
    }
  }
  let summary = root.querySelector<HTMLElement>("[data-action-summary]");
  if (!summary) {
    summary = el("div", "mb-3 space-y-2");
    summary.dataset.actionSummary = "";
    summary.hidden = true;
    root.prepend(summary);
  }
  const displayed = new Set<string>();
  const slots = [
    ...document.querySelectorAll<HTMLElement>("#confirm[open][data-confirmation=host] [data-action-feedback]"),
    ...root.querySelectorAll<HTMLElement>("[data-action-feedback]"),
  ];
  for (const node of slots) {
    if (summary.contains(node)) continue;
    const id = node.dataset.actionFeedback || "";
    const item = latest.get(feedbackKey(appState.screen, id));
    render(node, displayed.has(id) || (item?.context || "") !== node.dataset.actionContext ? undefined : item);
    if (!node.hidden && node.checkVisibility()) displayed.add(id);
  }
  const fallback = [...latest.values()].filter(
    (item) => item.screen === appState.screen && !displayed.has(item.control) && !item.dismissed,
  );
  const vanished = removed.get(appState.screen);
  if (vanished && !vanished.dismissed) fallback.push(vanished);
  // Preserve the dismiss button while polls repeat, including its focus.
  const wanted = new Set(fallback.map((item) => item.action.ID));
  for (const node of summary.querySelectorAll<HTMLElement>("[data-action-id]")) {
    if (!wanted.has(node.dataset.actionId || "")) node.remove();
  }
  for (const item of fallback) {
    let node = summary.querySelector<HTMLElement>(`[data-action-id="${item.action.ID}"]`);
    if (!node) {
      node = actionFeedback(item.control, summary);
      node.dataset.actionId = item.action.ID;
    }
    render(node, item, true);
  }
  summary.hidden = !fallback.length;
}
