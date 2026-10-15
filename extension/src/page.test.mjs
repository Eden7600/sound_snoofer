import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import vm from "node:vm";

const source = readFileSync(new URL("page.js", import.meta.url), "utf8");

// Sanitized shape from observations.md; fake clicks model mutually exclusive state.
function page({ origin = "https://music.youtube.com", top = true, marker, paused = true } = {}) {
  const events = new Map(), mediaEvents = new Map(), timers = [], intervals = [], reports = [];
  class Media {
    isConnected = true; paused = paused; ended = false; duration = 20; currentTime = 2; plays = 0; pauses = 0;
    play() { this.plays++; this.paused = false; return Promise.resolve(); }
    pause() { this.pauses++; this.paused = true; }
  }
  const video = new Media();
  const button = () => ({ disabled: false, pressed: "false", clicks: 0, ariaDisabled: null,
    matches() { return false; }, getAttribute(name) { return name === "aria-pressed" ? this.pressed : this.ariaDisabled; } });
  const like = button(), dislike = button();
  const renderer = { tagName: "YTMUSIC-LIKE-BUTTON-RENDERER", status: "LIKE",
    data: { target: { videoId: "track-a" }, likeStatus: "LIKE", likesAllowed: true },
    getAttribute() { return this.status; },
    querySelectorAll(selector) { return selector.startsWith(".like-button") ? h.likes : h.dislikes; } };
  const player = { id: "track-a", state: 2, ad: false, getVideoData() { return { video_id: this.id }; },
    getPlayerState() { return this.state; }, classList: { contains() { return player.ad; } }, querySelectorAll() { return h.videos; } };
  const session = { metadata: { title: "Same title", artist: "Same artist" }, setActionHandler() {} };
  const window = { __snooferMedia: marker, addEventListener(name, fn) { const all = events.get(name) || []; all.push(fn); events.set(name, all); },
    dispatchEvent(event) { for (const fn of events.get(event.type) || []) fn(event); } };
  window.top = top ? window : {};
  window.addEventListener("snoofer-media:report-v3", (e) => reports.push(JSON.parse(e.detail)));
  const h = { player, renderer, video, like, dislike, likes: [like], dislikes: [dislike], videos: [video], players: [player], renderers: [renderer], modals: [], reports, session, window };
  const context = vm.createContext({ navigator: { mediaSession: session }, location: { origin, href: origin + "/browse" }, window, URL,
    document: { addEventListener(name, fn) { mediaEvents.set(name, fn); }, querySelectorAll(selector) {
      if (selector === "audio, video" || selector === "#movie_player video") return h.videos;
      if (selector === "#movie_player") return h.players;
      if (selector === "ytmusic-player-bar #like-button-renderer") return h.renderers;
      return h.modals;
    } }, HTMLMediaElement: Media,
    setTimeout(fn) { timers.push(fn); return timers.length; }, setInterval(fn) { intervals.push(fn); },
    CustomEvent: class { constructor(type, options) { this.type = type; this.detail = options.detail; } } });
  h.status = (value) => {
    renderer.status = renderer.data.likeStatus = value;
    like.pressed = String(value === "LIKE"); dislike.pressed = String(value === "DISLIKE");
  };
  like.click = () => { like.clicks++; h.status(renderer.status === "LIKE" ? "INDIFFERENT" : "LIKE"); };
  dislike.click = () => { dislike.clicks++; h.status(renderer.status === "DISLIKE" ? "INDIFFERENT" : "DISLIKE"); };
  h.status("LIKE");
  h.install = () => vm.runInContext(source, context);
  h.flush = () => { while (timers.length) timers.shift()(); return reports.at(-1); };
  h.tick = () => { intervals.forEach((fn) => fn()); return h.flush(); };
  h.command = (command) => window.dispatchEvent({ type: "snoofer-media:command-v3", detail: JSON.stringify(command) });
  h.media = (type) => { mediaEvents.get(type)?.({ target: video }); h.flush(); };
  h.install(); h.flush();
  return h;
}

test("fresh helpers retain captured transport intent and inject only once", () => {
  const h = page({ origin: "https://example.com", paused: false });
  let next = 0;
  h.session.setActionHandler("nexttrack", () => next++);
  h.install(); h.command({ op: "next" }); assert.equal(next, 1);
  h.command({ op: "toggle", value: 1 }); h.video.paused = true;
  h.command({ op: "toggle", value: 0 }); h.command({ op: "toggle" });
  assert.equal(h.video.plays, 1); assert.equal(h.video.pauses, 1);
  assert.equal(h.window.__snooferMedia, 3); assert.equal(h.flush().rating, undefined);
});

test("old helper upgrade requires reload and cannot duplicate old transport", () => {
  const h = page({ marker: 2 }); let oldCalls = 0;
  h.window.addEventListener("snoofer-media:command-v2", () => oldCalls++);
  const state = h.tick();
  assert.equal(state.reloadRequired, true); assert.equal(state.canRate, false); assert.equal(state.canSeek, false);
  h.command({ op: "next" }); h.command({ op: "toggle", value: 1 }); h.command({ op: "rate", track: "1:track-a", rating: "none" });
  assert.equal(oldCalls, 0); assert.equal(h.video.plays, 0); assert.equal(h.like.clicks, 0);
  const reloaded = page(); assert.equal(reloaded.tick().reloadRequired, undefined); assert.equal(reloaded.tick().canRate, true);
});

test("already-paused player reports its rating and observes manual changes", () => {
  const h = page(); const first = h.tick();
  assert.equal(first.state, "paused"); assert.equal(first.rating, "like"); assert.equal(first.canRate, true); assert.ok(first.track.length <= 128);
  h.status("INDIFFERENT"); assert.equal(h.tick().rating, "none");
  h.status("DISLIKE"); const after = h.tick();
  assert.equal(after.rating, "dislike"); assert.equal(after.track, first.track); assert.equal(h.video.plays, 0);
});

test("a quiet YouTube Music landing page does not adopt a blank paused video", () => {
  const h = page();
  h.session.metadata = null; h.player.id = ""; h.player.state = -1;
  assert.equal(h.tick().present, false);
  h.player.id = "track-a"; h.player.state = 2;
  assert.equal(h.tick().present, false, "unseen paused media needs metadata before adoption");
});

test("stable paused state keeps reporting for content-port recovery without clicks", () => {
  const h = page(); const track = h.tick().track; const before = h.reports.length;
  h.tick(); h.tick();
  assert.equal(h.reports.length, before + 2);
  assert.equal(h.reports.at(-1).track, track);
  assert.equal(h.like.clicks + h.dislike.clicks + h.video.plays, 0);
});

test("explicit desired state sets, clears and switches once without optimistic reports", () => {
  const h = page(); const track = h.tick().track;
  h.command({ op: "rate", track, rating: "like" }); assert.equal(h.like.clicks, 0);
  h.command({ op: "rate", track, rating: "none" }); assert.equal(h.flush().rating, "none");
  h.command({ op: "rate", track, rating: "dislike" }); assert.equal(h.flush().rating, "dislike");
  h.command({ op: "rate", track, rating: "like" }); assert.equal(h.flush().rating, "like");
  assert.equal(h.like.clicks, 2); assert.equal(h.dislike.clicks, 1);
  h.dislike.click = () => { h.dislike.clicks++; };
  h.command({ op: "rate", track, rating: "dislike" }); assert.equal(h.flush().rating, "like");
  h.tick(); h.tick(); assert.equal(h.dislike.clicks, 2, "no retry when website never confirms");
});

test("same-title replacement and observed A-B-A reject old intent", () => {
  const h = page(); const first = h.tick().track;
  h.player.id = h.renderer.data.target.videoId = "track-b";
  h.command({ op: "rate", track: first, rating: "none" }); assert.equal(h.like.clicks, 0, "rechecks before polling");
  const second = h.tick().track; assert.notEqual(second, first);
  h.player.id = h.renderer.data.target.videoId = "track-a";
  const third = h.tick().track; assert.notEqual(third, first);
  h.command({ op: "rate", track: first, rating: "none" }); assert.equal(h.like.clicks, 0);
  h.media("emptied"); assert.notEqual(h.tick().track, third);
});

test("identity loss and player replacement retire the prior track token", () => {
  const h = page(); const first = h.tick().track;
  h.renderer.data.target.videoId = "different"; assert.equal(h.tick().canRate, false);
  h.renderer.data.target.videoId = h.player.id; const recovered = h.tick().track; assert.notEqual(recovered, first);
  h.players = [{ ...h.player }]; assert.notEqual(h.tick().track, recovered);
});

test("ambiguous, disabled and unreadable states are unknown and never click", () => {
  const changes = [
    (h) => h.likes.push(h.like), (h) => h.dislikes.push(h.dislike), (h) => h.players.push(h.player),
    (h) => { h.renderer.data.likeStatus = "INDIFFERENT"; }, (h) => { h.dislike.pressed = "true"; },
    (h) => { h.like.pressed = "TRUE"; }, (h) => { h.renderer.data.likesAllowed = false; },
    (h) => { h.like.disabled = true; }, (h) => { h.dislike.ariaDisabled = "true"; },
    (h) => { h.player.ad = true; }, (h) => { h.player.state = 3; },
    (h) => { h.player.getVideoData = () => { throw new Error("page updating"); }; },
    (h) => { h.renderer.tagName = "OTHER-RENDERER"; }, (h) => h.modals.push({ getClientRects: () => [1] }),
    (h) => { h.player.id = ""; }, (h) => { h.player.state = -1; },
  ];
  for (const change of changes) {
    const h = page(); const track = h.tick().track; change(h); h.command({ op: "rate", track, rating: "none" });
    const report = h.tick(); assert.equal(report.rating, "unknown"); assert.equal(report.canRate, false); assert.equal(h.like.clicks + h.dislike.clicks, 0);
  }
});

test("unsupported origin, embedded frame and malformed commands never rate", () => {
  for (const options of [{ origin: "https://example.com" }, { top: false }]) {
    const h = page(options); assert.equal(h.flush().rating, undefined);
    h.command({ op: "rate", track: "1:track-a", rating: "none" }); assert.equal(h.like.clicks, 0);
  }
  const h = page(); const track = h.tick().track;
  for (const command of [null, { op: "rate", track, value: 0 }, { op: "rate", track, rating: "unknown" }, { op: "rate", track: "x".repeat(129), rating: "none" }]) h.command(command);
  assert.equal(h.like.clicks, 0);
});
