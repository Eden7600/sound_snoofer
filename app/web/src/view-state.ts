import { captureDraft, restoreDraft } from "./draft.ts";
// Only explicitly keyed presentation state survives an automatic screen rebuild.
const searches = new WeakMap<HTMLInputElement, () => void>();
export function viewKey<T extends HTMLElement>(node: T, ...parts: Array<string | number>): T {
  node.dataset.viewKey = JSON.stringify(parts);
  return node;
}
export function preserveSearch(input: HTMLInputElement, key: Array<string | number>, refresh: () => void) {
  viewKey(input, ...key);
  searches.set(input, refresh);
}
function keyed(root: HTMLElement) {
  const nodes = new Map<string, HTMLElement>();
  for (const node of root.querySelectorAll<HTMLElement>("[data-view-key]")) {
    const key = node.dataset.viewKey;
    if (!key) continue;
    if (nodes.has(key)) throw new Error("Duplicate view key: " + key);
    nodes.set(key, node);
  }
  return nodes;
}
export function captureView(root: HTMLElement) {
  const focused = document.activeElement;
  return {
    focused: focused instanceof HTMLElement ? focused.dataset.viewKey : undefined,
    focusInside: !!focused && root.contains(focused),
    x: window.scrollX,
    y: window.scrollY,
    nodes: new Map(
      [...keyed(root)].map(([key, node]) => [
        key,
        {
          open: node instanceof HTMLDetailsElement ? node.open : undefined,
          query: node instanceof HTMLInputElement && searches.has(node) ? node.value : undefined,
          draft: captureDraft(node),
          left: node.scrollLeft,
          top: node.scrollTop,
        },
      ]),
    ),
  };
}
export function restoreView(root: HTMLElement, saved: ReturnType<typeof captureView>) {
  const nodes = keyed(root);
  for (const [key, state] of saved.nodes) {
    const node = nodes.get(key);
    if (!node) continue;
    if (node instanceof HTMLDetailsElement && state.open !== undefined) node.open = state.open;
    if (node instanceof HTMLInputElement && state.query !== undefined) {
      node.value = state.query;
      searches.get(node)?.();
    }
    restoreDraft(node, state.draft);
  }
  // A modal opened by update owns focus; it lives outside the rebuilt content.
  if (saved.focusInside && !document.querySelector("dialog[open]")) {
    const node = saved.focused ? nodes.get(saved.focused) : undefined;
    if (node && !node.matches(":disabled") && node.checkVisibility({ visibilityProperty: true }))
      node.focus({ preventScroll: true });
    else {
      root.tabIndex = -1;
      root.focus({ preventScroll: true });
    }
  }
  for (const [key, state] of saved.nodes) {
    const node = nodes.get(key);
    if (node) {
      node.scrollLeft = state.left;
      node.scrollTop = state.top;
      // Chromium scrolls a rebuilt native listbox to its selection during its
      // first rendering frame. Restore after that frame's native layout.
      if (node instanceof HTMLSelectElement && node.size > 1)
        requestAnimationFrame(() => {
          if (!node.isConnected) return;
          requestAnimationFrame(() => {
            if (!node.isConnected) return;
            node.scrollLeft = state.left;
            node.scrollTop = state.top;
          });
        });
    }
  }
  window.scrollTo(saved.x, saved.y);
}
