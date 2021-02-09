package wireguard

import (
	"io"
)

// Enable calling Write() multiple times in succession, without error checking.
// MultiWriter does not write more after an error occurred.
type multiWriter struct {
	w       io.Writer
	written int
	err     error
}

func (w multiWriter) Write(data []byte) (int, error) {
	var wt int
	if w.err == nil {
		wt, w.err = w.w.Write(data)
		w.written += wt
	}
	return w.Status()
}

func (w multiWriter) Status() (int, error) {
	return w.written, w.err
}
