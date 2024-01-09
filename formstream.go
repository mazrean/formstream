package formstream

import (
	"io"
	"mime"
	"net/textproto"
)

type Parser struct {
	boundary       string
	maxMemFileSize DataSize
	valueMap       map[string][]Value
	hookMap        map[string]streamHook
}

func NewParser(boundary string, options ...ParserOption) *Parser {
	c := &parserConfig{
		maxMemFileSize: 32 * MB,
	}
	for _, opt := range options {
		opt(c)
	}

	return &Parser{
		boundary:       boundary,
		maxMemFileSize: c.maxMemFileSize,
		valueMap:       make(map[string][]Value),
		hookMap:        make(map[string]streamHook),
	}
}

type parserConfig struct {
	maxMemFileSize DataSize
}

type ParserOption func(*parserConfig)

type DataSize uint64

const (
	_ DataSize = 1 << (iota * 10)
	KB
	MB
	GB
)

func WithMaxMemFileSize(maxMemFileSize DataSize) ParserOption {
	return func(c *parserConfig) {
		c.maxMemFileSize = maxMemFileSize
	}
}

type Value struct {
	content []byte
	header  Header
}

func (v Value) Unwrap() (string, Header) {
	return string(v.content), v.header
}

func (v Value) UnwrapRaw() ([]byte, Header) {
	return v.content, v.header
}

type Header struct {
	dispositionParams map[string]string
	header            textproto.MIMEHeader
}

func newHeader(h textproto.MIMEHeader) Header {
	contentDisposition := h.Get("Content-Disposition")
	_, params, err := mime.ParseMediaType(contentDisposition)
	if err != nil {
		params = make(map[string]string)
	}

	return Header{
		dispositionParams: params,
		header:            h,
	}
}

func (h Header) Get(key string) string {
	return h.header.Get(key)
}

func (h Header) ContentType() string {
	return h.header.Get("Content-Type")
}

func (h Header) Name() string {
	return h.dispositionParams["name"]
}

func (h Header) FileName() string {
	return h.dispositionParams["filename"]
}

type StreamHookFunc = func(r io.Reader, header Header) error

type streamHook struct {
	fn           StreamHookFunc
	requireParts []string
}
