// The fake Desktop host and the `gui` test fixture. Each test opens its own
// page with its own fixture; the page routes http://snoofer.test/ to app/web.
import fs from "node:fs";
import path from "node:path";
import { test as base, expect, type Page } from "@playwright/test";
import { studioFixture, type Variant } from "./fixtures/index.ts";
import type { DeckView } from "../src/screens/deck-types.ts";
import type { Action, Animations, Control, Fixture, Request } from "./types.ts";

const repository = path.resolve(__dirname, "../../..");
const web = path.join(repository, "app/web");
const origin = "http://snoofer.test";

declare global {
  interface Window {
    fixture: Fixture;
    animations: Animations;
    sent: Action[];
    copied: string[];
    liveMeters: boolean;
    transport: { polls: number; unchanged: number; artwork: number; animations: number };
    // Finds a fixture control by ID (page side).
    ctl: <T = unknown>(id: string) => Control<T>;
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
  window.liveMeters = true;
  window.transport = { polls: 0, unchanged: 0, artwork: 0, animations: 0 };
  window.ctl = <T = unknown>(id: string): Control<T> => {
    const control = window.fixture.Controls.find((c) => c.ID === id);
    if (!control) throw new Error(`fixture has no ${id}`);
    return control as Control<T>;
  };
  Object.defineProperty(navigator, "clipboard", {
    value: {
      writeText: async (text: string) => {
        window.copied.push(text);
      },
    },
  });
  let version = 0,
    artworkVersion = 0,
    animationVersion = 0;
  let previousState = "",
    previousArtwork = "",
    previousAnimations = "";
  const desktop = {
    State: async (sinceVersion: number) => {
      window.transport.polls++;
      // Meter readings expire after 500 ms; keep the fixture's readings live.
      const next = structuredClone(window.fixture);
      const artwork: Record<string, string> = {};
      const animations: Animations = {};
      for (const c of next.Controls) {
        if (window.liveMeters && c.Meter) c.Meter.At = new Date().toISOString();
        if (c.Artwork) artwork[c.ID] = c.Artwork;
        c.Artwork = "";
        const frames = window.animations[c.ID];
        if (frames && frames.length > 1) animations[c.ID] = frames;
      }
      const stateJSON = JSON.stringify(next),
        artworkJSON = JSON.stringify(artwork),
        animationJSON = JSON.stringify(animations);
      if (stateJSON !== previousState || artworkJSON !== previousArtwork || animationJSON !== previousAnimations) {
        version++;
        if (artworkJSON !== previousArtwork) artworkVersion = version;
        if (animationJSON !== previousAnimations) animationVersion = version;
        previousState = stateJSON;
        previousArtwork = artworkJSON;
        previousAnimations = animationJSON;
      }
      if (sinceVersion === version) {
        window.transport.unchanged++;
        return null;
      }
      const includeArtwork = sinceVersion === 0 || sinceVersion < artworkVersion || sinceVersion > version;
      if (includeArtwork) window.transport.artwork++;
      return {
        Version: version,
        State: next,
        AnimationVersion: animationVersion,
        ...(includeArtwork ? { Artwork: artwork } : {}),
      };
    },
    Animation: async (id: string) => {
      window.transport.animations++;
      return structuredClone(window.animations[id] ?? null);
    },
    Send: async (action: Action) => {
      window.sent.push(action);
      const r = action.Request;
      if (!r) return;
      const target = window.fixture.Controls.find((c) => c.ID === r.ID);
      const preview = window.fixture.Controls.find((c) => c.ID === "streamdeck.preview");
      const view = preview?.ViewData as DeckView | undefined;
      if (r.ID === "streamdeck.slot" && view) {
        const [kind, number] = r.Value.split(" ");
        view.Selected = Number(number) - 1 + (kind === "Dial" ? view.Keys.length : 0);
      }
      // shortcut: region math assumes + XL, use view geometry when adding other-model region edit scenarios.
      if (r.ID.startsWith("streamdeck.region-") && view && preview) {
        const regions = (view.Regions ??= []);
        const label = (source: string) => view.Collections?.find((x) => x.ID === source)?.Label || source;
        if (r.ID === "streamdeck.region-add") {
          const [first, last, source] = r.Value.split(",");
          if (source === undefined) throw new Error("region-add requires a source");
          const dials = Number(first) >= 36;
          regions.push({
            Source: source,
            Label: label(source),
            Stacked: "",
            First: Number(first) - (dials ? 36 : 0),
            Last: Number(last) - (dials ? 36 : 0),
            Dials: dials,
            Legacy: false,
            Clip: false,
          });
        }
        if (r.ID === "streamdeck.region-source") {
          const [i, source] = r.Value.split(",");
          const region = regions[Number(i)];
          if (!region || source === undefined) throw new Error("region-source requires a region and source");
          Object.assign(region, { Source: source, Label: label(source) });
        }
        if (r.ID === "streamdeck.region-clip") {
          const [i, clip] = r.Value.split(",");
          const region = regions[Number(i)];
          if (!region) throw new Error("region-clip requires a region");
          region.Clip = clip === "on";
        }
        if (r.ID === "streamdeck.region-remove") regions.splice(Number(r.Value), 1);
        // The fixture's collections overflow unless every region is clipped.
        view.Sets = regions.some((g) => !g.Clip) ? 3 : 1;
        for (const k of view.Keys) k.Region = -1;
        for (const d of view.Dials) d.Region = -1;
        regions.forEach((g, n) => {
          if (g.Dials) {
            for (let d = Math.min(g.First, g.Last); d <= Math.max(g.First, g.Last); d++) {
              const dial = view.Dials[d];
              if (!dial) throw new Error("region references absent dial " + d);
              dial.Region = n;
            }
            return;
          }
          const firstRow = Math.floor(g.First / 9),
            lastRow = Math.floor(g.Last / 9);
          const firstCol = g.First % 9,
            lastCol = g.Last % 9;
          for (let row = Math.min(firstRow, lastRow); row <= Math.max(firstRow, lastRow); row++) {
            for (let col = Math.min(firstCol, lastCol); col <= Math.max(firstCol, lastCol); col++) {
              const key = view.Keys[row * 9 + col];
              if (!key) throw new Error("region references absent key " + (row * 9 + col));
              key.Region = n;
            }
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
  // update edits the page's fixture; the next poll delivers it.
  readonly update: Page["evaluate"];

  constructor(readonly page: Page) {
    this.update = page.evaluate.bind(page);
    page.on("pageerror", (error) => this.errors.push(String(error)));
  }

  // open loads the GUI with a fresh fixture, optionally edited first.
  async open(edit?: (fixture: Fixture, animations: Animations) => void, variant: Variant = "live"): Promise<void> {
    const { fixture, animations } = studioFixture(variant);
    edit?.(fixture, animations);
    await this.page.route(`${origin}/**`, async (route) => {
      const url = new URL(route.request().url());
      const file =
        url.pathname === "/logo.ico"
          ? path.join(repository, "app/tray.ico")
          : path.join(web, url.pathname === "/" ? "index.html" : decodeURIComponent(url.pathname));
      if (!file.startsWith(web + path.sep) && !file.endsWith("tray.ico")) return route.fulfill({ status: 403 });
      if (!fs.existsSync(file)) return route.fulfill({ status: 404 });
      return route.fulfill({
        body: fs.readFileSync(file),
        contentType: types[path.extname(file)] ?? "application/octet-stream",
      });
    });
    await this.page.addInitScript(installHost, [fixture, animations] as [Fixture, Animations]);
    await this.page.goto(`${origin}/`);
    await expect(this.page.locator("#content")).not.toContainText("Connecting to Snoofer");
  }

  // screen opens a screen from the sidebar.
  async screen(name: string): Promise<void> {
    await this.page.getByRole("navigation", { name: "Workspace" }).getByRole("button", { name, exact: true }).click();
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
    return sent.flatMap((a) => (a.Request?.ID === id ? [JSON.parse(a.Request.Value) as unknown] : []));
  }

  // settled waits until the GUI accepts input again after a send.
  async settled(): Promise<void> {
    await expect(this.page.getByText("Sending", { exact: true })).toBeHidden();
    await expect(this.page.locator("#content")).toHaveAttribute("aria-busy", "false");
  }

  async noHorizontalScroll(): Promise<void> {
    expect(
      await this.page.evaluate(() => document.documentElement.scrollWidth > innerWidth),
      "horizontal overflow",
    ).toBe(false);
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
