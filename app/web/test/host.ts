// The fake Desktop host and the `gui` test fixture. Each test opens its own
// page with its own fixture; the page routes http://snoofer.test/ to app/web.
import fs from "node:fs";
import path from "node:path";
import { test as base, expect, type Page } from "@playwright/test";
import { manualFixture } from "./fixtures/manual";
import type { Action, Animations, Fixture, Request } from "./types";

const repository = path.resolve(__dirname, "../../..");
const web = path.join(repository, "app/web");
const origin = "http://snoofer.test";

declare global {
  interface Window {
    fixture: Fixture;
    animations: Animations;
    sent: Action[];
    copied: string[];
    // Finds a fixture control by ID (page side).
    ctl: (id: string) => Fixture["Controls"][number];
  }
}

const types: Record<string, string> = {
  ".css": "text/css",
  ".js": "text/javascript",
  ".mjs": "text/javascript",
  ".html": "text/html",
  ".ico": "image/x-icon",
  ".svg": "image/svg+xml",
};

// installHost runs in the page before the GUI: it fakes window.go.app.Desktop.
// Send records each action and applies the few mutations the host would.
function installHost([fixture, animations]: [Fixture, Animations]): void {
  window.fixture = fixture;
  window.animations = animations;
  window.sent = [];
  window.copied = [];
  window.ctl = (id) => {
    const control = window.fixture.Controls.find((c) => c.ID === id);
    if (!control) throw new Error(`fixture has no ${id}`);
    return control;
  };
  Object.defineProperty(navigator, "clipboard", {
    value: {
      writeText: async (text: string) => {
        window.copied.push(text);
      },
    },
  });
  const desktop = {
    State: async () => {
      // Meter readings expire after 500 ms; keep the fixture's readings live.
      const next = structuredClone(window.fixture);
      for (const c of next.Controls) if (c.Meter) c.Meter.At = new Date().toISOString();
      return next;
    },
    Animation: async (id: string) => window.animations[id] ?? null,
    Send: async (action: Action) => {
      window.sent.push(action);
      const r = action.Request;
      if (!r) return;
      const target = window.fixture.Controls.find((c) => c.ID === r.ID);
      const preview = window.fixture.Controls.find((c) => c.ID === "streamdeck.preview");
      const view = preview?.ViewData;
      if (r.ID === "streamdeck.slot" && view) view.Selected = Number(r.Value.split(" ")[1]) - 1;
      if (r.ID.startsWith("streamdeck.region-") && view && preview) {
        const label = (source: string) => view.Collections.find((x: { ID: string }) => x.ID === source)?.Label || source;
        if (r.ID === "streamdeck.region-add") {
          const [first, last, source] = r.Value.split(",");
          const dials = Number(first) >= 36;
          view.Regions.push({
            Source: source,
            Label: label(source!),
            First: Number(first) - (dials ? 36 : 0),
            Last: Number(last) - (dials ? 36 : 0),
            Dials: dials,
          });
        }
        if (r.ID === "streamdeck.region-source") {
          const [i, source] = r.Value.split(",");
          Object.assign(view.Regions[Number(i)], { Source: source, Label: label(source!) });
        }
        if (r.ID === "streamdeck.region-clip") {
          const [i, clip] = r.Value.split(",");
          view.Regions[Number(i)].Clip = clip === "on";
        }
        if (r.ID === "streamdeck.region-remove") view.Regions.splice(Number(r.Value), 1);
        // The fixture's collections overflow unless every region is clipped.
        view.Sets = view.Regions.some((g: { Clip?: boolean }) => !g.Clip) ? 3 : 1;
        for (const k of view.Keys) k.Region = -1;
        for (const d of view.Dials) d.Region = -1;
        view.Regions.forEach((g: { Dials: boolean; First: number; Last: number }, n: number) => {
          if (g.Dials) {
            for (let d = Math.min(g.First, g.Last); d <= Math.max(g.First, g.Last); d++) view.Dials[d].Region = n;
            return;
          }
          const rows = [Math.floor(g.First / 9), Math.floor(g.Last / 9)].sort((a, b) => a - b);
          const cols = [g.First % 9, g.Last % 9].sort((a, b) => a - b);
          for (let row = rows[0]!; row <= rows[1]!; row++) {
            for (let col = cols[0]!; col <= cols[1]!; col++) view.Keys[row * 9 + col].Region = n;
          }
        });
        view.Dirty = true;
        preview.Revision++;
      }
      if (!target) return;
      if (r.Operation === "set") target.Value = r.Value;
      target.Revision++;
    },
  };
  Object.assign(window, { go: { app: { Desktop: desktop } } });
}

export class Gui {
  readonly errors: string[] = [];

  constructor(readonly page: Page) {
    page.on("pageerror", (error) => this.errors.push(String(error)));
  }

  // open loads the GUI with a fresh fixture, optionally edited first.
  async open(edit?: (fixture: Fixture, animations: Animations) => void): Promise<void> {
    const { fixture, animations } = manualFixture();
    edit?.(fixture, animations);
    await this.page.route(`${origin}/**`, async (route) => {
      const url = new URL(route.request().url());
      const file =
        url.pathname === "/logo.ico"
          ? path.join(repository, "app/tray.ico")
          : path.join(web, url.pathname === "/" ? "index.html" : decodeURIComponent(url.pathname));
      if (!file.startsWith(web + path.sep) && !file.endsWith("tray.ico")) return route.fulfill({ status: 403 });
      if (!fs.existsSync(file)) return route.fulfill({ status: 404 });
      return route.fulfill({ body: fs.readFileSync(file), contentType: types[path.extname(file)] ?? "application/octet-stream" });
    });
    await this.page.addInitScript(installHost, [fixture, animations] as [Fixture, Animations]);
    await this.page.goto(`${origin}/`);
    await expect(this.page.locator("#content")).not.toContainText("Connecting to Snoofer");
  }

  // screen opens a screen from the sidebar.
  async screen(name: string): Promise<void> {
    await this.page.getByRole("navigation", { name: "Workspace" }).getByRole("button", { name, exact: true }).click();
  }

  // update edits the page's fixture; the next poll delivers it.
  async update<T>(edit: (arg: T) => void, arg?: T): Promise<void> {
    await this.page.evaluate(edit, arg as T);
  }

  sent(): Promise<Action[]> {
    return this.page.evaluate(() => window.sent);
  }

  async clearSent(): Promise<void> {
    await this.page.evaluate(() => {
      window.sent.length = 0;
    });
  }

  // last returns the most recent request, waiting until one with the ID arrives when given.
  async last(id?: string): Promise<Request> {
    if (id) await this.page.waitForFunction((id) => window.sent.some((a) => a.Request?.ID === id), id);
    const action = await this.page.evaluate(() => window.sent.at(-1));
    if (!action?.Request) throw new Error("no request sent");
    return action.Request;
  }

  // sentValues decodes the JSON values of every request to a control.
  async sentValues(id: string): Promise<unknown[]> {
    const sent = await this.sent();
    return sent.filter((a) => a.Request?.ID === id).map((a) => JSON.parse(a.Request!.Value));
  }

  // settled waits until the GUI accepts input again after a send.
  async settled(): Promise<void> {
    await expect(this.page.getByText("Sending", { exact: true })).toBeHidden();
    await expect(this.page.locator("#content")).toHaveAttribute("aria-busy", "false");
  }

  async noHorizontalScroll(): Promise<void> {
    expect(await this.page.evaluate(() => document.documentElement.scrollWidth > innerWidth), "horizontal overflow").toBe(false);
  }
}

export const test = base.extend<{ gui: Gui }>({
  gui: async ({ page }, use) => {
    const gui = new Gui(page);
    await use(gui);
    expect(gui.errors, "page errors").toEqual([]);
  },
});

export { expect };
