// Checks the browser extension's page script in Chrome: it runs as an init
// script in the page's main world, as the extension's MAIN-world content
// script does, against a page with an <audio> element and a media session.
const { chromium } = require("playwright");
const fs = require("node:fs"), path = require("node:path"), http = require("node:http"), assert = require("node:assert/strict");
const root = path.resolve(__dirname, "..");
const pageScript = fs.readFileSync(path.join(root, "extension/src/page.js"), "utf8");

// Three seconds of silent 8 kHz mono 16-bit PCM.
function silence(seconds) {
  const samples = 8000 * seconds, data = Buffer.alloc(44 + samples * 2);
  data.write("RIFF", 0); data.writeUInt32LE(36 + samples * 2, 4); data.write("WAVEfmt ", 8);
  data.writeUInt32LE(16, 16); data.writeUInt16LE(1, 20); data.writeUInt16LE(1, 22); data.writeUInt32LE(8000, 24);
  data.writeUInt32LE(16000, 28); data.writeUInt16LE(2, 32); data.writeUInt16LE(16, 34); data.write("data", 36); data.writeUInt32LE(samples * 2, 40);
  return data;
}

const html = `<!doctype html><title>Tab title</title><audio id="a" src="/clip.wav"></audio><script>
navigator.mediaSession.metadata = new MediaMetadata({ title: "Song", artist: "Band", album: "Album",
  artwork: [{ src: "/small.png", sizes: "96x96" }, { src: "/large.png", sizes: "512x512" }] });
window.nextCalls = 0;
navigator.mediaSession.setActionHandler("nexttrack", () => { window.nextCalls++; });
</script>`;

(async () => {
  const server = http.createServer((req, res) => {
    if (req.url === "/clip.wav") {
      // Seeking needs range requests.
      const data = silence(3), range = /bytes=(\d+)-(\d*)/.exec(req.headers.range || "");
      res.setHeader("Content-Type", "audio/wav");
      res.setHeader("Accept-Ranges", "bytes");
      if (!range) { res.end(data); return; }
      const start = Number(range[1]), end = range[2] ? Number(range[2]) : data.length - 1;
      res.writeHead(206, { "Content-Range": "bytes " + start + "-" + end + "/" + data.length, "Content-Length": end - start + 1 });
      res.end(data.subarray(start, end + 1));
      return;
    }
    res.setHeader("Content-Type", "text/html");
    res.end(req.url === "/quiet" ? "<!doctype html><p>No media</p>" : html);
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  const browser = await chromium.launch({ channel: "chrome", headless: true, args: ["--autoplay-policy=no-user-gesture-required"] });
  try {
    const page = await browser.newPage();
    const errors = [];
    page.on("pageerror", (e) => errors.push(String(e)));
    await page.addInitScript(() => {
      window.reports = [];
      window.addEventListener("snoofer-media:report-v3", (e) => window.reports.push(JSON.parse(e.detail)));
    });
    await page.addInitScript(pageScript);
    const base = "http://127.0.0.1:" + server.address().port;
    await page.goto(base + "/");
    const last = () => page.evaluate(() => window.reports.at(-1));
    const command = (c) => page.evaluate((c) => window.dispatchEvent(new CustomEvent("snoofer-media:command-v3", { detail: JSON.stringify(c) })), c);

    await page.evaluate(() => document.getElementById("a").play());
    await page.waitForFunction(() => window.reports.at(-1)?.state === "playing", null, { polling: 50 });
    let r = await last();
    assert.equal(r.present, true);
    assert.equal(r.title, "Song");
    assert.equal(r.artist, "Band");
    assert.equal(r.artwork, base + "/large.png", "largest artwork, absolute");
    assert.equal(r.canNext, true);
    assert.equal(r.canPrev, false);
    assert.equal(r.canSeek, true);
    assert.ok(Math.abs(r.durationMs - 3000) < 50, "duration " + r.durationMs);

    await command({ op: "toggle", value: 0 });
    await page.waitForFunction(() => document.getElementById("a").paused && window.reports.at(-1)?.state === "paused", null, { polling: 50 });
    await command({ op: "next" });
    assert.equal(await page.evaluate(() => window.nextCalls), 1, "page nexttrack handler not used");
    await command({ op: "seek", value: 1500 });
    await page.waitForFunction(() => Math.abs(document.getElementById("a").currentTime - 1.5) < 0.05, null, { polling: 50 });
    await page.waitForFunction(() => Math.abs((window.reports.at(-1)?.positionMs || 0) - 1500) < 100, null, { polling: 50 });
    await command({ op: "toggle", value: 1 });
    await page.waitForFunction(() => !document.getElementById("a").paused, null, { polling: 50 });

    // Injected into a tab that is already playing (installing the extension
    // mid-video), the script adopts the element and reports it.
    const late = await browser.newPage();
    await late.goto(base + "/");
    await late.evaluate(() => document.getElementById("a").play());
    await late.waitForFunction(() => !document.getElementById("a").paused, null, { polling: 50 });
    await late.evaluate(() => { window.reports = []; window.addEventListener("snoofer-media:report-v3", (e) => window.reports.push(JSON.parse(e.detail))); });
    await late.evaluate(pageScript);
    await late.waitForFunction(() => window.reports.at(-1)?.state === "playing" && window.reports.at(-1)?.title === "Song", null, { polling: 50 });

    // A page with no media never reports.
    const quiet = await browser.newPage();
    await quiet.addInitScript(() => { window.reports = []; window.addEventListener("snoofer-media:report-v3", (e) => window.reports.push(e.detail)); });
    await quiet.addInitScript(pageScript);
    await quiet.goto(base + "/quiet");
    await quiet.waitForTimeout(400);
    assert.equal(await quiet.evaluate(() => window.reports.length), 0);
    assert.deepEqual(errors, []);
    console.log("PASS: extension page script reports metadata, state and controls; toggle, next and seek work");
    await browser.close();
    await live(base);
  } finally {
    await browser.close().catch(() => {});
    server.close();
  }
})().catch((error) => { console.error(error); process.exit(1); });

// bridge is a minimal stand-in for Snoofer's WebSocket bridge: text frames
// only, enough to record what the extension sends and to send it commands.
function bridge() {
  const crypto = require("node:crypto");
  const messages = [];
  let socket = null;
  const server = http.createServer((_, res) => { res.writeHead(404); res.end(); });
  server.on("upgrade", (req, sock) => {
    const accept = crypto.createHash("sha1").update(req.headers["sec-websocket-key"] + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11").digest("base64");
    sock.write("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + accept + "\r\n\r\n");
    socket = sock;
    messages.origin = req.headers.origin;
    let buffer = Buffer.alloc(0);
    sock.on("data", (chunk) => {
      buffer = Buffer.concat([buffer, chunk]);
      for (;;) {
        if (buffer.length < 2) return;
        let length = buffer[1] & 0x7f, offset = 2;
        if (length === 126) { if (buffer.length < 4) return; length = buffer.readUInt16BE(2); offset = 4; }
        else if (length === 127) { if (buffer.length < 10) return; length = Number(buffer.readBigUInt64BE(2)); offset = 10; }
        if (buffer.length < offset + 4 + length) return;
        const mask = buffer.subarray(offset, offset + 4), data = Buffer.from(buffer.subarray(offset + 4, offset + 4 + length));
        for (let i = 0; i < data.length; i++) data[i] ^= mask[i % 4];
        if ((buffer[0] & 0x0f) === 1) messages.push(JSON.parse(data.toString("utf8")));
        buffer = buffer.subarray(offset + 4 + length);
      }
    });
    sock.on("error", () => {});
  });
  const send = (value) => {
    const data = Buffer.from(JSON.stringify(value));
    const head = data.length < 126 ? Buffer.from([0x81, data.length]) : Buffer.from([0x81, 126, data.length >> 8, data.length & 0xff]);
    socket.write(Buffer.concat([head, data]));
  };
  return { server, messages, send, close: () => { if (socket) socket.destroy(); server.close(); } };
}

// live loads the built extension in Brave with a fresh profile, points it at
// the stand-in bridge, plays media and checks the whole path both ways.
async function live(base) {
  const os = require("node:os");
  const brave = "C:/Program Files/BraveSoftware/Brave-Browser/Application/brave.exe";
  if (!fs.existsSync(brave)) { console.log("SKIP: live extension check needs Brave"); return; }
  require("node:child_process").execFileSync(process.execPath, [path.join(root, "extension/build.mjs")], { stdio: "ignore" });
  const ext = path.join(root, "extension/dist/chrome");
  const fake = bridge();
  await new Promise((resolve) => fake.server.listen(0, "127.0.0.1", resolve));
  const profile = fs.mkdtempSync(path.join(os.tmpdir(), "snoofer-brave-"));
  const context = await chromium.launchPersistentContext(profile, { executablePath: brave, headless: true,
    args: ["--headless=new", "--autoplay-policy=no-user-gesture-required", `--disable-extensions-except=${ext}`, `--load-extension=${ext}`] });
  try {
    const worker = context.serviceWorkers()[0] || await context.waitForEvent("serviceworker");
    await worker.evaluate((port) => chrome.storage.local.set({ port }), fake.server.address().port);
    const until = async (what, ready) => {
      for (const end = Date.now() + 10000; Date.now() < end; await new Promise((r) => setTimeout(r, 50))) {
        const found = fake.messages.findLast(ready);
        if (found) return found;
      }
      throw new Error("timed out waiting for " + what + ": " + JSON.stringify(fake.messages.slice(-3)));
    };
    const hello = await until("hello", (m) => m.type === "hello");
    assert.equal(hello.protocol, 2);
    assert.equal(hello.browser, "Brave");
    assert.match(fake.messages.origin, /^chrome-extension:\/\//);

    const page = await context.newPage();
    await page.goto(base + "/");
    await page.evaluate(() => document.getElementById("a").play());
    const playing = await until("playing session", (m) => m.type === "sessions" && m.sessions.some((s) => s.title === "Song" && s.state === "playing"));
    const session = playing.sessions.find((s) => s.title === "Song");
    assert.equal(session.artist, "Band");
    assert.ok(typeof session.instance === "string" && session.instance.length > 0, "session has a document instance");
    assert.equal(session.site, "127.0.0.1");
    assert.equal(session.canNext, true);

    fake.send({ type: "command", id: session.id, instance: session.instance, op: "toggle", value: 0 });
    await page.waitForFunction(() => document.getElementById("a").paused, null, { polling: 50, timeout: 10000 });
    await until("paused session", (m) => m.type === "sessions" && m.sessions.some((s) => s.id === session.id && s.state === "paused"));
    fake.send({ type: "command", id: session.id, instance: session.instance, op: "next", value: 0 });
    await page.waitForFunction(() => window.nextCalls === 1, null, { polling: 50, timeout: 10000 });
    fake.send({ type: "command", id: session.id, instance: session.instance, op: "mute", value: 1 });
    await until("muted tab", (m) => m.type === "sessions" && m.sessions.some((s) => s.id === session.id && s.muted));

    // The install hook injects into open tabs. On a tab that already has the
    // scripts it must be harmless: still one session, still controllable.
    // (Automation cannot restart an unpacked extension after an update, and the
    // late-injection adoption itself is checked in Chrome above.)
    await page.evaluate(() => document.getElementById("a").play());
    await until("playing again", (m) => m.type === "sessions" && m.sessions.some((s) => s.id === session.id && s.state === "playing"));
    await worker.evaluate(() => injectOpenTabs());
    fake.messages.length = 0;
    fake.send({ type: "command", id: session.id, instance: session.instance, op: "toggle", value: 0 });
    await page.waitForFunction(() => document.getElementById("a").paused, null, { polling: 50, timeout: 10000 });
    const after = await until("paused after injection", (m) => m.type === "sessions" && m.sessions.some((s) => s.id === session.id && s.state === "paused"));
    assert.equal(after.sessions.filter((s) => s.title === "Song").length, 1, "injection duplicated the session");
    console.log("PASS: extension in Brave connects, reports the tab, runs toggle, next and mute, and re-injects into open tabs harmlessly");
    await ratings(context, fake, until);
    await meetControls(context, fake, until);
  } finally {
    await context.close();
    fake.close();
    fs.rmSync(profile, { recursive: true, force: true });
  }
}

// The exact origin is intercepted in this fresh headless profile. Every request
// is fulfilled locally; this fixture never reaches YouTube or an account.
async function ratings(context, fake, until) {
  const page = await context.newPage();
  const errors = [];
  page.on("pageerror", (error) => errors.push(String(error)));
  await page.route("**/*", (route) => route.fulfill({ contentType: "text/html", body: `<!doctype html>
<title>Isolated rating fixture</title><div id="movie_player"><video></video></div>
<ytmusic-player-bar><ytmusic-like-button-renderer id="like-button-renderer" like-status="LIKE">
  <div class="like-button-wrapper"><yt-button-shape><button aria-pressed="true">Localized like</button></yt-button-shape></div>
  <yt-button-shape><button aria-pressed="false">Localized dislike</button></yt-button-shape>
</ytmusic-like-button-renderer></ytmusic-player-bar>
<script>
const player = document.querySelector('#movie_player');
const renderer = document.querySelector('#like-button-renderer');
const like = renderer.querySelector('.like-button-wrapper button');
const dislike = renderer.querySelector(':scope > yt-button-shape > button');
window.clicks = 0;
window.song = 'fixture-a';
player.getVideoData = () => ({video_id: window.song});
player.getPlayerState = () => 2;
renderer.data = {target: {videoId: window.song}, likeStatus: 'LIKE', likesAllowed: true};
window.setRating = (status) => {
  renderer.setAttribute('like-status', status);
  like.setAttribute('aria-pressed', String(status === 'LIKE'));
  dislike.setAttribute('aria-pressed', String(status === 'DISLIKE'));
};
like.onclick = () => { window.clicks++; setRating(renderer.getAttribute('like-status') === 'LIKE' ? 'INDIFFERENT' : 'LIKE'); };
dislike.onclick = () => { window.clicks++; setRating(renderer.getAttribute('like-status') === 'DISLIKE' ? 'INDIFFERENT' : 'DISLIKE'); };
window.setSong = (id) => { window.song = id; renderer.data.target.videoId = id; };
navigator.mediaSession.metadata = new MediaMetadata({title: 'Fixture song', artist: 'Fixture artist'});
</script>` }));
  await page.goto("https://music.youtube.com/isolated-snoofer-fixture");
  const initial = await until("paused rating fixture", (m) => m.type === "sessions" && m.sessions.some((s) => s.title === "Fixture song" && s.canRate && s.rating === "like"));
  const session = initial.sessions.find((s) => s.title === "Fixture song");
  assert.equal(session.state, "paused");
  const send = (rating, track = session.track) => fake.send({ type: "command", id: session.id, instance: session.instance, op: "rate", track, rating });
  send("none");
  await until("cleared rating", (m) => m.type === "sessions" && m.sessions.some((s) => s.id === session.id && s.rating === "none"));
  send("dislike");
  await until("disliked rating", (m) => m.type === "sessions" && m.sessions.some((s) => s.id === session.id && s.rating === "dislike"));
  assert.equal(await page.evaluate(() => window.clicks), 2);
  assert.equal(await page.evaluate(() => document.querySelector("#like-button-renderer").data.likeStatus), "LIKE", "auxiliary rating stays unchanged while live rating switches");
  fake.messages.length = 0;
  await page.evaluate(() => window.setRating("LIKE"));
  await until("paused manual rating", (m) => m.type === "sessions" && m.sessions.some((s) => s.id === session.id && s.rating === "like" && s.state === "paused"));
  send("like");
  await page.waitForTimeout(300);
  assert.equal(await page.evaluate(() => window.clicks), 2, "already desired rating is a no-op");

  fake.messages.length = 0;
  await page.evaluate(() => document.querySelector('.like-button-wrapper button').setAttribute('aria-pressed', 'false'));
  await until("inconsistent rating", (m) => m.type === "sessions" && m.sessions.some((s) => s.id === session.id && s.rating === "unknown" && !s.canRate));
  send("none"); await page.waitForTimeout(300);
  assert.equal(await page.evaluate(() => window.clicks), 2);
  await page.evaluate(() => { window.setRating("LIKE"); window.setSong("fixture-b"); });
  const changed = await until("same-title replacement track", (m) => m.type === "sessions" && m.sessions.some((s) => s.id === session.id && s.canRate && s.track !== session.track));
  const second = changed.sessions.find((s) => s.id === session.id);
  send("none"); await page.waitForTimeout(300);
  assert.equal(await page.evaluate(() => window.clicks), 2, "stale track cannot click");
  send("none", second.track);
  await page.waitForFunction(() => window.clicks === 3);
  assert.deepEqual(errors, []);
  await page.close();
  console.log("PASS: isolated YouTube Music DOM and real extension bridge handle paused ratings, clear/switch, unknown state and stale track rejection");
}

// The Meet origin and all subresources are fulfilled locally in the temporary
// headless profile. No real call, account, camera or microphone is involved.
async function meetControls(context, fake, until) {
  fake.send({ type: "capabilities", meet: 1, mediaEnabled: true, meetEnabled: true });
  const page = await context.newPage();
  const errors = [];
  page.on("pageerror", (error) => errors.push(String(error)));
  await page.route("**/*", (route) => route.fulfill({ contentType: "text/html", body: `<!doctype html><title>Isolated Meet fixture</title>
<div role="region" jsname="vNB5le"><button jsname="hw0c9" data-is-muted="true">Mic</button><button jsname="psRWwc" data-is-muted="true">Camera</button><button jsname="CQylAd">Leave</button></div>
<script>
window.meetClicks = {mic:0,camera:0}; window.leaveClicks = 0;
const wire = () => {
  document.querySelector('button[jsname="hw0c9"]').onclick = function() { window.meetClicks.mic++; this.setAttribute('data-is-muted', String(this.getAttribute('data-is-muted') !== 'true')); };
  document.querySelector('button[jsname="psRWwc"]').onclick = function() { window.meetClicks.camera++; if (!window.ignoreCamera) this.setAttribute('data-is-muted', String(this.getAttribute('data-is-muted') !== 'true')); };
  document.querySelector('button[jsname="CQylAd"]').onclick = function() { window.leaveClicks++; this.closest('[role="region"]').remove(); document.body.insertAdjacentHTML('beforeend', '<button jsname="hw0c9" data-is-muted="true">Mic</button><button jsname="psRWwc" data-is-muted="true">Camera</button>'); };
}; wire();
window.replaceCall = () => { const old = document.querySelector('[role="region"]'); const replacement = old.cloneNode(true); old.replaceWith(replacement); wire(); };
window.replaceMic = () => { const old = document.querySelector('button[jsname="hw0c9"]'); const replacement = old.cloneNode(true); replacement.setAttribute('data-is-muted', 'true'); old.replaceWith(replacement); wire(); };
</script>` }));
  await page.goto("https://meet.google.com/isolated-snoofer-fixture");
  const joined = await until("isolated Meet joined state", (m) => m.type === "meet-state" && m.calls.some((c) => c.state === "joined" && c.mic.available && c.camera.available));
  let call = joined.calls.find((c) => c.state === "joined");
  assert.equal(call.mic.identity, "1"); assert.equal(call.camera.identity, "1");
  assert.equal(await page.evaluate(() => window.__snooferMedia), undefined, "Meet must not install the generic Media page helper");
  let request = 100;
  const send = (control, value, target = call) => {
    const id = String(++request);
    fake.send({ type: "meet-command", id, tab: target.tab, instance: target.instance, call: target.call, control, revision: target[control].revision, value });
    return id;
  };
  const result = (id) => until("Meet result " + id, (m) => m.type === "meet-result" && m.id === id);
  const fresh = async (ready) => {
    const state = await until("fresh Meet state", (m) => m.type === "meet-state" && m.calls.some((c) => c.tab === call.tab && ready(c)));
    call = state.calls.find((c) => c.tab === call.tab); return call;
  };
  assert.equal(call.mic.value, true); assert.equal(call.camera.value, false);
  assert.equal((await result(send("mic", true))).status, "observed");
  assert.deepEqual(await page.evaluate(() => window.meetClicks), { mic: 0, camera: 0 });
  assert.equal((await result(send("mic", false))).status, "observed");
  await fresh((c) => c.mic.value === false);
  assert.equal((await result(send("camera", true))).status, "observed");
  await fresh((c) => c.camera.value === true);
  assert.deepEqual(await page.evaluate(() => window.meetClicks), { mic: 1, camera: 1 });

  assert.equal(call.mic.identity, "1", "value changes retain control identity");
  const oldControl = call;
  await page.evaluate(() => window.replaceMic());
  await fresh((c) => c.mic.identity !== oldControl.mic.identity);
  assert.equal(call.call, oldControl.call); assert.equal(call.mic.value, true);
  assert.equal((await result(send("mic", false, oldControl))).status, "rejected");
  assert.deepEqual(await page.evaluate(() => window.meetClicks), { mic: 1, camera: 1 });

  const old = call;
  await page.evaluate(() => window.replaceCall());
  await fresh((c) => c.call !== old.call);
  assert.equal((await result(send("mic", true, old))).status, "rejected");
  assert.deepEqual(await page.evaluate(() => window.meetClicks), { mic: 1, camera: 1 });

  await page.evaluate(() => { window.ignoreCamera = true; });
  const uncertain = send("camera", false);
  assert.equal((await result(uncertain)).status, "unknown");
  assert.equal(await page.evaluate(() => window.meetClicks.camera), 2);
  await page.waitForTimeout(500); assert.equal(await page.evaluate(() => window.meetClicks.camera), 2, "no replay after timeout");

  fake.send({ type: "capabilities", meet: 1, mediaEnabled: true, meetEnabled: false });
  await page.waitForTimeout(100);
  assert.equal((await result(send("mic", true))).reason, "disabled");
  fake.send({ type: "capabilities", meet: 1, mediaEnabled: true, meetEnabled: true });
  const previousSample = call.sample;
  await fresh((c) => BigInt(c.sample) > BigInt(previousSample));
  assert.deepEqual(await page.evaluate(() => window.meetClicks), { mic: 1, camera: 2 });
  assert.equal(call.leave.available, true);
  assert.equal((await result(send("leave", true))).status, "observed");
  assert.equal(await page.evaluate(() => window.leaveClicks), 1);
  await fresh((c) => c.state === "prejoin" && !c.mic.available && !c.camera.available);
  assert.deepEqual(errors, []);
  await page.close();
  console.log("PASS: isolated Meet DOM/runtime bridge confirms one-shot mic/camera changes and Leave departure, rejects stale calls, expires uncertain results and excludes prejoin without replay");
}
