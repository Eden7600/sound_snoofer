// The GUI test fixture: audio controls generated from the audio provider
// (plugins/audio TestGUIFixture), everything else hand-written (manual.ts).
import fs from "node:fs";
import path from "node:path";
import { manualFixture } from "./manual";
import type { Animations, Control, Fixture } from "../types";

export type Variant = "live" | "preview";

export function studioFixture(variant: Variant = "live"): { fixture: Fixture; animations: Animations } {
  const { fixture, animations } = manualFixture();
  const generated = JSON.parse(fs.readFileSync(path.join(__dirname, `audio.${variant}.json`), "utf8")) as Control[];
  // Connection reports (audio.app-*) are still hand-written.
  const manual = fixture.Controls.filter((c) => !c.ID.startsWith("audio.") || c.ID.startsWith("audio.app-"));
  fixture.Controls = [...manual, ...generated];
  return { fixture, animations };
}
