// Hand-written provider scenarios retained from scripts/check-gui.cjs.
// Every call builds a fresh copy, so tests never share state.
import fs from "node:fs";
import path from "node:path";
import type { Animations, Control, Fixture } from "../types";

const repository = path.resolve(__dirname, "../../../..");

// A 1×1 PNG: the second frame of the animated clip.
export const dot =
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==";
const sceneArt = fs.readFileSync(path.join(repository, "docs/design/hue-scene-art.png")).toString("base64");

export function manualFixture(): { fixture: Fixture; animations: Animations } {
  const controls: Control[] = [];
  const find = (id: string): Control => {
    const control = controls.find((c) => c.ID === id);
    if (!control) throw new Error(`fixture has no ${id}`);
    return control;
  };
  function add(ID: string, Label: string, Kind: string, Value: string, extra: Partial<Control> = {}): void {
    const Operations =
      Kind === "selection" || Kind === "text"
        ? ["set"]
        : Kind === "numeric"
          ? ["adjust", "press"]
          : Kind === "status"
            ? []
            : ["press"];
    controls.push({ ID, Label, Kind, Value, Available: true, Revision: 1, Operations, ...extra });
  }

  add("audio.health", "Audio Health", "status", "Healthy");
  add("audio.interface", "Interface", "status", "Universal Audio Volt");
  add("audio.mic-device", "Microphone", "status", "Lavalier", { SurfaceOnly: true });
  add("audio.playback-device", "Playback Device", "status", "Arena", { SurfaceOnly: true });
  for (const [id, label] of [
    ["mic-stack", "Mic Stack"],
    ["mic-mute", "Mic Mute"],
    ["speaker-mute", "Playback Mute"],
  ] as const) {
    add("audio." + id, label, "toggle", id === "mic-stack" ? "On" : "Off");
  }
  for (const id of ["mic", "playback"]) {
    add("audio.gain-" + id, id + " gain", "numeric", "-6.0 dB", {
      Meter: { Present: true, Known: true, DB: -15, At: new Date().toISOString() },
    });
  }
  for (const prefix of ["normal-", "vr-profile-"]) {
    for (const [id, label, value, options] of [
      ["source", "Microphone", "lav", ["auto", "lav", "webcam"]],
      ["mode", "Processing", "direct", ["direct", "element"]],
      ["monitor", "Monitor", "off", ["off", "pre", "post"]],
      ["output", "Playback", "Arena", ["Arena", "Headset"]],
    ] as const) {
      add("audio." + prefix + id, label, "selection", value, {
        Options: [...options],
        OptionLabels: { lav: "Lavalier · Volt", auto: "Automatic" },
        Subdued: prefix === "normal-",
        Status: prefix === "normal-" ? "VR override" : "",
      });
    }
  }
  add("audio.record-mic", "Record Microphone", "toggle", "On", { Group: "Recording" });
  add("audio.record-toggle", "Record", "command", "Stopped", { Group: "Transport", SurfaceOnly: true, Icon: "record-toggle" });
  add("audio.tape-play", "Play Recording", "command", "Playing", { Group: "Transport", SurfaceOnly: true, Icon: "tape-pause" });
  add("audio.tape-stop", "Stop Playback", "command", "", { Group: "Transport", SurfaceOnly: true, Icon: "tape-stop" });
  add("audio.tape-rew", "Rewind", "command", "", { Group: "Transport", SurfaceOnly: true, Icon: "tape-rew", Available: false });
  add("audio.tape-ff", "Fast-forward", "command", "", { Group: "Transport", SurfaceOnly: true, Icon: "tape-ff" });
  add("audio.auto-recover", "Automatic Recovery", "toggle", "On", { Group: "Shared Audio" });
  add("audio.pause-devices", "Disable Device Manipulation", "toggle", "Off", { Group: "Shared Audio" });
  add("audio.pause-sends", "Disable Send Manipulation", "toggle", "On", { Group: "Shared Audio", Status: "3 held" });
  add("audio.engine-restart", "Restart Audio Engine", "command", "", { Group: "Bindings", SurfaceOnly: true });
  add("core.startup", "Start at Login", "toggle", "Off", { Group: "System", Status: "Disabled in Windows startup apps" });
  add("aec.mode", "Echo Cancellation", "selection", "auto", {
    Options: ["auto", "on", "off"],
    OptionLabels: { auto: "Auto", on: "On", off: "Off" },
    Icon: "echo",
  });
  add("aec.engine", "Echo Engine", "selection", "aec3", {
    Options: ["aec3", "localvqe-aec", "localvqe-voice", "localvqe-full"],
    OptionLabels: {
      aec3: "WebRTC AEC3",
      "localvqe-aec": "LocalVQE Echo-Only",
      "localvqe-voice": "LocalVQE Voice Cleanup",
      "localvqe-full": "LocalVQE Full-Band",
    },
    Icon: "echo",
  });
  add("aec.strength", "Echo Strength", "selection", "strong", {
    Options: ["strong", "balanced", "gentle"],
    OptionLabels: { strong: "Strong", balanced: "Balanced", gentle: "Gentle" },
    Icon: "echo",
  });
  add("aec.timing", "Neural Timing", "status", "2.5 / 1.2 ms", { Status: "Peak worker / queue · gaps 3 · underruns 0" });
  add("aec.retry", "Retry Echo Cancellation", "command", "", { Available: false });
  add("aec.status", "Echo Cancellation Status", "status", "Active", { Status: "−32 dB echo · 54 ms", Icon: "echo" });
  add("soundboard.volume", "Volume", "numeric", "0.0 dB");
  add("soundboard.stop", "Stop", "command", "");
  add("soundboard.status", "Soundboard", "status", "Ready");
  for (const name of ["fah", "sadge", "instinct", "airhorn", "ping"]) {
    add("soundboard.clip-" + name, name, "command", "Ready", { Operations: ["press", "hold"] });
  }
  add("hue.status", "Hue", "status", "Not paired", { ViewData: { bridge: "172.16.102.3" } });
  add("hue.pair", "Pair Hue Bridge", "command", "Ready", { ShortLabel: "Pair" });
  add("hue.group", "Hue Room", "selection", "room-1", {
    Options: ["zone-1", "room-1"],
    OptionLabels: { "zone-1": "Desk (zone)", "room-1": "Studio" },
  });
  add("hue.brightness", "Hue Brightness", "numeric", "62%", { ShortLabel: "Brightness" });
  add("hue.motion", "Hue Motion Sensors", "toggle", "On", { ShortLabel: "Motion", Icon: "hue-motion" });
  add("hue.sort", "Hue Scene Sort", "selection", "name", {
    ShortLabel: "Sort",
    Icon: "hue-sort",
    Options: ["name", "hue", "brightness"],
    OptionLabels: { name: "Name", hue: "Hue", brightness: "Brightness" },
  });
  add("hue.rooms", "Hue Rooms", "text", "", {
    ViewData: [
      { ID: "room-1", Name: "Studio", Kind: "room", Chosen: true },
      { ID: "zone-1", Name: "Desk", Kind: "zone", Chosen: true },
    ],
  });
  find("soundboard.clip-sadge").Artwork = sceneArt;
  const animations: Animations = {
    "soundboard.clip-sadge": [
      { Artwork: sceneArt, Delay: 50e6 },
      { Artwork: dot, Delay: 50e6 },
    ],
  };
  for (const [room, name, value] of [
    ["Studio", "Bright", "Ready"],
    ["Studio", "Relax", "Active"],
    ["Studio", "Concentrate", "Ready"],
    ["Desk", "Focus", "Ready"],
    ["Kitchen", "Cook", "Ready"],
  ]) {
    add("hue.scene-" + room!.toLowerCase() + "-" + name!.toLowerCase(), room + " " + name, "command", value!, {
      ShortLabel: name,
      Group: "Hue Scenes",
      Artwork: name === "Relax" ? sceneArt : "",
    });
  }
  add("hue.sync-status", "Hue Sync", "status", "Ready");
  add("hue.sync", "Hue Sync", "toggle", "Off", { ShortLabel: "Sync" });
  add("hue.sync-mode", "Hue Sync Mode", "selection", "video", {
    Options: ["video", "games", "music"],
    OptionLabels: { video: "Video", games: "Games", music: "Music" },
    Available: false,
    Hidden: true,
  });
  add("hue.sync-intensity", "Hue Sync Intensity", "selection", "moderate", {
    Options: ["subtle", "moderate", "high", "extreme"],
    OptionLabels: { subtle: "Subtle", moderate: "Moderate", high: "High", extreme: "Extreme" },
    Available: false,
    Hidden: true,
  });
  for (let n = 1; n <= 12; n++) add("hue.room-scene-" + n, "", "command", "", { Available: false, Group: "Hue Room Scenes" });

  const ago = (seconds: number): string => new Date(Date.now() - seconds * 1000).toISOString();
  const report = (ID: string, Label: string, Group: string, Value: string, Connection: Control["Connection"]): void => {
    controls.push({ ID, Label, Group, Kind: "connection", Value, Available: true, SurfaceOnly: true, Revision: 1, Operations: [], Connection });
  };
  report("hue.app-bridge", "Hue Bridge", "Hue", "Connected", {
    State: "connected",
    Endpoint: "172.16.102.3",
    Since: ago(600),
    LastActivity: ago(1),
    LastError: "event stream ended: EOF",
    LastErrorAt: ago(665),
    Details: [
      { Label: "Bridge ID", Value: "001788fffe2490e0" },
      { Label: "Software", Value: "1978293000" },
    ],
  });
  report("hue.app-sync", "Hue Sync", "Hue", "N/A", {
    State: "disconnected",
    Endpoint: "ws://127.0.0.1:24851/",
    Since: ago(90),
    LastError: "connection refused",
    LastErrorAt: ago(5),
    Details: [],
  });
  report("audio.app-voicemeeter", "Voicemeeter", "Audio", "Disconnected", {
    State: "disconnected",
    Endpoint: "C:\\Program Files (x86)\\VB\\Voicemeeter\\VoicemeeterRemote64.dll",
    Since: ago(30),
    LastActivity: ago(31),
    LastError: "voicemeeter disconnected",
    LastErrorAt: ago(30),
    Details: [
      { Label: "Required", Value: "Yes" },
      { Label: "Edition", Value: "Potato" },
      { Label: "Interval", Value: "1s" },
    ],
  });
  report("audio.app-callback", "Audio Callback Monitor", "Audio", "Off", {
    State: "off",
    Endpoint: "snoofer-audio-monitor.dll",
    Since: "0001-01-01T00:00:00Z",
    LastActivity: "0001-01-01T00:00:00Z",
    Details: [],
  });

  // App audio: Discord pinned first, Chrome heard, Spotify pinned but closed; Voicemeeter hidden by default.
  const appMeter = { Present: true, Known: true, DB: -18, At: new Date().toISOString() };
  [
    ["discord", "Discord", "40%"],
    ["chrome", "Google Chrome", "Mixed"],
    ["spotify", "Spotify", "Closed"],
  ].forEach(([id, label, value], n) => {
    add("appaudio.app-" + id, label!, "numeric", value!, {
      ShortLabel: label,
      Group: "App Audio",
      Collection: "appaudio.apps",
      CollectionLabel: "Apps",
      Order: n + 1,
      Icon: "app-audio",
      Operations: ["press", "adjust", "set"],
      Meter: value === "Closed" ? { Present: true } : appMeter,
    });
  });
  add("appaudio.status", "App Audio", "status", "3 apps", {
    ViewData: {
      RecentMinutes: 5,
      Apps: [
        {
          ID: "appaudio.app-discord",
          Name: "Discord",
          Picked: true,
          Open: true,
          Sessions: 2,
          Executables: ["C:\\Discord\\Discord.exe"],
          Devices: ["Voicemeeter Input"],
          PIDs: [11, 12],
          Naming: [{ Match: "(^|\\\\)updater\\.exe$", Programs: ["updater.exe"] }],
        },
        {
          ID: "appaudio.app-chrome",
          Name: "Google Chrome",
          Open: true,
          Sessions: 3,
          Executables: ["C:\\Chrome\\chrome.exe"],
          Devices: ["Voicemeeter Input"],
          PIDs: [21],
          LastHeard: new Date().toISOString(),
        },
        { ID: "appaudio.app-spotify", Name: "Spotify", Picked: true, Open: false },
        { ID: "appaudio.app-vm", Name: "Voicemeeter", Hidden: true, Open: true, Sessions: 1, Rule: "Excluded (voicemeeter*.exe)" },
        { ID: "appaudio.app-hue", Name: "Hue Sync", Hidden: true, Open: true, Sessions: 1, Rule: "Excluded (huesync.exe)" },
        { ID: "appaudio.app-rule", Name: "Launcher", Hidden: true, Open: true, Sessions: 1, Rule: "Hidden (launcher)" },
      ],
      Exclude: ["snoofer.exe", "voicemeeter*.exe", "audiodg.exe", "huesync.exe"],
    },
  });
  add("appaudio.edit", "App Audio Edit", "text", "abc");

  // Routing: one interface, playback with an ambiguous entry, a webcam and the mic priority.
  const suggestion = (Name: string, Driver: string, Exact: string, Device: string, ID = "") => ({ Name, Driver, Exact, Device, ID });
  add("audio.priorities", "Device Priorities", "status", "", {
    Group: "Routing",
    ViewData: {
      Lists: {
        interfaces: [
          {
            ASIOPattern: "(?i)^Universal Audio Volt$",
            PresencePattern: "(?i)^INPUT 1/2 \\(Volt 2\\)$",
            Inputs: { desk: [1], lav: [2] },
            Matches: ["INPUT 1/2 (Volt 2)"],
            Drivers: ["Universal Audio Volt"],
            InUse: true,
          },
        ],
        playback: [
          { Driver: "wdm", DeviceID: "{airpods}", DeviceName: "Headphones (AirPods)", Matches: ["Headphones (AirPods Pro)"], InUse: true },
          { Driver: "wdm", Pattern: "(?i)steelseries.*arena", Matches: ["Speakers (Arena 7)", "Game (Arena 7)"] },
          { Driver: "asio", Pattern: "(?i)^Universal Audio Volt$", Matches: ["Universal Audio Volt"] },
        ],
        "mic-devices:webcam": [{ Driver: "wdm", Pattern: "(?i)insta360.*link.*2", Matches: [] }],
        microphones: [
          { ID: "lav", Option: true, InUse: true },
          { ID: "webcam", Option: false },
        ],
      },
      Suggestions: {
        playback: [suggestion("Speakers (Realtek Audio)", "wdm", "(?i)^Speakers \\(Realtek Audio\\)$", "(?i)Realtek Audio", "{realtek}")],
        "mic-devices:": [suggestion("Microphone (Realtek Audio)", "wdm", "(?i)^Microphone \\(Realtek Audio\\)$", "(?i)Realtek Audio", "{realtek-mic}")],
      },
      Drivers: [suggestion("Focusrite USB ASIO", "asio", "(?i)^Focusrite USB ASIO$", "", "{focusrite}")],
      Inputs: [suggestion("Analogue 1 + 2 (Focusrite USB)", "wdm", "(?i)^Analogue 1 \\+ 2 \\(Focusrite USB\\)$", "(?i)Focusrite USB")],
      Microphones: ["desk", "lav", "webcam", "off"],
      Mics: [
        { ID: "desk", Name: "Desk", Silent: true },
        { ID: "lav", Name: "Lavalier", InUse: true, Option: true, Ready: true },
        { ID: "webcam", Name: "Webcam", Device: true },
      ],
      Profiles: true,
      Activity: { Check: 2, SilenceDB: -70, SilentAfterS: 10 },
      // Outputs: Playback plus a Music slot; positions 2 and 3 are hidden.
      OutputDevices: [suggestion("Speakers (Realtek Audio)", "wdm", "", "", "{realtek}"), suggestion("Speakers (Arena 7)", "wdm", "", "")],
    },
  });
  add("audio.priority-edit", "Device Priority Edit", "text", "abc", { Group: "Routing" });
  add("audio.playback:virtual:1", "virtual:1 playback", "toggle", "On", { Group: "Shared Audio" });
  add("audio.monitor", "Monitor", "selection", "pre", {
    Group: "Bindings",
    Options: ["off", "pre", "post"],
    OptionLabels: { off: "Off", pre: "Pre", post: "Post" },
    Operations: ["set", "press"],
    SurfaceOnly: true,
  });
  add("audio.output-edit", "Output Edit", "text", "abc", { Group: "Routing" });
  add("audio.slot-1", "Music", "status", "In use", { Group: "Routing", Status: "Speakers (Arena 7)", ViewData: { ID: "music", Bus: "A3" } });
  for (const [source, value, available] of [
    ["virtual:1", "On", true],
    ["virtual:2", "Off", false],
    ["virtual:3", "Off", false],
    ["monitor", "Off", true],
    ["soundboard", "Off", true],
    ["tape", "On", true],
  ] as const) {
    add("audio.slot-1:" + source, "Music " + source, "toggle", value, { Group: "Routing", Available: available, Hidden: !available });
  }
  for (const n of [2, 3]) {
    add("audio.slot-" + n, "Output " + n, "status", "", { Group: "Routing", Available: false, Hidden: true });
    for (const source of ["virtual:1", "monitor", "soundboard", "tape"]) {
      add("audio.slot-" + n + ":" + source, "Output " + n + " " + source, "toggle", "Off", { Group: "Routing", Available: false, Hidden: true });
    }
  }

  // Now playing: a focused YouTube tab and a paused Windows player.
  const session = { Group: "Now Playing", Collection: "nowplaying.sessions", CollectionLabel: "Media Sessions", Icon: "media-play", Operations: ["press", "set"] };
  add("nowplaying.s-tab", "Video A", "command", "Playing", { ...session, ShortLabel: "Video A", Order: 1 });
  add("nowplaying.s-spotify", "Song", "command", "Paused", { ...session, ShortLabel: "Song", Order: 2 });
  for (const [id, label] of [
    ["prev", "Previous Track"],
    ["next", "Next Track"],
    ["toggle", "Play or Pause"],
    ["mute", "Mute Tab"],
  ]) {
    add("nowplaying." + id, label!, "command", "");
  }
  add("nowplaying.dial", "Now Playing", "numeric", "1:05 / 4:45", { Operations: ["adjust", "press"] });
  add("nowplaying.status", "Now Playing", "status", "2 sessions", {
    ViewData: {
      Windows: "Connected",
      Bridge: { Port: 47815, Error: "", Refused: "Update the Snoofer Media extension in Firefox" },
      Browsers: [{ Name: "Brave", Version: "1.0.0", Sessions: 1 }],
      Sessions: [
        {
          ID: "nowplaying.s-tab",
          Source: "Brave",
          App: "Brave · youtube.com",
          Title: "Video A",
          Artist: "Channel",
          Status: "Playing",
          PositionMs: 65000,
          AtMs: Date.now(),
          Rate: 1,
          DurationMs: 285000,
          Focused: true,
          CanToggle: true,
          CanNext: true,
          CanSeek: true,
          CanMute: true,
        },
        {
          ID: "nowplaying.s-spotify",
          Source: "windows",
          App: "Spotify",
          Title: "Song",
          Artist: "Band",
          Album: "Album",
          Status: "Paused",
          PositionMs: 0,
          DurationMs: 200000,
          CanToggle: true,
          CanNext: true,
          CanPrev: true,
          CanSeek: true,
        },
      ],
    },
  });
  add("discord.channel", "Discord Channel", "status", "General", { Group: "Discord" });
  for (const [id, label, value] of [
    ["mute", "Discord Mute", "On"],
    ["deafen", "Discord Deafen", "Off"],
    ["video", "Discord Camera", "Off"],
    ["screenshare", "Discord Screen Share", "Off"],
  ]) {
    add("discord." + id, label!, "toggle", value!, { Group: "Discord" });
  }
  add("discord.leave", "Leave Discord Call", "command", "", { Group: "Discord" });
  add("discord.connect", "Connect Discord", "command", "", { Group: "Discord", Available: false });

  const fixture: Fixture = {
    Controls: controls,
    Plugins: {
      audio: "Running",
      soundboard: "Running",
      streamdeck: "Running",
      vr: "Running",
      media: "Disabled",
      hue: "Running",
      appaudio: "Running",
      nowplaying: "Running",
      discord: "Running",
    },
    Enabled: {
      discord: true,
      audio: true,
      soundboard: true,
      streamdeck: true,
      vr: true,
      media: false,
      hue: true,
      appaudio: true,
      nowplaying: true,
    },
    Notice: "",
    Confirmation: "",
  };
  return { fixture, animations };
}
