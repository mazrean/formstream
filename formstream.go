package formstream

import (
	"io"
	"mime"
	"net/textproto"
)

type Parser struct {
	boundary string
	valueMap map[string][]Value
	hookMap  map[string]streamHook
	parserConfig
}

func NewParser(boundary string, options ...ParserOption) *Parser {
	c := parserConfig{
		maxParts:       defaultMaxParts,
		maxHeaders:     defaultMaxHeaders,
		maxMemSize:     defaultMaxMemSize,
		maxMemFileSize: defaultMaxMemFileSize,
	}
	for _, opt := range options {
		opt(&c)
	}

	return &Parser{
		boundary:     boundary,
		valueMap:     make(map[string][]Value),
		hookMap:      make(map[string]streamHook),
		parserConfig: c,
	}
}

type parserConfig struct {
	maxParts       uint
	maxHeaders     uint
	maxMemSize     DataSize
	maxMemFileSize DataSize
}

type ParserOption func(*parserConfig)

type DataSize int64

const (
	_ DataSize = 1 << (iota * 10)
	KB
	MB
	GB
)

const (
	defaultMaxParts       = 10000
	defaultMaxHeaders     = 10000
	defaultMaxMemSize     = 32 * MB
	defaultMaxMemFileSize = 32 * MB
)

func WithMaxParts(maxParts uint) ParserOption {
	return func(c *parserConfig) {
		c.maxParts = maxParts
	}
}

func WithMaxHeaders(maxHeaders uint) ParserOption {
	return func(c *parserConfig) {
		c.maxHeaders = maxHeaders
	}
}

func WithMaxMemSize(maxMemSize DataSize) ParserOption {
	return func(c *parserConfig) {
		c.maxMemSize = maxMemSize
	}
}

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
