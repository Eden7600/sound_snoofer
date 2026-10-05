// Package soundboard plays local MP3 clips through a dedicated Voicemeeter input.
package soundboard

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type clip struct {
	Artwork, ArtworkError string
	ID, Label, Path       string
	Size                  int64
	Modified              time.Time
}

func catalogue(folder string) ([]clip, error) {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil, fmt.Errorf("read soundboard folder: %w", err)
	}
	images := map[string]string{}
	for _, entry := range entries {
		if entry.Type()&(os.ModeSymlink|os.ModeDir) == 0 {
			images[strings.ToLower(entry.Name())] = entry.Name()
		}
	}
	clips := []clip{}
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 || !strings.EqualFold(filepath.Ext(entry.Name()), ".mp3") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		id := fmt.Sprintf("soundboard.clip-%x", sha256.Sum256([]byte(entry.Name())))
		label := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		artwork, artworkErr := clipArtwork(folder, label, images)
		clips = append(clips, clip{Artwork: artwork, ArtworkError: artworkErr, ID: id, Label: strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())), Path: filepath.Join(folder, entry.Name()), Size: info.Size(), Modified: info.ModTime()})
	}
	slices.SortFunc(clips, func(a, b clip) int { return strings.Compare(a.Label, b.Label) })
	return clips, nil
}
func (c clip) unchanged() error {
	info, err := os.Lstat(c.Path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() != c.Size || !info.ModTime().Equal(c.Modified) {
		return fmt.Errorf("clip changed; wait for refresh")
	}
	return nil
}
