// Runs in the extension's isolated world and relays between page.js (window
// events with JSON strings) and the service worker (a runtime port). The port
// opens only once this frame has media, so idle pages cost nothing.
(() => {
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
    if (!port) connect();
    port.postMessage({ report: e.detail });
  });
})();
