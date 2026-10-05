package soundboard

import (
	"context"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNormalizationCacheAndInvalidation(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "clip.mp3")
	if err := os.WriteFile(source, []byte("source"), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(source)
	if err != nil {
		t.Fatal(err)
	}
	c := clip{Path: source, Size: info.Size(), Modified: info.ModTime()}
	folder := filepath.Join(root, "cache")
	calls := 0
	decode := func(ctx context.Context, source, destination string) error {
		calls++
		data := make([]byte, 46)
		copy(data, "RIFF")
		copy(data[8:], "WAVEfmt ")
		binary.LittleEndian.PutUint32(data[4:], uint32(len(data)-8))
		binary.LittleEndian.PutUint32(data[16:], 16)
		binary.LittleEndian.PutUint16(data[20:], 1)
		binary.LittleEndian.PutUint16(data[22:], 1)
		binary.LittleEndian.PutUint32(data[24:], 48000)
		binary.LittleEndian.PutUint16(data[34:], 16)
		binary.LittleEndian.PutUint32(data[40:], uint32(len(data)-44))
		copy(data[36:], "data")
		return os.WriteFile(destination, data, 0600)
	}
	first, err := prepareClip(context.Background(), c, folder, decode)
	if err != nil {
		t.Fatal(err)
	}
	again, err := prepareClip(context.Background(), c, folder, decode)
	if err != nil || first != again || calls != 1 {
		t.Fatal("cache not reused", err, calls)
	}
	if err := os.WriteFile(source, []byte("new source"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareClip(context.Background(), c, folder, decode); err == nil {
		t.Fatal("changed source accepted")
	}
	info, err = os.Stat(source)
	if err != nil {
		t.Fatal(err)
	}
	c.Size, c.Modified = info.Size(), info.ModTime()
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := prepareClip(cancelled, c, folder, decode); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	fail := func(context.Context, string, string) error { return errors.New("decode failed") }
	if _, err := prepareClip(context.Background(), c, folder, fail); err == nil {
		t.Fatal("failure accepted")
	}
	entries, err := os.ReadDir(folder)
	if err != nil || len(entries) != 1 {
		t.Fatal("partial output retained", entries, err)
	}
	if err := pruneNormalized(folder, []clip{c}, first); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(first); err != nil {
		t.Fatal("playing cache deleted")
	}
	if err := pruneNormalized(folder, []clip{c}, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(first); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("obsolete cache retained")
	}
}

func TestPreparationReplacesAndStopsWithoutLatePlayback(t *testing.T) {
	p := &preparation{}
	first, second := &clip{ID: "first"}, &clip{ID: "second"}
	started := make(chan string, 2)
	prepare := func(ctx context.Context, c clip) (string, error) {
		started <- c.ID
		<-ctx.Done()
		return "", ctx.Err()
	}
	p.replace(first)
	p.start(context.Background(), prepare)
	if <-started != "first" {
		t.Fatal("wrong job")
	}
	oldDone := p.done
	p.replace(second)
	p.start(context.Background(), prepare)
	if p.done != oldDone {
		t.Fatal("parallel decoder started")
	}
	select {
	case result := <-p.done:
		if p.finish(result) {
			t.Fatal("superseded clip would play")
		}
	case <-time.After(time.Second):
		t.Fatal("decode not cancelled")
	}
	p.start(context.Background(), prepare)
	if <-started != "second" {
		t.Fatal("replacement lost")
	}
	p.replace(nil)
	select {
	case result := <-p.done:
		if p.finish(result) {
			t.Fatal("stopped clip would play")
		}
	case <-time.After(time.Second):
		t.Fatal("stop not responsive")
	}
	if p.done != nil || p.requested != nil {
		t.Fatal("work retained")
	}
}
