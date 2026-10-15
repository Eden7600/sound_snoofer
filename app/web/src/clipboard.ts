import { el } from "./dom.ts";
import { showError } from "./state.ts";

export async function copyText(text: string, feedback: HTMLElement) {
  try {
    if (navigator.clipboard?.writeText) await navigator.clipboard.writeText(text);
    else throw new Error("Clipboard unavailable");
  } catch {
    // WebView clipboard permissions vary; fall back to a temporary selection.
    const area = el("textarea", "fixed opacity-0");
    area.value = text;
    area.setAttribute("readonly", "");
    document.body.append(area);
    area.select();
    const ok = document.execCommand("copy");
    area.remove();
    if (!ok) {
      showError("Copy failed: clipboard unavailable");
      return;
    }
  }
  feedback.textContent = "Copied";
  setTimeout(() => {
    feedback.textContent = "";
  }, 1500);
}
