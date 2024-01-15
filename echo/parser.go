package echoform

import (
	"io"
	"mime"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mazrean/formstream"
)

type Parser struct {
	*formstream.Parser
	reader io.Reader
}

func NewParser(c echo.Context, options ...formstream.ParserOption) (*Parser, error) {
	contentType := c.Request().Header.Get("Content-Type")
	d, params, err := mime.ParseMediaType(contentType)
	if err != nil || d != "multipart/form-data" {
		return nil, http.ErrNotMultipart
	}

	boundary, ok := params["boundary"]
	if !ok {
		return nil, http.ErrMissingBoundary
	}

	return &Parser{
		Parser: formstream.NewParser(boundary, options...),
		reader: c.Request().Body,
	}, nil
}

func (p *Parser) Parse() error {
	return p.Parser.Parse(p.reader)
}
