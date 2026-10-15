// plugins/nowplaying/publish.go viewSession/statusView.
export interface MediaSession {
  ID: string;
  Source: string;
  App: string;
  Title: string;
  Artist: string;
  Album: string;
  Status: string;
  Progress: string;
  PositionMs: number;
  DurationMs: number;
  AtMs: number;
  Rate: number;
  Focused: boolean;
  CanToggle: boolean;
  CanNext: boolean;
  CanPrev: boolean;
  CanSeek: boolean;
  CanMute: boolean;
  Muted: boolean;
  Pending: boolean;
  Failure: string;
}

export interface MediaView {
  Bridge: { Port: number; Error: string; Refused: string };
  Windows: string;
  Browsers: { Name: string; Version: string; Sessions: number }[] | null;
  Sessions: MediaSession[] | null;
}
