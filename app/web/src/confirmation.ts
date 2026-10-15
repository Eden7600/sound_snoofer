import { $ } from "./dom.ts";
import { appState, c, root } from "./state.ts";
import { send } from "./host.ts";
import { restoreFocus } from "./view-state.ts";
import type { Request } from "./types.ts";
import { pluginGateState } from "./plugin-gate.ts";

interface LocalConfirmation {
  title: string;
  message: string;
  accept: string;
  request: Request;
  context: string;
  currentContext: () => string;
  screen: string;
  invoker: string | undefined;
  invalid: boolean;
}
type Confirmation = { kind: "host"; message: string } | { kind: "local"; ticket: LocalConfirmation };
let local: LocalConfirmation | null = null;
let shown: Confirmation["kind"] | undefined;

function valid(ticket: LocalConfirmation) {
  const control = c(ticket.request.ID);
  return (
    !ticket.invalid &&
    pluginGateState(ticket.screen) === "running" &&
    !appState.reconnecting &&
    appState.screen === ticket.screen &&
    ticket.currentContext() === ticket.context &&
    !!control?.Available &&
    control.Revision === ticket.request.Revision &&
    !!control.Operations?.includes(ticket.request.Operation)
  );
}
export function confirmAction(
  invoker: HTMLElement,
  id: string,
  title: string,
  message: string,
  accept: string,
  currentContext: () => string,
) {
  const control = c(id);
  if (
    local ||
    appState.state.Confirmation ||
    appState.pending ||
    appState.reconnecting ||
    !control?.Available ||
    !control.Operations?.includes("press")
  )
    return;
  local = {
    title,
    message,
    accept,
    request: { ID: id, Revision: control.Revision, Operation: "press", Value: "", Delta: 0 },
    context: currentContext(),
    currentContext,
    screen: appState.screen,
    invoker: invoker.dataset.viewKey,
    invalid: false,
  };
  renderConfirmation();
}
export function closeLocalConfirmation() {
  if (!local) return;
  const invoker = local.invoker;
  local = null;
  renderConfirmation();
  if (!appState.state.Confirmation) restoreFocus(root, invoker);
}
export function renderConfirmation() {
  if (appState.state.Confirmation) local = null;
  if (local && !valid(local)) local.invalid = true;
  const confirmation: Confirmation | null = appState.state.Confirmation
    ? { kind: "host", message: appState.state.Confirmation }
    : local
      ? { kind: "local", ticket: local }
      : null;
  const dialog = $<HTMLDialogElement>("#confirm");
  if (!confirmation) {
    if (dialog.open) dialog.close();
    shown = undefined;
    return;
  }
  const ticket = confirmation.kind === "local" ? confirmation.ticket : null;
  $("#confirm-title").textContent = ticket?.title || "Apply changes?";
  $("#confirm-text").textContent = ticket?.message || (confirmation.kind === "host" ? confirmation.message : "");
  const status = $("#confirm-status");
  status.hidden = !ticket?.invalid;
  status.textContent = ticket?.invalid ? "Selection changed; cancel and try again." : "";
  const yes = $<HTMLButtonElement>("#confirm-yes");
  yes.textContent = ticket?.accept || "Apply & restart";
  yes.disabled = !!ticket?.invalid || !!appState.pending || appState.reconnecting;
  const focusCancel = !dialog.open || shown !== confirmation.kind;
  if (!dialog.open) dialog.showModal();
  if (focusCancel) $("#confirm-no").focus();
  shown = confirmation.kind;
}
export function initializeConfirmation() {
  const cancel = () => {
    if (appState.state.Confirmation) send({ Kind: "cancel" });
    else closeLocalConfirmation();
  };
  $("#confirm-no").onclick = cancel;
  $("#confirm-yes").onclick = () => {
    if (appState.state.Confirmation) {
      send({ Kind: "confirm" });
      return;
    }
    const ticket = local;
    if (!ticket) return;
    if (!valid(ticket) || appState.pending) {
      renderConfirmation();
      return;
    }
    closeLocalConfirmation();
    send({ Request: ticket.request });
  };
  const dialog = $<HTMLDialogElement>("#confirm");
  dialog.oncancel = (event) => {
    event.preventDefault();
    cancel();
  };
  dialog.onclose = () => {
    if (!dialog.open) closeLocalConfirmation();
  };
}
