package ioconvenience

import (
	"io"
)

// Enable calling Write() multiple times in succession, without error checking.
// ErrorWriter does not write more after an error occurred.
type ErrorWriter struct {
	w       io.Writer
	written int
	err     error
}

func NewErrorWriter(w io.Writer) *ErrorWriter {
	return &ErrorWriter{
		w: w,
	}
}

func (w ErrorWriter) Write(data []byte) (int, error) {
	var wt int

	if w.err == nil {
		wt, w.err = w.w.Write(data)
		w.written += wt
	}

	return w.Status()
}

func (w ErrorWriter) Status() (int, error) {
	return w.written, w.err
}
