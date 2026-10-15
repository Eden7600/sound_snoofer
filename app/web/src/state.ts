import type { Control, ViewState, Pending, Widget } from "./types.ts";
import { $, button, ui, badge, tc } from "./dom.ts";
import { display, tone } from "./model.ts";
export const root = $("#content");
export const widgets: Widget[] = [];
export const updaters: Array<() => void> = [];
export const appState: {
  state: ViewState;
  animationVersion: number;
  controls: Map<string, Control>;
  screen: string;
  pending: Pending | null;
  reconnecting: boolean;
  localError: string;
  dismissedNotice: string;
} = {
  state: { Controls: [], Plugins: {}, PluginLabels: {}, Enabled: {}, Notice: "", Confirmation: "" },
  animationVersion: 0,
  controls: new Map(),
  screen: "audio",
  pending: null,
  reconnecting: false,
  localError: "",
  dismissedNotice: "",
};
export function c<T = unknown>(id: string): Control<T> | undefined {
  return appState.controls.get(id) as Control<T> | undefined;
}
export function setReconnecting(value: boolean) {
  appState.reconnecting = value;
  $("#connection").hidden = !value;
}
export function showError(error: unknown) {
  appState.localError = String(error);
  refreshNotice();
}
export function refreshNotice() {
  const message =
    appState.localError || (appState.state.Notice !== appState.dismissedNotice ? appState.state.Notice : "");
  const node = $("#notice");
  node.hidden = !message;
  node.replaceChildren();
  if (message) {
    node.append(
      button(
        "Dismiss",
        () => {
          appState.localError = "";
          appState.dismissedNotice = appState.state.Notice;
          refreshNotice();
        },
        ui.small + " float-right ml-3",
      ),
      document.createTextNode(message),
    );
  }
}
export function update() {
  if (
    appState.pending &&
    (Date.now() > appState.pending.until ||
      appState.state.Notice !== appState.pending.notice ||
      (appState.pending.id && c(appState.pending.id)?.Revision !== appState.pending.revision) ||
      (!appState.pending.id && appState.state.Confirmation))
  )
    appState.pending = null;
  for (const w of widgets) {
    const item = c(w.id) || w.initial;
    w.title.classList.toggle("text-muted", !!item.Subdued);
    w.note.textContent = item.Subdued && item.Status === "VR override" ? "" : item.Status || "";
    w.note.className = "mt-1 block text-[11px] break-words " + tc(tone(item));
    const input = w.input;
    if (item.Kind === "toggle") {
      input.textContent = display(item);
      input.setAttribute("aria-pressed", String(item.Value === "On"));
      input.dataset.tone = tone(item);
    } else if (item.Kind === "selection" || item.Kind === "text") {
      if (document.activeElement !== input && "value" in input) input.value = item.Value || "";
    } else if (item.Kind === "status") {
      input.textContent = display(item);
      input.className = "max-w-[65%] text-right break-words " + tc(tone(item));
    }
    if (["toggle", "selection", "text", "command"].includes(item.Kind) && "disabled" in input)
      input.disabled = !!appState.pending || !item.Available;
  }
  for (const fn of updaters) fn();
  // Audio health lives on the Audio and Diagnostics screens, not the shared header.
  $("#summary").replaceChildren();
  if (appState.pending) $("#summary").append(badge("Sending", "attention"));
  if (c("audio.normal-source")?.Subdued) $("#summary").append(badge("VR active", "active"));
  if (c("audio.pause-devices")?.Value === "On") $("#summary").append(badge("Devices paused", "attention"));
  if (c("audio.pause-sends")?.Value === "On") $("#summary").append(badge("Sends paused", "attention"));
  root.setAttribute("aria-busy", String(!!appState.pending));
  refreshNotice();
  const dialog = $<HTMLDialogElement>("#confirm");
  if (appState.state.Confirmation) {
    $("#confirm-text").textContent = appState.state.Confirmation;
    if (!dialog.open) dialog.showModal();
  } else if (dialog.open) dialog.close();
}
