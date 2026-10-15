// plugins/hue/plugin.go statusView and rooms.go roomChoice.
export interface HueStatusView {
  bridge?: string;
  address?: string;
}

export interface HueRoom {
  ID: string;
  Name: string;
  Kind: string;
  Chosen: boolean;
}
