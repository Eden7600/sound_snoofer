// plugins/appaudio/plugin.go appView/statusView and rules.go namingView.
export interface AppAudioApp {
  ID: string;
  Name: string;
  Rule: string;
  Picked: boolean;
  Hidden: boolean;
  Open: boolean;
  Naming?: { Match: string; Programs: string[] | null }[];
  Executables: string[] | null;
  Devices: string[] | null;
  PIDs: number[] | null;
  Sessions: number;
  LastHeard?: string;
}

export interface AppAudioView {
  Apps: AppAudioApp[] | null;
  Exclude: string[] | null;
  RecentMinutes: number;
}
