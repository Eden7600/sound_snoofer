// The toolbar popup: connection status, host access (Firefox makes it
// optional) and the Snoofer port.
const api = globalThis.browser ?? chrome;
const { DEFAULT_PORT, validPort } = globalThis.SnooferLogic;
const all = { origins: ["<all_urls>"] };
const $ = (id) => document.getElementById(id);

async function refresh() {
  let status = null;
  try {
    status = await api.runtime.sendMessage({ type: "status" });
  } catch {
    status = null; // The background is starting; the next refresh answers.
  }
  const line = $("status");
  if (status && status.refused) {
    line.textContent = status.refused;
    line.className = "warn";
  } else if (status && status.connected) {
    line.textContent = "Connected to Snoofer";
    line.className = "ok";
  } else {
    line.textContent = "Snoofer is not running on this computer (port " + ((status && status.port) || DEFAULT_PORT) + ").";
    line.className = "muted";
  }
  const playing = status ? status.playing : 0;
  $("playing").textContent = playing === 1 ? "1 tab with media" : playing + " tabs with media";
  $("access").hidden = await api.permissions.contains(all);
  if (document.activeElement !== $("port")) $("port").value = String((status && status.port) || DEFAULT_PORT);
}

$("grant").addEventListener("click", async () => {
  // Must run in the click: browsers only grant permissions on a user gesture.
  if (await api.permissions.request(all)) $("access").hidden = true;
});

$("save").addEventListener("click", async () => {
  const port = validPort($("port").value);
  if (!port) {
    $("port").value = String(DEFAULT_PORT);
    return;
  }
  await api.storage.local.set({ port });
  refresh();
});

refresh();
setInterval(refresh, 1000);
