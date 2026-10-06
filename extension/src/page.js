// Runs in the page's own world so it can see the page's media session
// handlers. It reports this frame's media to bridge.js and runs commands from
// it, both through window events carrying JSON strings.
(() => {
  if (window.__snooferMedia) return;
  window.__snooferMedia = true;
  const handlers = new Map();
  const session = navigator.mediaSession;
  if (session && typeof session.setActionHandler === "function") {
    // Remember the page's handlers (for example YouTube's nexttrack) so
    // Snoofer can call them like the browser's own media controls do.
    const original = session.setActionHandler.bind(session);
    session.setActionHandler = (action, handler) => {
      if (handler) handlers.set(action, handler);
      else handlers.delete(action);
      return original(action, handler);
    };
  }
  const elements = new Set();
  let lastPlayed = null;

  // active is the element Snoofer controls: one that is playing (the longest,
  // to prefer content over previews), else the last one that played.
  function active() {
    let best = null;
    for (const el of elements) {
      if (!el.isConnected) {
        elements.delete(el);
        continue;
      }
      if (!el.paused && !el.ended && (!best || (el.duration || 0) > (best.duration || 0))) best = el;
    }
    return best || (lastPlayed && lastPlayed.isConnected ? lastPlayed : null);
  }

  function artwork(metadata) {
    const list = (metadata && metadata.artwork) || [];
    let best = null;
    for (const item of list) {
      const size = parseInt(String(item.sizes || "").split("x")[0], 10) || 0;
      if (!best || size > best.size) best = { src: item.src, size };
    }
    if (!best) return "";
    try {
      return new URL(best.src, location.href).href;
    } catch {
      return "";
    }
  }

  function state() {
    const el = active();
    const metadata = session && session.metadata;
    if (!el && !metadata) return { present: false };
    let playing = el ? !el.paused && !el.ended : session.playbackState === "playing";
    if (!el && session.playbackState === "none") playing = false;
    const duration = el && Number.isFinite(el.duration) ? el.duration : 0;
    return {
      present: true,
      title: (metadata && metadata.title) || "",
      artist: (metadata && metadata.artist) || "",
      album: (metadata && metadata.album) || "",
      artwork: artwork(metadata),
      state: playing ? "playing" : "paused",
      positionMs: el ? Math.round(el.currentTime * 1000) : 0,
      durationMs: Math.round(duration * 1000),
      rate: el ? el.playbackRate || 1 : 1,
      at: Date.now(),
      canNext: handlers.has("nexttrack"),
      canPrev: handlers.has("previoustrack"),
      canSeek: handlers.has("seekto") || duration > 0,
    };
  }

  let timer = 0;
  let lastSent = 0;
  function report(soon) {
    if (timer) return;
    // Position ticks are throttled to once a second; state changes go quickly.
    const delay = soon ? 150 : Math.max(0, 1000 - (Date.now() - lastSent));
    timer = setTimeout(() => {
      timer = 0;
      lastSent = Date.now();
      window.dispatchEvent(new CustomEvent("snoofer-media:report", { detail: JSON.stringify(state()) }));
    }, delay);
  }

  for (const type of ["play", "playing", "pause", "ended", "durationchange", "loadedmetadata", "emptied", "seeked", "ratechange"]) {
    document.addEventListener(type, (e) => {
      if (!(e.target instanceof HTMLMediaElement)) return;
      elements.add(e.target);
      if (type === "play" || type === "playing") lastPlayed = e.target;
      report(true);
    }, true);
  }
  // Progress from an element not seen starting (it was playing before this
  // script ran, for example after installing the extension) adopts it.
  document.addEventListener("timeupdate", (e) => {
    if (!(e.target instanceof HTMLMediaElement)) return;
    if (!elements.has(e.target)) {
      elements.add(e.target);
      if (!e.target.paused) lastPlayed = e.target;
      report(true);
      return;
    }
    report(false);
  }, true);

  // Adopt media already in the document when injected into an open tab.
  for (const el of document.querySelectorAll("audio, video")) {
    elements.add(el);
    if (!el.paused && !el.ended) lastPlayed = el;
  }
  if (lastPlayed) report(true);

  function call(action, details) {
    const handler = handlers.get(action);
    if (!handler) return false;
    handler({ action, ...details });
    return true;
  }

  window.addEventListener("snoofer-media:command", (e) => {
    let command;
    try {
      command = JSON.parse(e.detail);
    } catch {
      return;
    }
    const el = active();
    const playing = el ? !el.paused && !el.ended : session && session.playbackState === "playing";
    switch (command.op) {
      case "toggle":
        if (playing) {
          if (!call("pause") && el) el.pause();
        } else if (!call("play") && el) {
          el.play().catch(() => {});
        }
        break;
      case "next":
        call("nexttrack");
        break;
      case "prev":
        call("previoustrack");
        break;
      case "seek": {
        const seconds = Math.max(0, (command.value || 0) / 1000);
        if (!call("seekto", { seekTime: seconds }) && el) el.currentTime = seconds;
        break;
      }
    }
    report(true);
  });
})();
