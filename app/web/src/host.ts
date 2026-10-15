import type { Control, UIAction, ViewUpdate, ViewState } from "./types.ts";
import { appState, c, update } from "./state.ts";
import { beginAction, finishAction, uncertainAction } from "./actions.ts";
export let viewVersion = 0;
let artwork: Record<string, string> = {};

export function hydrateState(next: ViewUpdate | null): ViewState | null {
  if (!next) return null;
  const nextArtwork = next.Artwork ?? artwork;
  const state = {
    ...next.State,
    Controls: next.State.Controls?.map((control) => ({ ...control, Artwork: nextArtwork[control.ID] || "" })) ?? null,
  };
  artwork = nextArtwork;
  viewVersion = next.Version;
  appState.animationVersion = next.AnimationVersion;
  return state;
}

export async function send(input: Omit<UIAction, "ID">, rejection = "", context?: string) {
  const action = beginAction(input, rejection, context);
  update();
  if (!action) return;
  try {
    const desktop = window.go?.app?.Desktop;
    if (!desktop) throw new Error("Desktop connection unavailable");
    finishAction(action.ID, await desktop.Send(action));
  } catch (error) {
    uncertainAction(action.ID, error);
  }
  // A result settles its reservation, not the provider snapshot. Refresh busy
  // gates immediately without flickering committed inputs to the old readback.
  update(false);
}
export function request(
  control: Control | undefined,
  operation: string,
  value = "",
  delta = 0,
  revision = control?.Revision,
  context?: string,
) {
  if (!control) return;
  send(
    {
      Request: {
        ID: control.ID,
        Revision: revision ?? control.Revision,
        Operation: operation,
        Value: value,
        Delta: delta,
      },
    },
    !control.Available
      ? "This control is unavailable; action not sent."
      : !control.Operations?.includes(operation)
        ? "This operation is unavailable; action not sent."
        : "",
    context,
  );
}
export function press(id: string, context?: string) {
  const control = c(id);
  if (control) request(control, "press", "", 0, control.Revision, context);
  else
    send(
      { Request: { ID: id, Revision: 0, Operation: "press", Value: "", Delta: 0 } },
      "This control is no longer available; action not sent.",
      context,
    );
}
