package soundboard

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCatalogueAndChangedClip(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"b.MP3", "a.mp3", "ignore.wav"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "folder.mp3"), 0700); err != nil {
		t.Fatal(err)
	}
	clips, err := catalogue(dir, nil)
	if err != nil || len(clips) != 2 || clips[0].Label != "a" {
		t.Fatalf("%v %v", clips, err)
	}
	id := clips[0].ID
	if err := clips[0].unchanged(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(clips[0].Path, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if clips[0].unchanged() == nil {
		t.Fatal("changed clip accepted")
	}
	next, err := catalogue(dir, nil)
	if err != nil || next[0].ID != id {
		t.Fatal("identity changed")
	}
	if _, err := catalogue(filepath.Join(dir, "missing"), nil); err == nil {
		t.Fatal("missing folder accepted")
	}
}
