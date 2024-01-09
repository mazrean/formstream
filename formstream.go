package formstream

import (
	"io"
	"net/textproto"
)

type Parser struct {
	boundary       string
	maxMemFileSize DataSize
	Value          map[string][]*Value
	valueHeaderMap map[string][]*FileHeader
	hookMap        map[string]streamHook
}

func NewParser(options ...ParserOption) *Parser {
	c := &parserConfig{
		boundary: "",
	}
	for _, opt := range options {
		opt(c)
	}

	return &Parser{
		boundary:       c.boundary,
		maxMemFileSize: c.maxMemFileSize,
		Value:          make(map[string][]*Value),
		hookMap:        make(map[string]streamHook),
	}
}

type parserConfig struct {
	boundary       string
	maxMemFileSize DataSize
}

type ParserOption func(*parserConfig)

func WithBoundary(boundary string) ParserOption {
	return func(c *parserConfig) {
		c.boundary = boundary
	}
}

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
	Body []byte
	*FileHeader
}

// String returns the value as a string.
// Note: This function ignores the charset. If you need to consider the charset, convert to string by yourself.
func (v *Value) String() string {
	return string(v.Body)
}

type FileHeader struct {
	FileName string
	Header   textproto.MIMEHeader
}

type StreamHookFunc = func(r io.Reader, fileName string, header textproto.MIMEHeader) error

type streamHook struct {
	fn           StreamHookFunc
	requireParts []string
}
