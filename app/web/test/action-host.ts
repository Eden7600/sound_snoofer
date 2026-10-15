import type { Gui } from "./host.ts";
import type { ActionResult } from "../src/types.ts";
declare global {
  interface Window {
    actionReplies: Record<string, (result: ActionResult) => void>;
  }
}

export async function holdActions(gui: Gui) {
  await gui.update(() => {
    window.actionReplies = {};
    const desktop = window.go?.app?.Desktop;
    if (!desktop) throw new Error("Desktop missing");
    desktop.Send = (action) => {
      window.sent.push(action);
      return new Promise((resolve) => {
        window.actionReplies[action.ID] = resolve;
      });
    };
  });
}
export async function reply(gui: Gui, id: string, status: ActionResult["Status"], detail = "") {
  await gui.update(
    ({ id, status, detail }) => {
      const action = window.sent.find((action) => action.Request?.ID === id);
      if (!action) throw new Error("Missing action " + id);
      window.actionReplies[action.ID]?.({ ID: action.ID, Status: status, Detail: detail });
    },
    { id, status, detail },
  );
}
