package myio

import "io"

type nopSeekCloser struct {
	io.ReadSeeker
}

func NopSeekCloser(r io.ReadSeeker) io.ReadSeekCloser {
	return nopSeekCloser{r}
}

func (nopSeekCloser) Close() error { return nil }
