package myio

import (
	"io"
	"time"
)

type slowWriter struct{}

func SlowWriter() io.Writer {
	return &slowWriter{}
}

func (w *slowWriter) Write(p []byte) (n int, err error) {
	time.Sleep(time.Duration(len(p)) * 50 * time.Nanosecond)
	return len(p), nil
}
