package cli

import (
	"context"
	"io"
)

// checkedWriter preserves the first output failure and stops a watch or apply.
type checkedWriter struct {
	writer io.Writer
	cancel context.CancelFunc
	err    error
}

func (w *checkedWriter) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	n, e := w.writer.Write(p)
	if e == nil && n != len(p) {
		e = io.ErrShortWrite
	}
	if e != nil {
		w.err = e
		w.cancel()
	}
	return n, e
}
