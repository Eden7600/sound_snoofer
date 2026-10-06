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

	"sound-snoofer/snoofer"
)

type clip struct {
	Artwork, ArtworkError string
	Animation             []snoofer.ArtworkFrame
	ID, Label, Path       string
	Size                  int64
	Modified              time.Time
}

// catalogue lists the folder's clips. Artwork comes from cache while its file
// is unchanged; entries for images no longer used are dropped. A nil cache
// decodes every image.
func catalogue(folder string, cache artworkCache) ([]clip, error) {
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
	used := map[string]bool{}
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
		art := clipArtwork(folder, label, images, cache)
		used[art.path] = true
		clips = append(clips, clip{Artwork: art.Image, ArtworkError: art.Err, Animation: art.Animation, ID: id, Label: strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())), Path: filepath.Join(folder, entry.Name()), Size: info.Size(), Modified: info.ModTime()})
	}
	for path := range cache {
		if !used[path] {
			delete(cache, path)
		}
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
