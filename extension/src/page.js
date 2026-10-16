// Runs in the page's own world so it can see the page's media session
// handlers. It reports this frame's media to bridge.js and runs commands from
// it, both through window events carrying JSON strings.
(() => {
  if (window.__snooferMedia === 3) return;
  const reloadRequired = !!window.__snooferMedia;
  window.__snooferMedia = 3;
  const youtubeMusic = location.origin === "https://music.youtube.com" && window.top === window;
  const handlers = new Map();
  const session = navigator.mediaSession;
  if (!reloadRequired && session && typeof session.setActionHandler === "function") {
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
    if (best || (lastPlayed && lastPlayed.isConnected)) return best || lastPlayed;
    // A paused YouTube Music player can predate injection without a play event.
    if (youtubeMusic && session?.metadata) {
      try {
        const players = document.querySelectorAll("#movie_player");
        const player = players.length === 1 ? players[0] : null;
        const id = player?.getVideoData?.().video_id;
        const videos = player?.querySelectorAll("video") || [];
        if (typeof id === "string" && /^[A-Za-z0-9_-]{1,96}$/.test(id) && [0, 1, 2].includes(player.getPlayerState?.()) &&
            !player.classList.contains("ad-showing") && videos.length === 1 && videos[0].isConnected) return videos[0];
      } catch { /* A player being replaced is not an adoptable paused session. */ }
    }
    return null;
  }

  let playingIdentity = null;
  let trackLifetime = 0n;
  function ratingState() {
    if (!youtubeMusic) return { wire: {} };
    const unknown = { rating: "unknown", canRate: false };
    try {
      const players = document.querySelectorAll("#movie_player");
      const renderers = document.querySelectorAll("ytmusic-player-bar #like-button-renderer");
      const player = players.length === 1 ? players[0] : null;
      const renderer = renderers.length === 1 ? renderers[0] : null;
      const videos = player ? player.querySelectorAll("video") : [];
      const el = active();
      const id = player?.getVideoData?.().video_id;
      if (reloadRequired || !player || !renderer || renderer.tagName !== "YTMUSIC-LIKE-BUTTON-RENDERER" ||
          videos.length !== 1 || videos[0] !== el || !el?.isConnected ||
          typeof id !== "string" || !/^[A-Za-z0-9_-]{1,96}$/.test(id) || renderer.data?.target?.videoId !== id ||
          ![0, 1, 2].includes(player.getPlayerState?.()) || player.classList.contains("ad-showing")) {
        playingIdentity = null;
        return { wire: unknown };
      }
      if (!playingIdentity || playingIdentity.player !== player || playingIdentity.el !== el || playingIdentity.id !== id) {
        playingIdentity = { player, el, id, track: (++trackLifetime).toString(36) + ":" + id };
      }
      const wire = { ...unknown, track: playingIdentity.track };
      const likes = renderer.querySelectorAll(".like-button-wrapper button[aria-pressed]");
      const dislikes = renderer.querySelectorAll(":scope > yt-button-shape > button[aria-pressed]");
      const modal = [...document.querySelectorAll('dialog[open], [role="dialog"][aria-modal="true"]')].some((el) => el.getClientRects().length);
      if (modal || renderer.data.likesAllowed !== true || likes.length !== 1 || dislikes.length !== 1) return { wire };
      const like = likes[0], dislike = dislikes[0];
      if ([like, dislike].some((button) => button.disabled || button.matches(":disabled") || button.getAttribute("aria-disabled") === "true")) return { wire };
      const status = renderer.getAttribute("like-status");
      const expected = { LIKE: ["like", "true", "false"], INDIFFERENT: ["none", "false", "false"], DISLIKE: ["dislike", "false", "true"] }[status];
      // The auxiliary data.likeStatus may lag the live controls after a click.
      if (!expected || like.getAttribute("aria-pressed") !== expected[1] || dislike.getAttribute("aria-pressed") !== expected[2]) return { wire };
      return { wire: { ...wire, rating: expected[0], canRate: true }, like, dislike };
    } catch {
      // Page-owned getters and transient replacements may be unreadable.
      playingIdentity = null;
      return { wire: unknown };
    }
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
      canNext: !reloadRequired && handlers.has("nexttrack"),
      canPrev: !reloadRequired && handlers.has("previoustrack"),
      canSeek: !reloadRequired && (handlers.has("seekto") || duration > 0),
      ...(reloadRequired ? { reloadRequired: true } : {}),
      ...ratingState().wire,
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
      window.dispatchEvent(new CustomEvent("snoofer-media:report-v3", { detail: JSON.stringify(state()) }));
    }, delay);
  }

  for (const type of ["play", "playing", "pause", "ended", "durationchange", "loadedmetadata", "emptied", "seeked", "ratechange"]) {
    document.addEventListener(type, (e) => {
      if (!(e.target instanceof HTMLMediaElement)) return;
      elements.add(e.target);
      if (type === "emptied") playingIdentity = null;
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
  if (lastPlayed || session?.metadata) report(true);
  if (youtubeMusic) {
    let lastRating = "";
    // Bounded polling includes paused data-only changes and player replacement.
    setInterval(() => {
      const current = JSON.stringify(ratingState().wire);
      if (current !== lastRating) { lastRating = current; report(true); }
      // The bridge reconnects on a fresh report even if a paused rating is stable.
      else if (active() || session?.metadata) report(false);
    }, 500);
  }

  function call(action, details) {
    const handler = handlers.get(action);
    if (!handler) return false;
    handler({ action, ...details });
    return true;
  }

  window.addEventListener("snoofer-media:command-v3", (e) => {
    let command;
    try {
      command = JSON.parse(e.detail);
    } catch {
      return;
    }
    if (!command || reloadRequired) return;
    const el = active();
    switch (command.op) {
      case "rate": {
        if (typeof command.track !== "string" || !command.track || command.track.length > 128 || !["none", "like", "dislike"].includes(command.rating)) return;
        const current = ratingState();
        if (!current.wire.canRate || current.wire.track !== command.track) return;
        if (current.wire.rating === command.rating) break;
        const desired = command.rating === "none" ? current.wire.rating : command.rating;
        current[desired].click();
        break;
      }
      case "toggle":
        if (command.value !== 0 && command.value !== 1) return;
        if (command.value === 0) {
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
