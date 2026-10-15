// JSON wire fields from app/state.go and snoofer/controls.go; provider ViewData is narrowed by its consumer.
export interface Meter {
  Present: boolean;
  Known: boolean;
  DB: number;
  At: string;
}

export interface Progress {
  Known: boolean;
  Playing: boolean;
  PositionMs: number;
  DurationMs: number;
  Rate: number;
  At: string;
}

export interface Connection {
  State: string;
  Endpoint: string;
  Since: string;
  LastActivity: string;
  LastError: string;
  LastErrorAt: string;
  Details: { Label: string; Value: string }[] | null;
}

export interface Control<T = unknown> {
  ViewData?: T;
  Artwork: string;
  ShortLabel: string;
  Collection: string;
  CollectionLabel: string;
  Mirrors: string;
  Order: number;
  Meter: Meter;
  Progress: Progress;
  Timers?: { Control: string; Ends: string }[];
  Connection?: Connection;
  SurfaceOnly: boolean;
  Hidden: boolean;
  Inactive: boolean;
  Brightness?: number;
  OptionLabels: Record<string, string> | null;
  EnterOnly: boolean;
  Epoch: number;
  Subdued: boolean;
  ID: string;
  Label: string;
  Group: string;
  Kind: string;
  Value: string;
  Status: string;
  Icon: string;
  Options: string[] | null;
  Operations: string[] | null;
  Available: boolean;
  Revision: number;
}

export interface ViewState {
  Controls: Control[] | null;
  Plugins: Record<string, string> | null;
  Enabled: Record<string, boolean> | null;
  Notice: string;
  Confirmation: string;
}

export interface Request {
  ID: string;
  Revision: number;
  Operation: string;
  Value: string;
  Delta: number;
}

export interface UIAction {
  Request?: Request;
  Kind?: string;
  Plugin?: string;
  Enable?: boolean;
}

export interface ArtworkFrame {
  Artwork: string;
  Delay: number; // Go time.Duration in nanoseconds.
}

export interface Desktop {
  State(): Promise<ViewState | null>;
  Send(action: UIAction): Promise<void>;
  Animation(id: string): Promise<ArtworkFrame[] | null>;
}

declare global {
  interface Window {
    go?: { app?: { Desktop?: Desktop } };
  }
}

export type WidgetInput = HTMLButtonElement | HTMLInputElement | HTMLSelectElement | HTMLDivElement | HTMLSpanElement;
export interface Widget {
  id: string;
  row: HTMLElement;
  title: HTMLLabelElement;
  note: HTMLElement;
  input: WidgetInput;
  initial: Control;
}

export interface Pending {
  id?: string;
  revision?: number;
  notice: string;
  until: number;
}
