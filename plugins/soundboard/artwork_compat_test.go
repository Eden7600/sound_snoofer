package soundboard

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

// One-pixel WebP checks content detection independently of a filename suffix.
func TestWebPWithPNGExtension(t *testing.T) {
	data, err := base64.StdEncoding.DecodeString("UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEADsD+JaQAA3AAAAAA")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "clip.png")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	artwork, err := loadArtwork(path)
	if err != nil || artwork == "" {
		t.Fatalf("WebP named PNG rejected: %v", err)
	}
}

func TestReportedArtwork(t *testing.T) {
	folder := os.Getenv("SNOOFER_ARTWORK_TEST_FOLDER")
	if folder == "" {
		t.Skip("set SNOOFER_ARTWORK_TEST_FOLDER to check reported files")
	}
	clips, err := catalogue(folder)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"fah", "sadge"} {
		found := false
		for _, c := range clips {
			if c.Label != name {
				continue
			}
			found = true
			if c.Artwork == "" || c.ArtworkError != "" {
				t.Fatalf("%s: %s", name, c.ArtworkError)
			}
			t.Logf("%s: decoded and catalogued (%d bytes)", name, len(c.Artwork))
			if out := os.Getenv("SNOOFER_ARTWORK_PREVIEW_DIR"); out != "" {
				data, err := base64.StdEncoding.DecodeString(c.Artwork)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(out, name+".png"), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
		}
		if !found {
			t.Fatalf("%s clip missing", name)
		}
	}
}
