import test from "node:test";
import assert from "node:assert/strict";
import type { Control, ViewState } from "./types.ts";
import { screenAttention } from "./attention.ts";
import { pluginScreens } from "./plugin-lifecycle.ts";

function state(): ViewState {
  const PluginStates = Object.fromEntries(
    Object.values(pluginScreens).map(({ plugin }) => [plugin, "running" as const]),
  );
  const Controls = Object.values(pluginScreens).map(
    ({ publication }) => ({ ID: publication, ViewData: {}, Kind: "status" }) as Control,
  );
  return {
    Controls,
    PluginStates: { ...PluginStates, aec: "running" },
    Plugins: {},
    PluginLabels: {},
    Enabled: {},
    Notice: "",
    Confirmation: "",
    ConfigRevision: "fixture-config",
    ProposalID: "",
  };
}
function add(view: ViewState, control: Partial<Control>) {
  view.Controls!.push(control as Control);
}

test("canonical provider attention maps only to the screen that displays it", () => {
  for (const [ID, screen] of [
    ["audio.health", "audio"],
    ["aec.status", "audio"],
    ["audio.slot-1", "routing"],
    ["audio.slot-2", "routing"],
    ["audio.slot-3", "routing"],
    ["soundboard.status", "soundboard"],
    ["appaudio.status", "appaudio"],
    ["nowplaying.status", "media"],
    ["streamdeck.status", "deck"],
  ] as const) {
    const view = state();
    add(view, { ID, Attention: "critical" });
    assert.deepEqual(screenAttention(view), { [screen]: "critical" }, ID);
    view.PluginStates![ID.split(".")[0]!] = "disabled";
    assert.deepEqual(screenAttention(view), {}, ID + " disabled");
  }
});

test("lifecycle failures mark Plugins and gated screens, never working Audio for optional AEC", () => {
  const view = state();
  view.PluginStates!.audio = "failed";
  view.Plugins!.audio = "Running";
  assert.deepEqual(screenAttention(view), { plugins: "critical", audio: "critical", routing: "critical" });
  view.PluginStates!.audio = "running";
  view.PluginStates!.aec = "failed";
  assert.deepEqual(screenAttention(view), { plugins: "critical" });
});

test("typed current connection issues and setup use the finite destination map", () => {
  for (const [ID, owner] of [
    ["audio.app-engine", "audio"],
    ["aec.app-engine", "audio"],
    ["hue.app-bridge", "lights"],
    ["discord.app-client", "meetings"],
    ["soundboard.app-playback", "soundboard"],
    ["streamdeck.app-device", ""],
    ["custom.peer", ""],
  ] as const) {
    for (const condition of ["error", "attention", "required", "stale"]) {
      const view = state();
      const severity = ["error", "required"].includes(condition) ? "critical" : "attention";
      add(view, {
        ID,
        Kind: "connection",
        Connection: {
          State: condition === "required" ? "disconnected" : condition === "stale" ? "connected" : condition,
          Endpoint: "",
          Since: "",
          LastActivity: "2026-01-01T00:00:00Z",
          LastError: "Running",
          LastErrorAt: "",
          Details: [
            { Label: "Required", Value: "Yes" },
            { Label: "Interval", Value: "1s" },
          ],
        },
      });
      assert.deepEqual(
        screenAttention(view, Date.parse("2026-01-01T00:00:04Z")),
        { apps: severity, ...(owner ? { [owner]: severity } : {}) },
        ID + condition,
      );
    }
  }
  for (const [ID, owner] of [
    ["hue.app-bridge", "lights"],
    ["discord.app-client", "meetings"],
  ] as const) {
    const view = state();
    add(view, { ID, Kind: "connection", Connection: { State: "unconfigured" } as Control["Connection"] });
    assert.deepEqual(screenAttention(view), { apps: "attention", [owner]: "attention" });
  }
});

test("ordinary, historical and unknown connection states remain neutral", () => {
  for (const State of ["off", "unknown", "unconfigured", "disconnected", "connecting", "connected", "ready"]) {
    const view = state();
    add(view, {
      ID: "custom.peer",
      Kind: "connection",
      Value: "Error",
      Status: "Failed",
      Connection: { State, LastError: "failed earlier", Details: [] } as unknown as Control["Connection"],
    });
    assert.deepEqual(screenAttention(view), {}, State);
  }
});

test("muting, Off, preview, VR overrides, missing slots and arbitrary status prose do not create markers", () => {
  const view = state();
  for (const ID of [
    "audio.mic-mute",
    "audio.mic-stack",
    "audio.normal-source",
    "streamdeck.status",
    "soundboard.status",
    "nowplaying.status",
    "audio.slot-1",
  ])
    add(view, {
      ID,
      Value: "Off",
      Status: "Error muted Unsaved Preview",
      Subdued: true,
      ViewData: { Device: "disconnected", Bus: "" },
    });
  add(view, { ID: "custom.status", Attention: "critical" });
  add(view, { ID: "audio.health", Attention: "unknown" as Control["Attention"] });
  assert.deepEqual(screenAttention(view), {});
});

test("priority ambiguity counts only non-selected multi-matches and critical takes precedence", () => {
  const view = state(),
    priorities = view.Controls!.find((control) => control.ID === "audio.priorities")!;
  const entry = { InUse: false, Matches: ["A", "B"] };
  priorities.ViewData = { Lists: { playback: [entry] } };
  assert.deepEqual(screenAttention(view), { routing: "attention" });
  entry.InUse = true;
  assert.deepEqual(screenAttention(view), {});
  entry.InUse = false;
  entry.Matches = [];
  assert.deepEqual(screenAttention(view), {});
  entry.Matches = ["A", "B"];
  add(view, { ID: "audio.slot-1", Attention: "critical" });
  assert.deepEqual(screenAttention(view), { routing: "critical" });
  view.Controls = view.Controls!.filter((control) => control.ID !== "audio.priorities");
  assert.deepEqual(screenAttention(view), {});
});

test("disabled reports and hidden controls cannot manufacture attention", () => {
  const view = state();
  add(view, { ID: "hue.app-bridge", Kind: "connection", Connection: { State: "error" } as Control["Connection"] });
  view.PluginStates!.hue = "disabled";
  add(view, { ID: "audio.slot-1", Hidden: true, Attention: "attention" });
  assert.deepEqual(screenAttention(view), {});
  view.PluginStates!.hue = "running";
  view.Controls = view.Controls!.filter((control) => control.ID !== "hue.status");
  assert.deepEqual(screenAttention(view), { apps: "critical" });
});

test("audio critical severity wins until its source recovers", () => {
  const view = state();
  add(view, { ID: "audio.health", Attention: "attention" });
  const failure = { ID: "aec.status", Attention: "critical" as const };
  add(view, failure);
  assert.deepEqual(screenAttention(view), { audio: "critical" });
  view.Controls = view.Controls!.filter((control) => control.ID !== "aec.status");
  assert.deepEqual(screenAttention(view), { audio: "attention" });
});
