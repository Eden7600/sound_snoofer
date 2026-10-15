// Native closed-select navigation fires change while browsing, and Chromium's
// popup can fire it even on Escape. Only an explicit commitment may write.
export function commitSelect(select: HTMLSelectElement, apply: () => void, observed?: () => string) {
  let original = select.value;
  let browsing = false;
  let popup = false;
  let popupValue: string | undefined;
  let committing = false;
  const cancel = () => {
    select.value = observed?.() ?? original;
    popupValue = undefined;
    browsing = popup = false;
  };
  const commit = (previous = original) => {
    if (popupValue !== undefined) select.value = popupValue;
    popupValue = undefined;
    const changed = select.value !== previous;
    original = select.value;
    browsing = popup = false;
    committing = true;
    if (changed) apply();
    select.blur();
    committing = false;
  };
  select.addEventListener("focus", () => {
    original = select.value;
  });
  select.addEventListener("blur", () => {
    if (!committing) cancel();
  });
  select.addEventListener("pointerdown", () => {
    popup = true;
  });
  select.addEventListener("keydown", (event) => {
    // A native dismissal can swallow keyup; the change handler already cancelled its DOM value.
    if (popup && !select.matches(":open")) cancel();
    if (event.key === "Escape") {
      event.preventDefault();
      cancel();
      select.blur();
    } else if (event.key === "Tab") {
      cancel();
    } else if (event.key === "Enter" && browsing && !popup) {
      event.preventDefault();
      commit();
    } else if (
      event.key === " " ||
      event.key === "Enter" ||
      event.key === "F4" ||
      (event.altKey && ["ArrowDown", "ArrowUp"].includes(event.key))
    ) {
      popup = true;
    } else if (
      !event.altKey &&
      !event.ctrlKey &&
      !event.metaKey &&
      (event.key.length === 1 ||
        ["ArrowDown", "ArrowUp", "ArrowLeft", "ArrowRight", "Home", "End", "PageDown", "PageUp"].includes(event.key))
    ) {
      browsing = true;
    }
  });
  select.addEventListener("change", () => {
    if (popup) {
      // A closing confirmation click consumes this choice; dismissals need no later key event.
      popupValue = select.value;
      select.value = observed?.() ?? original;
    } else if (!browsing) commit(observed?.() ?? original);
  });
  select.addEventListener("click", () => {
    // The opening click keeps :open; the closing pointer/Enter click confirms.
    if (popup && !select.matches(":open")) commit();
  });
  select.addEventListener("keyup", (event) => {
    if (!popup || select.matches(":open")) return;
    if (event.key === "Escape" || (event.altKey && ["ArrowUp", "ArrowDown"].includes(event.key))) {
      cancel();
      select.blur();
    } else if (event.key === "Enter") commit();
    else if (event.key === " ") {
      // Space can extend native typeahead without opening the popup.
      if (popupValue !== undefined) select.value = popupValue;
      popupValue = undefined;
      popup = false;
      browsing = true;
    }
  });
}
