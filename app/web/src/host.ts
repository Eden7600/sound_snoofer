import type { Control, UIAction } from "./types.ts";
import { appState, c, update, showError } from "./state.ts";
export async function send(action: UIAction) {
  if (appState.pending) return;
  appState.localError = "";
  appState.pending = {
    id: action.Request?.ID,
    revision: action.Request?.Revision,
    notice: appState.state.Notice,
    until: Date.now() + 2500,
  };
  update();
  try {
    const desktop = window.go?.app?.Desktop;
    if (!desktop) throw new Error("Desktop connection unavailable");
    await desktop.Send(action);
  } catch (error) {
    appState.pending = null;
    showError(error);
    update();
  }
}
export function request(
  control: Control | undefined,
  operation: string,
  value = "",
  delta = 0,
  revision = control?.Revision,
) {
  if (!control || !control.Available || !(control.Operations || []).includes(operation)) return;
  send({
    Request: {
      ID: control.ID,
      Revision: revision ?? control.Revision,
      Operation: operation,
      Value: value,
      Delta: delta,
    },
  });
}
export function press(id: string) {
  request(c(id), "press");
}
