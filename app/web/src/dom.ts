import { icon } from "./icons.ts";
export { icon };
// Utility groups reused across screens. They are plain Tailwind utility
// literals (no component classes), kept complete so the CSS build finds them.
export const ui = {
  nav: "flex items-center gap-[13px] border-transparent bg-transparent px-3 py-[11px] text-left whitespace-nowrap text-muted hover:bg-hover aria-[current=page]:border-[#214a5a] aria-[current=page]:bg-[#163341] aria-[current=page]:text-active max-[1120px]:gap-[9px] max-[1120px]:px-2 max-[850px]:whitespace-normal",
  panel: "min-w-0 rounded-xl border border-seam bg-panel p-5 max-[1120px]:p-4",
  panelHead: "mb-[18px] flex items-center justify-between gap-2.5",
  h2: "text-base font-semibold",
  h3: "mb-2.5 text-[13px] font-bold",
  grid: "grid grid-cols-2 gap-[18px] max-[1120px]:gap-3.5 max-[850px]:grid-cols-1",
  actions: "flex flex-wrap items-center gap-2",
  toolbar: "mb-[18px] flex flex-wrap items-center gap-2",
  empty: "rounded-xl border border-dashed border-line px-6 py-11 text-center text-muted",
  note: "-mt-1 mb-[15px] text-xs text-attention",
  small: "px-[9px] py-[5px]",
  primary: "border-transparent bg-active font-semibold text-[#06232b] hover:bg-active",
  danger: "text-critical",
  toggle:
    "min-w-[65px] aria-pressed:border-active-edge aria-pressed:bg-active-bg aria-pressed:text-active data-[tone=critical]:aria-pressed:border-critical-edge data-[tone=critical]:aria-pressed:bg-critical-bg data-[tone=critical]:aria-pressed:text-critical",
  strip: "rounded-[9px] border border-seam bg-sidebar p-[18px] max-[850px]:p-3",
  stripTitle: "mb-3 text-xs font-bold text-muted",
  readout: "text-[27px] tracking-[-.5px] tabular-nums max-[850px]:text-[23px]",
  meterLabel: "mt-[5px] text-[10px] text-muted tabular-nums",
  // Gradient stops follow the dBFS scale: green below -12, amber to -3, red above.
  meterGradient:
    "absolute inset-0 bg-[linear-gradient(90deg,var(--color-meter-green)_0%,var(--color-meter-green)_70%,var(--color-meter-amber)_80%,var(--color-meter-amber)_90%,var(--color-meter-red)_96%)]",
  gainButtons: "mt-3.5 flex gap-[7px] *:flex-1",
  clipGrid:
    "grid grid-cols-[repeat(auto-fill,minmax(145px,1fr))] gap-3.5 min-[1500px]:grid-cols-[repeat(auto-fill,minmax(170px,1fr))]",
  sceneGrid: "mt-[18px] grid grid-cols-[repeat(auto-fill,minmax(118px,1fr))] gap-2.5",
  clip: "relative flex min-h-[168px] flex-col items-start rounded-xl bg-panel p-4 text-left data-[state=playing]:border-active data-[state=playing]:bg-[#12303a] data-[state=wait]:border-attention data-[state=error]:border-critical",
  scene:
    "relative flex min-h-[112px] flex-col items-start rounded-xl bg-panel p-3 text-left data-[state=playing]:border-active data-[state=playing]:bg-[#12303a] data-[state=wait]:border-attention data-[state=error]:border-critical",
  clipArt: "mb-3 grid h-[70px] w-full place-items-center text-active",
  sceneArt: "mb-1.5 grid h-10 w-full place-items-center text-active",
  badge:
    "inline-block rounded-full border border-line px-[9px] py-1 text-[11px] text-muted data-[tone=active]:border-[#23505c] data-[tone=active]:bg-[#11313b] data-[tone=active]:text-active data-[tone=attention]:border-attention-edge data-[tone=attention]:text-attention data-[tone=critical]:border-critical-edge data-[tone=critical]:text-critical",
};
const toneText: Record<string, string> = {
  active: "text-active",
  attention: "text-attention",
  critical: "text-critical",
  muted: "text-muted",
};
export function tc(t: string) {
  return toneText[t] || "";
}

export function el<K extends keyof HTMLElementTagNameMap>(tag: K, className = "", text: string | number = "") {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text !== undefined && text !== "") node.textContent = String(text);
  return node;
}
export function button(text: string, handler: (event: MouseEvent) => void, className = "") {
  const node = el("button", className, text);
  node.type = "button";
  node.onclick = handler;
  return node;
}
export function panel(title: string, parent: HTMLElement, className = "") {
  const node = el("section", ui.panel + " " + className),
    head = el("div", ui.panelHead);
  head.dataset.part = "panel-head";
  head.append(el("h2", ui.h2, title));
  node.append(head);
  parent.append(node);
  return node;
}
export function badge(text: string, t = "") {
  const b = el("span", ui.badge, text);
  b.dataset.tone = t;
  return b;
}
export function withIcon<T extends HTMLElement>(node: T, name: string) {
  node.prepend(icon(name, "size-4 shrink-0"));
  node.classList.add("inline-flex", "items-center", "gap-1.5");
  return node;
}
export function empty(parent: HTMLElement, text: string) {
  parent.append(el("div", ui.empty, text));
}
// iconButton is an icon-only button; label is its accessible name and tooltip.
export function iconButton(name: string, label: string, handler: (event: MouseEvent) => void) {
  const b = button("", handler, "flex items-center justify-center");
  b.append(icon(name, "size-4 shrink-0"));
  b.setAttribute("aria-label", label);
  b.title = label;
  return b;
}

export function $<T extends Element = HTMLElement>(selector: string): T {
  const node = document.querySelector<T>(selector);
  if (!node) throw new Error("Missing GUI element: " + selector);
  return node;
}
