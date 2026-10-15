// plugins/streamdeck/editor_view.go; editorView always initializes Keys and Dials.
export interface DeckView {
  Keys: DeckSlot[];
  Dials: DeckSlot[];
  Regions: DeckRegion[] | null;
  Collections: DeckCollection[] | null;
  Selected: number;
  Dirty: boolean;
  Home: boolean;
  Sets: number;
  Scrolls: boolean;
  Model: string;
  Columns: number;
  Reserved: number;
  NavDial: boolean;
  Decks: DeckDevice[] | null;
}

export interface DeckSlot {
  Control: string;
  Label: string;
  Source: string;
  Region: number;
}

export interface DeckCollection {
  ID: string;
  Label: string;
}

export interface DeckRegion {
  Source: string;
  Label: string;
  Stacked: string;
  First: number;
  Last: number;
  Legacy: boolean;
  Dials: boolean;
  Clip: boolean;
}

export interface DeckDevice {
  Serial: string;
  Name: string;
  Model: string;
  Connected: boolean;
  Own: boolean;
}
