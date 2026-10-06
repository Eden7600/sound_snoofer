package soundboard

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func normalizedName(c clip) string {
	return fmt.Sprintf("%x.wav", sha256.Sum256([]byte(fmt.Sprintf("peak-v1\x00%s\x00%d\x00%d", c.Path, c.Size, c.Modified.UnixNano()))))
}

func prepareClip(ctx context.Context, c clip, folder string, decode func(context.Context, string, string) error) (path string, err error) {
	if err = ctx.Err(); err != nil {
		return "", err
	}
	if err = c.unchanged(); err != nil {
		return "", err
	}
	path = filepath.Join(folder, normalizedName(c))
	if validNormalized(path) {
		return path, nil
	}
	if err = os.MkdirAll(folder, 0700); err != nil {
		return "", err
	}
	temp, err := os.CreateTemp(folder, "prepare-*.wav")
	if err != nil {
		return "", err
	}
	temporary := temp.Name()
	if err = temp.Close(); err != nil {
		os.Remove(temporary)
		return "", err
	}
	defer func() {
		if cleanup := os.Remove(temporary); cleanup != nil && !errors.Is(cleanup, os.ErrNotExist) {
			err = errors.Join(err, fmt.Errorf("remove partial normalization: %w", cleanup))
		}
	}()
	if err = decode(ctx, c.Path, temporary); err != nil {
		return "", fmt.Errorf("normalize %s: %w", c.Label, err)
	}
	if err = ctx.Err(); err != nil {
		return "", err
	}
	if err = c.unchanged(); err != nil {
		return "", err
	}
	if !validNormalized(temporary) {
		return "", fmt.Errorf("normalization produced invalid PCM")
	}
	if err = os.Rename(temporary, path); err != nil {
		return "", err
	}
	return path, nil
}

func validNormalized(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	header := make([]byte, 44)
	_, readErr := io.ReadFull(file, header)
	stat, statErr := file.Stat()
	closeErr := file.Close()
	if readErr != nil || statErr != nil || closeErr != nil {
		return false
	}
	channels := binary.LittleEndian.Uint16(header[22:])
	rate := binary.LittleEndian.Uint32(header[24:])
	return stat.Size() > 44 && stat.Size() <= 64*1024*1024+44 &&
		string(header[:4]) == "RIFF" && string(header[8:16]) == "WAVEfmt " && string(header[36:40]) == "data" &&
		int64(binary.LittleEndian.Uint32(header[4:])) == stat.Size()-8 && int64(binary.LittleEndian.Uint32(header[40:])) == stat.Size()-44 &&
		binary.LittleEndian.Uint32(header[16:]) == 16 && binary.LittleEndian.Uint16(header[20:]) == 1 &&
		(channels == 1 || channels == 2) && binary.LittleEndian.Uint16(header[34:]) == 16 &&
		rate >= 8000 && rate <= 192000 && (stat.Size()-44)%int64(channels*2) == 0
}

// wavLength is the play time of a normalized 16-bit PCM WAV, or false when the
// file is not one.
func wavLength(path string) (time.Duration, bool) {
	if !validNormalized(path) {
		return 0, false
	}
	file, err := os.Open(path)
	if err != nil {
		return 0, false
	}
	header := make([]byte, 44)
	_, readErr := io.ReadFull(file, header)
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return 0, false
	}
	channels := int64(binary.LittleEndian.Uint16(header[22:]))
	rate := int64(binary.LittleEndian.Uint32(header[24:]))
	samples := int64(binary.LittleEndian.Uint32(header[40:])) / (channels * 2)
	return time.Duration(samples) * time.Second / time.Duration(rate), true
}

// Prune only our hash-named generated WAVs; never source files or arbitrary contents.
func pruneNormalized(folder string, clips []clip, playing []string) error {
	entries, err := os.ReadDir(folder)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	keep := map[string]bool{}
	for _, path := range playing {
		keep[filepath.Base(path)] = true
	}
	for _, c := range clips {
		keep[normalizedName(c)] = true
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || keep[name] || len(name) != 68 || !strings.HasSuffix(name, ".wav") {
			continue
		}
		if _, err := hex.DecodeString(strings.TrimSuffix(name, ".wav")); err != nil {
			continue
		}
		if err := os.Remove(filepath.Join(folder, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

type normalizedClip struct {
	clip *clip
	path string
	err  error
}

// preparation retains only the latest request and never runs two decoders.
type preparation struct {
	requested *clip
	done      chan normalizedClip
	cancel    context.CancelFunc
}

func (p *preparation) replace(c *clip) {
	p.requested = c
	if p.cancel != nil {
		p.cancel()
	}
}
func (p *preparation) start(ctx context.Context, prepare func(context.Context, clip) (string, error)) {
	if p.requested == nil || p.done != nil {
		return
	}
	runCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.done = make(chan normalizedClip, 1)
	c, done := p.requested, p.done
	go func() { path, err := prepare(runCtx, *c); done <- normalizedClip{c, path, err} }()
}
func (p *preparation) finish(result normalizedClip) bool {
	p.cancel()
	p.cancel = nil
	p.done = nil
	if result.clip != p.requested {
		return false
	}
	p.requested = nil
	return true
}
func (p *preparation) close() error {
	p.replace(nil)
	if p.done == nil {
		return nil
	}
	result := <-p.done
	p.finish(result)
	if errors.Is(result.err, context.Canceled) {
		return nil
	}
	return result.err
}
