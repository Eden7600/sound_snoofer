import { expect, test, type Gui } from "./host";

const lastValue = async (gui: Gui, id: string): Promise<unknown> => JSON.parse((await gui.last(id)).Value);

test.describe("Routing", () => {
  test.beforeEach(async ({ gui }) => {
    await gui.open();
    await gui.screen("Routing");
    for (const name of ["Outputs", "Microphones", "Interfaces", "Playback", "Mic Priority"]) {
      await gui.page.getByRole("heading", { name, exact: true }).waitFor();
    }
  });

  test("outputs matrix shows sources by destination", async ({ gui }) => {
    const outputs = gui.page.locator("[data-part=outputs]");
    await expect(outputs.locator("[data-part=output-slot]"), "hidden slot positions shown").toHaveCount(1);
    expect(await outputs.locator("tbody th").allTextContents()).toEqual(["Computer", "Monitor", "Soundboard", "Tape"]);
    // The Monitor cell shows the raw value "pre"; gui-content-fixes labels it (docs/gui-audit.md item 7).
    expect(await outputs.locator("[data-part=route]").allTextContents()).toEqual(["On", "On", "pre", "Off", "Off", "On"]);
  });

  test("routes dispatch on click, never on focus", async ({ gui }) => {
    const outputs = gui.page.locator("[data-part=outputs]");
    await outputs.getByRole("button", { name: "Music Monitor" }).focus();
    await gui.page.keyboard.press("Tab");
    expect(await gui.sent(), "focus moves dispatched a route").toEqual([]);
    await outputs.getByRole("button", { name: "Music Monitor" }).click();
    expect((await gui.last("audio.slot-1:monitor")).Operation).toBe("press");
  });

  test("output slots are added and removed", async ({ gui }) => {
    const outputs = gui.page.locator("[data-part=outputs]");
    // Slots may exist without a device.
    await expect(outputs.getByLabel("New Output Device").locator("option").first()).toHaveText("No device");
    await expect(outputs.getByLabel("Music device").locator("option").first()).toHaveText("No device");
    await outputs.getByLabel("New Output Device").selectOption("Speakers (Realtek Audio)");
    await outputs.getByLabel("New Output Name").fill("Monitor output");
    await outputs.getByRole("button", { name: "Add Output" }).click();
    expect(await lastValue(gui, "audio.output-edit")).toEqual({
      op: "add",
      id: "",
      value: JSON.stringify({ name: "Monitor output", device: "Speakers (Realtek Audio)", device_id: "{realtek}" }),
    });
    await gui.settled();
    await outputs.getByRole("button", { name: "Remove Music" }).click();
    await expect.poll(async () => (await gui.sentValues("audio.output-edit")).length).toBe(2);
    expect(await lastValue(gui, "audio.output-edit")).toEqual({ op: "remove", id: "music", value: "" });
  });

  test("priority lists show matches and limits", async ({ gui }) => {
    const { page } = gui;
    const playback = page.locator("[data-part=list-playback]");
    expect(await playback.locator("[data-part=match]").allTextContents()).toEqual(["In use", "Ambiguous · 2", "Ready"]);
    await expect(playback.getByRole("button", { name: "Move Playback 1 up" }), "first entry moves up").toBeDisabled();
    await expect(
      page.locator("[data-part=list-interfaces]").getByRole("button", { name: "Remove interface 1" }),
      "last interface removable",
    ).toBeDisabled();
    // Identity entries show the device, not a pattern.
    await expect(playback.locator("[data-part=device-entry]").first()).toHaveText("Headphones (AirPods Pro)");
    await expect(playback.getByLabel("Playback 1 pattern")).toHaveCount(0);
  });

  test("microphones are named and numbered by input", async ({ gui }) => {
    const { page } = gui;
    const mics = page.locator("[data-part=mics]");
    await expect(mics.locator("[data-part=mic]")).toHaveCount(3);
    await expect(mics.getByLabel("Microphone 2 name")).toHaveValue("Lavalier");
    await expect(mics.locator("[data-part=match]")).toHaveText("No match");
    await expect(page.locator("[data-part=list-microphones]").getByText("Lavalier"), "priority shows configured names").toHaveCount(1);
    expect(await page.locator("[data-part=list-microphones] [data-part=priority] [data-tone]").allTextContents()).toEqual(["In use", "Unavailable"]);
  });

  test("suggestions add by identity", async ({ gui }) => {
    const playback = gui.page.locator("[data-part=list-playback]");
    // Pattern forms are Snoofer's generated ones.
    expect(await playback.locator("[data-part=suggestions] button").allTextContents()).toEqual(["Add", "Exact Pattern", "Partial Pattern"]);
    await playback.getByRole("button", { name: "Add Speakers (Realtek Audio)", exact: true }).click();
    expect(await lastValue(gui, "audio.priority-edit")).toEqual({
      list: "playback",
      op: "add",
      index: 0,
      value: JSON.stringify({ driver: "wdm", id: "{realtek}", name: "Speakers (Realtek Audio)" }),
      field: "",
    });
  });

  test("a pattern applies only on Enter, never while typing", async ({ gui }) => {
    const pattern = gui.page.locator("[data-part=list-playback]").getByLabel("Playback 2 pattern");
    await pattern.fill("(?i)arena 7");
    expect(await gui.sentValues("audio.priority-edit"), "typing dispatched an edit").toEqual([]);
    await pattern.press("Enter");
    expect(((await lastValue(gui, "audio.priority-edit")) as { value: string }).value).toBe("(?i)arena 7");
  });

  test("interfaces are added from driver suggestions", async ({ gui }) => {
    const add = gui.page.locator("[data-part=interface-add]");
    await add.getByLabel("New Interface Presence Input").selectOption({ label: "Analogue 1 + 2 (Focusrite USB)" });
    await add.getByRole("button", { name: "Add" }).click();
    const edit = (await lastValue(gui, "audio.priority-edit")) as { value: string };
    expect(JSON.parse(edit.value)).toEqual({
      asio_id: "{focusrite}",
      asio_name: "Focusrite USB ASIO",
      presence_pattern: "(?i)^Analogue 1 \\+ 2 \\(Focusrite USB\\)$",
      inputs: {},
    });
  });

  test("interface channels: off, mono or stereo per microphone", async ({ gui }) => {
    const interfaces = gui.page.locator("[data-part=list-interfaces]");
    await expect(interfaces.getByLabel("Desk on interface 1")).toHaveValue("mono");
    await expect(interfaces.getByLabel("Desk channel for interface 1")).toHaveValue("1");
    await expect(interfaces.getByLabel("Webcam on interface 1"), "device microphone offered interface channels").toHaveCount(0);
    await interfaces.getByLabel("Lavalier on interface 1").selectOption("stereo");
    expect(await lastValue(gui, "audio.priority-edit")).toEqual({ list: "interfaces", op: "set", index: 0, value: "2,3", field: "input:lav" });
    await gui.settled();
    await interfaces.getByLabel("Desk channel for interface 1").fill("5");
    await interfaces.getByLabel("Desk channel for interface 1").press("Tab");
    await expect.poll(async () => (await gui.sentValues("audio.priority-edit")).length).toBe(2);
    expect(await lastValue(gui, "audio.priority-edit")).toEqual({ list: "interfaces", op: "set", index: 0, value: "5", field: "input:desk" });
  });

  test("microphone readiness and source", async ({ gui }) => {
    const mics = gui.page.locator("[data-part=mics]");
    expect(await mics.locator("[data-part=mic-state]").allTextContents()).toEqual(["Silent", "In use", "Unavailable"]);
    await expect(mics.getByRole("button", { name: "Lavalier always ready" })).toHaveAttribute("aria-pressed", "true");
    await mics.getByRole("button", { name: "Desk always ready" }).click();
    expect(await lastValue(gui, "audio.priority-edit")).toEqual({ list: "mics", op: "set", index: 0, value: "true", field: "ready" });
    await gui.settled();
    await expect(mics.getByLabel("Webcam source")).toHaveValue("current");
    await mics.getByLabel("Desk source").selectOption("Microphone (Realtek Audio)");
    await expect.poll(async () => (await gui.sentValues("audio.priority-edit")).length).toBe(2);
    const source = (await lastValue(gui, "audio.priority-edit")) as { field: string; value: string };
    expect(source.field).toBe("source");
    expect(JSON.parse(source.value).driver).toBe("wdm");
  });

  test("activity metering settings", async ({ gui }) => {
    // The provider sends activity settings as {check, silence_db, silent_after_s}, but the GUI
    // reads {Check, SilenceDB, SilentAfterS}: docs/gui-audit.md item 15, owned by gui-content-fixes.
    test.fail(true, "activity settings key mismatch; gui-content-fixes");
    const activity = gui.page.locator("[data-part=activity]");
    await expect(activity.getByLabel("Meter")).toHaveValue("2");
    await activity.getByLabel("Silent After (s)").fill("5");
    await activity.getByLabel("Silent After (s)").press("Tab");
    expect(await lastValue(gui, "audio.priority-edit")).toEqual({ list: "activity", op: "set", index: 0, value: "5", field: "silent_after_s" });
    await gui.settled();
    await activity.getByLabel("Meter").selectOption("0");
    await expect.poll(async () => (await gui.sentValues("audio.priority-edit")).length).toBe(2);
    expect(await lastValue(gui, "audio.priority-edit")).toEqual({ list: "activity", op: "set", index: 0, value: "0", field: "check" });
  });

  test("a new microphone uses interface channels by default", async ({ gui }) => {
    const mics = gui.page.locator("[data-part=mics]");
    await mics.getByLabel("New Microphone Name").fill("Guest");
    await mics.getByRole("button", { name: "Add Microphone", exact: true }).click();
    expect(await lastValue(gui, "audio.priority-edit")).toEqual({ list: "mics", op: "add", index: 0, value: JSON.stringify({ name: "Guest" }), field: "" });
  });

  test("edit errors show on the screen", async ({ gui }) => {
    await gui.update(() => {
      window.ctl("audio.priority-edit").Status = "error parsing regexp: missing closing ): `(unclosed`";
    });
    await gui.page.locator("[data-part=priority-error]").getByText("missing closing").waitFor();
  });
});
