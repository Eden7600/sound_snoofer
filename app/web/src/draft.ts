type Field = HTMLInputElement | HTMLSelectElement;
interface Snapshot {
  value: string;
  revision: number | undefined;
  submitted: string | undefined;
  submissionContext: string | undefined;
  choiceIdentity: string | undefined;
  selection?: [number | null, number | null, "forward" | "backward" | "none" | null];
}
interface Options {
  observed?: () => string;
  revision: () => number | undefined;
  refresh?: () => void;
  submissionContext?: () => string;
  completed?: (submitted: string, context: string | undefined) => boolean;
  choiceIdentity?: (value: string) => string | undefined;
}
const bindings = new WeakMap<HTMLElement, ReturnType<typeof registerDraft>>();

// Only explicit Enter/Apply/Add editors register here; live controls keep their
// existing commit rules. State is owned by the current field, not a global cache.
export function registerDraft(input: Field, options: Options) {
  const choice = input instanceof HTMLSelectElement;
  const observed = options.observed || (() => "");
  const choiceIdentity = () =>
    input instanceof HTMLSelectElement && input.selectedIndex >= 0
      ? options.choiceIdentity
        ? options.choiceIdentity(input.value)
        : input.value
      : undefined;
  let dirty = false;
  let revision: number | undefined;
  let submitted: string | undefined;
  let submissionContext: string | undefined;
  const cancel = () => {
    dirty = false;
    revision = undefined;
    submitted = undefined;
    submissionContext = undefined;
    input.value = observed();
  };
  const sync = () => {
    if (submitted !== undefined && input.value === submitted && options.completed?.(submitted, submissionContext))
      cancel();
    if (!choice && dirty && input.value === observed()) {
      dirty = false;
      revision = undefined;
      submitted = undefined;
      submissionContext = undefined;
    }
    if (!choice && !dirty && document.activeElement !== input) input.value = observed();
  };
  input.addEventListener("focus", () => {
    if (!dirty) revision = options.revision();
  });
  const changed = () => {
    revision ??= options.revision();
    dirty = choice || input.value !== observed();
    options.refresh?.();
  };
  input.addEventListener("input", changed);
  if (choice) input.addEventListener("change", changed);
  const binding = {
    get revision() {
      return revision ?? options.revision();
    },
    sync,
    cancel,
    submit() {
      submitted = input.value;
      submissionContext = options.submissionContext?.();
    },
    capture(): Snapshot | undefined {
      sync();
      if (!dirty && !choice) return;
      return {
        value: input.value,
        revision,
        submitted,
        submissionContext,
        choiceIdentity: choiceIdentity(),
        ...(input instanceof HTMLInputElement
          ? { selection: [input.selectionStart, input.selectionEnd, input.selectionDirection] as Snapshot["selection"] }
          : {}),
      };
    },
    restore(snapshot: Snapshot) {
      revision = snapshot.revision;
      submitted = snapshot.submitted;
      submissionContext = snapshot.submissionContext;
      dirty = true;
      if (
        input instanceof HTMLSelectElement &&
        (snapshot.choiceIdentity === undefined ||
          ![...input.options].some((option) => option.value === snapshot.value && !option.disabled) ||
          (options.choiceIdentity ? options.choiceIdentity(snapshot.value) : snapshot.value) !==
            snapshot.choiceIdentity)
      ) {
        input.selectedIndex = -1;
        revision = undefined;
        submitted = undefined;
        submissionContext = undefined;
      } else input.value = snapshot.value;
      if (input instanceof HTMLInputElement && snapshot.selection) {
        const [start, end, direction] = snapshot.selection;
        if (start !== null && end !== null) input.setSelectionRange(start, end, direction || "none");
      }
      sync();
      options.refresh?.();
    },
  };
  bindings.set(input, binding);
  return binding;
}
export function syncDraft(input: HTMLElement) {
  const binding = bindings.get(input);
  binding?.sync();
  return !!binding;
}
export function captureDraft(input: HTMLElement) {
  return bindings.get(input)?.capture();
}
export function restoreDraft(input: HTMLElement, snapshot: Snapshot | undefined) {
  if (snapshot) bindings.get(input)?.restore(snapshot);
}
