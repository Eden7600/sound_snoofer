import type { NodeResult } from "axe-core";

// Existing defects only. Exact targets prevent an exception from hiding new controls.
// No WCAG 2.1 A/AA violations were found in the initial expanded-screen scans.
export const knownA11y: { screen: string; rule: string; target: NodeResult["target"]; owner: string }[] = [];
