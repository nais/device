package outtune

import (
	"io"
)

type errorWriter struct {
	w   io.Writer
	err error
}

func (e *errorWriter) Write(p []byte) (int, error) {
	var n int
	if e.err != nil {
		return 0, e.err
	}
	n, e.err = e.w.Write(p)
	return n, e.err
}

func (e *errorWriter) Error() error {
	return e.err
}
