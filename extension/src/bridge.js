// Runs in the extension's isolated world and relays between page.js (window
// events with JSON strings) and the service worker (a runtime port). The port
// opens only once this frame has media, so idle pages cost nothing.
(() => {
  // Injection at install can run this again in the same world; one copy is
  // enough. A copy from an older install lives in another world.
  if (globalThis.__snooferBridge) return;
  globalThis.__snooferBridge = true;
  // A newer copy may be injected after an install or update; the old copy's
  // extension context is then gone and its sends fail, which is ignored.
  let port = null;
  function connect() {
    port = chrome.runtime.connect({ name: "media" });
    port.onMessage.addListener((command) => {
      window.dispatchEvent(new CustomEvent("snoofer-media:command", { detail: JSON.stringify(command) }));
    });
    port.onDisconnect.addListener(() => {
      port = null; // The worker restarted; the next report reconnects.
    });
  }
  window.addEventListener("snoofer-media:report", (e) => {
    if (typeof e.detail !== "string" || e.detail.length > 16384) return;
    try {
      if (!port) connect();
      port.postMessage({ report: e.detail });
    } catch {
      port = null; // Orphaned after an update: a newer copy carries reports now.
    }
  });
})();
