// Checks the browser extension's page script in Chrome: it runs as an init
// script in the page's main world, as the extension's MAIN-world content
// script does, against a page with an <audio> element and a media session.
const { chromium } = require("playwright");
const fs = require("node:fs"), path = require("node:path"), http = require("node:http"), assert = require("node:assert/strict");
const root = path.resolve(__dirname, "..");
const pageScript = fs.readFileSync(path.join(root, "plugins/nowplaying/extension/page.js"), "utf8");

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
      const data = silence(3), range = /bytes=(d+)-(d*)/.exec(req.headers.range || "");
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
      window.addEventListener("snoofer-media:report", (e) => window.reports.push(JSON.parse(e.detail)));
    });
    await page.addInitScript(pageScript);
    const base = "http://127.0.0.1:" + server.address().port;
    await page.goto(base + "/");
    const last = () => page.evaluate(() => window.reports.at(-1));
    const command = (c) => page.evaluate((c) => window.dispatchEvent(new CustomEvent("snoofer-media:command", { detail: JSON.stringify(c) })), c);

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

    await command({ op: "toggle" });
    await page.waitForFunction(() => document.getElementById("a").paused && window.reports.at(-1)?.state === "paused", null, { polling: 50 });
    await command({ op: "next" });
    assert.equal(await page.evaluate(() => window.nextCalls), 1, "page nexttrack handler not used");
    await command({ op: "seek", value: 1500 });
    await page.waitForFunction(() => Math.abs(document.getElementById("a").currentTime - 1.5) < 0.05, null, { polling: 50 });
    await page.waitForFunction(() => Math.abs((window.reports.at(-1)?.positionMs || 0) - 1500) < 100, null, { polling: 50 });
    await command({ op: "toggle" });
    await page.waitForFunction(() => !document.getElementById("a").paused, null, { polling: 50 });

    // A page with no media never reports.
    const quiet = await browser.newPage();
    await quiet.addInitScript(() => { window.reports = []; window.addEventListener("snoofer-media:report", (e) => window.reports.push(e.detail)); });
    await quiet.addInitScript(pageScript);
    await quiet.goto(base + "/quiet");
    await quiet.waitForTimeout(400);
    assert.equal(await quiet.evaluate(() => window.reports.length), 0);
    assert.deepEqual(errors, []);
    console.log("PASS: extension page script reports metadata, state and controls; toggle, next and seek work");
  } finally {
    await browser.close();
    server.close();
  }
})().catch((error) => { console.error(error); process.exit(1); });
