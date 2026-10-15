// Wire shapes from plugins/audio/priorities.go and outputs.go.
export interface PriorityEntry {
  Driver?: string;
  Pattern?: string;
  ASIOPattern?: string;
  PresencePattern?: string;
  DeviceID?: string;
  DeviceName?: string;
  ASIOID?: string;
  ASIOName?: string;
  PresenceID?: string;
  PresenceName?: string;
  Inputs?: Record<string, number[] | null>;
  ID?: string;
  Matches: string[] | null;
  Drivers?: string[];
  InUse: boolean;
  Option?: boolean;
}

export interface PrioritySuggestion {
  Name: string;
  Driver: string;
  ID?: string;
  Exact: string;
  Device: string;
}

export interface MicView {
  ID: string;
  Name: string;
  Device: boolean;
  InUse: boolean;
  Option: boolean;
  Ready: boolean;
  Silent: boolean;
}

export interface Activity {
  check: number;
  silence_db?: number;
  silent_after_s?: number;
}

export interface PriorityView {
  Lists: Record<string, PriorityEntry[] | null> | null;
  Suggestions: Record<string, PrioritySuggestion[] | null> | null;
  Drivers: PrioritySuggestion[] | null;
  Inputs: PrioritySuggestion[] | null;
  Microphones: string[] | null;
  Mics: MicView[] | null;
  OutputDevices: PrioritySuggestion[] | null;
  Profiles: boolean;
  Activity?: Activity;
}

export interface OutputSlotView {
  ID: string;
  Bus: string;
}
