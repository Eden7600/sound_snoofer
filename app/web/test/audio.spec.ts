import { holdActions, reply } from "./action-host.ts";
import { expect, test } from "./host.ts";

test.describe("Audio", () => {
  test.beforeEach(async ({ gui }) => {
    await gui.open();
    await gui.page.getByRole("heading", { name: "Live Controls" }).waitFor();
  });

  test("shows live controls, strips and meters", async ({ gui }) => {
    const { page } = gui;
    await page.waitForFunction(
      () => (document.querySelector<HTMLImageElement>("[data-part=brandmark]")?.naturalWidth ?? 0) > 0,
    );
    await expect(page.locator("[data-part=strip]")).toHaveCount(2);
    const live = page.locator("section", { has: page.getByRole("heading", { name: "Live Controls", exact: true }) });
    await expect(live.getByText("Engine", { exact: true }), "Audio screen lacks the Engine health row").toHaveCount(1);
    await expect(page.locator("#summary").getByText("Healthy"), "health badge still in the header").toHaveCount(0);
    await expect(page.getByText("Plugin host"), "sidebar host footer still shown").toHaveCount(0);
    await expect(page.getByRole("heading", { name: "PLAYBACK", exact: true })).toHaveCount(1);
    await expect(page.getByText("A1 OUTPUT", { exact: true })).toHaveCount(0);
    const strip = page.locator("[data-part=strip]").first();
    await strip.locator("[data-part=peak]").waitFor();
    await expect(strip.getByText("LEVEL N/A"), "live meter shown as unavailable").toHaveCount(0);
    await expect(strip.getByText("-15.0 dBFS")).toHaveCount(1);
    expect(
      await strip
        .locator("[data-part=position] > div")
        .first()
        .evaluate((e) => (e as HTMLElement).style.width),
      "gain position track",
    ).toBe("75%");
    for (const mark of ["-60", "-30", "-12", "-3", "0"]) {
      await expect(strip.getByText(mark, { exact: true }), "meter scale " + mark).toHaveCount(1);
    }
    await expect(page.locator("#summary").getByText("Sends paused"), "paused sends badge").toHaveCount(1);
    await expect(page.locator("#summary").getByText("Devices paused"), "devices badge while not paused").toHaveCount(0);
    await expect(page.getByText("Device Priorities"), "priority controls leaked onto the Audio screen").toHaveCount(0);
  });

  test("shows recovery toggles with their status", async ({ gui }) => {
    const recovery = gui.page.locator("section", {
      has: gui.page.getByRole("heading", { name: "Routing & Recovery" }),
    });
    await expect(recovery.getByText("Disable Send Manipulation")).toHaveCount(1);
    await expect(recovery.getByText("3 held")).toHaveCount(1);
    await expect(recovery.getByRole("button", { name: /^(virtual:\d+|Computer|Virtual [23]) Playback$/ })).toHaveCount(
      0,
    );
    await gui.screen("Routing");
    await gui.page.getByRole("button", { name: "Playback Computer", exact: true }).click();
    const route = await gui.last("audio.playback:virtual:1");
    expect(route.Operation).toBe("press");
  });

  test("selection labels preserve raw choices and configured names", async ({ gui }) => {
    const normal = gui.page.locator("section", { has: gui.page.getByRole("heading", { name: "Normal", exact: true }) });
    expect(await normal.getByLabel("Processing").locator("option").allTextContents()).toEqual(["Direct", "Element"]);
    expect(await normal.getByLabel("Monitor").locator("option").allTextContents()).toEqual(["Off", "Pre", "Post"]);
    expect(await normal.getByLabel("Microphone").locator("option").allTextContents()).toEqual([
      "Automatic",
      "Desk · Silent",
      "Lavalier",
    ]);
    expect(await normal.getByLabel("Playback").locator("option").allTextContents()).toEqual([
      "Automatic",
      "Headphones (AirPods Pro)",
      "Speakers (SteelSeries Arena 7)",
      "Game (SteelSeries Arena 7)",
    ]);
    expect(await gui.page.getByLabel("Mic Stage", { exact: true }).locator("option").allTextContents()).toEqual([
      "Pre",
      "Post",
    ]);
    await normal.getByLabel("Processing").selectOption({ label: "Element" });
    expect((await gui.last("audio.normal-mode")).Value).toBe("element");
    await gui.settled();
    await normal.getByLabel("Monitor").selectOption({ label: "Post" });
    expect((await gui.last("audio.normal-monitor")).Value).toBe("post");
  });

  test("tape transport sends its commands", async ({ gui }) => {
    const { page } = gui;
    const tape = page.locator("[data-part=tape]");
    expect(await tape.getByRole("button").allTextContents()).toEqual(["Record", "Pause", "Stop", "", ""]);
    await expect(tape.getByRole("button", { name: "Fast-forward" })).toHaveCount(1);
    await expect(page.getByText("Play Recording"), "tape commands leaked into Routing & recovery").toHaveCount(0);
    await gui.update(() => {
      window.ctl("audio.tape-rew").Available = false;
    });
    await expect(tape.getByRole("button", { name: "Rewind" }), "unavailable tape control enabled").toBeDisabled();
    await tape.getByRole("button", { name: "Stop" }).click();
    await gui.last("audio.tape-stop");
  });

  test("echo cancellation shows status and applies choices", async ({ gui }) => {
    const { page } = gui;
    const echo = page.locator("[data-part=echo]");
    await echo.getByText("2.5 / 1.2 ms", { exact: true }).waitFor();
    await echo.getByText("Peak worker / queue · gaps 3 · underruns 0", { exact: true }).waitFor();
    await expect(echo.getByRole("heading", { name: "Echo Cancellation" })).toHaveCount(1);
    await expect(echo.getByText("Active", { exact: true }), "echo status value").toHaveCount(1);
    await expect(echo.getByText("−32 dB echo · 54 ms"), "echo status detail").toHaveCount(1);
    await expect(echo.getByLabel("Mode")).toHaveValue("auto");
    expect(await echo.getByLabel("Strength").locator("option").allTextContents()).toEqual([
      "Strong",
      "Balanced",
      "Gentle",
    ]);
    expect(await echo.getByLabel("Engine").locator("option").allTextContents()).toEqual([
      "WebRTC AEC3",
      "LocalVQE Echo-Only",
      "LocalVQE Voice Cleanup",
      "LocalVQE Full-Band",
    ]);
    await echo.getByLabel("Engine").selectOption("localvqe-full");
    await page.waitForFunction(() => window.ctl("aec.engine").Value === "localvqe-full");
    await gui.settled();
    await echo.getByLabel("Mode").selectOption("off");
    await page.waitForFunction(() => window.ctl("aec.mode").Value === "off");
  });

  test("echo cancellation disables unavailable choices and retries", async ({ gui }) => {
    const { page } = gui;
    const echo = page.locator("[data-part=echo]");
    const retryButton = echo.getByRole("button", { name: "Retry", exact: true });
    await expect(
      echo.locator("[data-part=row]", { has: page.getByRole("button", { name: "Retry", exact: true }) }),
      "Retry has an empty field caption",
    ).toHaveCount(0);
    await gui.update(() => {
      window.ctl("aec.strength").Available = false;
      window.ctl("aec.status").Status = "48 kHz mono · hybrid · ~63 ms processing latency";
    });
    await expect(echo.getByLabel("Strength")).toBeDisabled();
    await echo.getByText("48 kHz mono · hybrid · ~63 ms processing latency").waitFor();
    await expect(echo.getByRole("button", { name: "Retry", exact: true })).toBeDisabled();
    await gui.update(() => {
      window.ctl("aec.retry").Available = true;
      window.ctl("aec.status").Value = "Error";
      window.ctl("aec.status").Status =
        "Engine error (missing output callback); mic passes through; automatic retry pending";
    });
    await holdActions(gui);
    await echo.getByRole("button", { name: "Retry", exact: true }).click();
    const retry = await gui.last("aec.retry");
    expect(retry.Operation).toBe("press");
    await expect(retryButton).toBeDisabled();
    await gui.update(() => {
      window.ctl("aec.retry").Revision++;
    });
    await expect(retryButton).toBeDisabled();
    await reply(gui, "aec.retry", "Submitted", "Retry submitted");
    await gui.settled();
    await expect(retryButton).toBeEnabled();
  });
});

test.describe("Audio in preview", () => {
  test("live-only controls are unavailable", async ({ gui }) => {
    await gui.open(undefined, "preview");
    await gui.page.getByRole("heading", { name: "Live Controls" }).waitFor();
    const tape = gui.page.locator("[data-part=tape]");
    for (const name of ["Stop", "Rewind", "Fast-forward"]) {
      await expect(tape.getByRole("button", { name }), name + " enabled in preview").toBeDisabled();
    }
  });
});
