// The host's view state as the GUI receives it (app/state.go, snoofer.Control).
// Handwritten fixtures may omit default fields; provider ViewData is narrowed at each read.

export interface Meter {
  Present?: boolean;
  Known?: boolean;
  DB?: number;
  At?: string;
}

export interface Connection {
  State: string;
  Endpoint?: string;
  Since?: string;
  LastActivity?: string;
  LastError?: string;
  LastErrorAt?: string;
  Details?: { Label: string; Value: string }[];
}

export interface Control<T = unknown> {
  ID: string;
  Label: string;
  Kind: string;
  Value: string;
  Available: boolean;
  Revision: number;
  Operations: string[];
  Group?: string;
  Status?: string;
  ShortLabel?: string;
  Icon?: string;
  Hidden?: boolean;
  Subdued?: boolean;
  SurfaceOnly?: boolean;
  Options?: string[];
  OptionLabels?: Record<string, string>;
  Meter?: Meter;
  Artwork?: string;
  Collection?: string;
  CollectionLabel?: string;
  Order?: number;
  Connection?: Connection;
  ViewData?: T;
}

export interface Fixture {
  Controls: Control[];
  Plugins: Record<string, string>;
  PluginLabels?: Record<string, string>;
  Enabled: Record<string, boolean>;
  Notice: string;
  Confirmation: string;
}

export type Animations = Record<string, { Artwork: string; Delay: number }[]>;

export interface Request {
  ID: string;
  Revision: number;
  Operation: string;
  Value: string;
  Delta: number;
}

export interface Action {
  Request?: Request;
  Kind?: string;
  Plugin?: string;
  Enable?: boolean;
}
