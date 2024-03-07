package echoform_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/mazrean/formstream"
	echoform "github.com/mazrean/formstream/echo"
)

func TestExample(t *testing.T) {
	e := echo.New()

	req := httptest.NewRequest(http.MethodPost, "/user", strings.NewReader(`
--boundary
Content-Disposition: form-data; name="name"

mazrean
--boundary
Content-Disposition: form-data; name="password"

password
--boundary
Content-Disposition: form-data; name="icon"; filename="icon.png"
Content-Type: image/png

icon contents
--boundary--`))
	req.Header.Set(echo.HeaderContentType, "multipart/form-data; boundary=boundary")

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := createUserHandler(c)
	if err != nil {
		t.Fatalf("failed to create user: %s\n", err)
		return
	}

	if user.name != "mazrean" {
		t.Errorf("user name is wrong: expected: mazrean, actual: %s\n", user.name)
	}
	if user.password != "password" {
		t.Errorf("user password is wrong: expected: password, actual: %s\n", user.password)
	}
	if user.icon != "icon contents" {
		t.Errorf("user icon is wrong: expected: icon contents, actual: %s\n", user.icon)
	}
}

func createUserHandler(c echo.Context) error {
	parser, err := echoform.NewParser(c)
	if err != nil {
		return c.NoContent(http.StatusBadRequest)
	}

	err = parser.Register("icon", func(r io.Reader, _ formstream.Header) error {
		name, _, _ := parser.Value("name")
		password, _, _ := parser.Value("password")

		return saveUser(c.Request().Context(), name, password, r)
	}, formstream.WithRequiredPart("name"), formstream.WithRequiredPart("password"))
	if err != nil {
		return err
	}

	err = parser.Parse()
	if err != nil {
		return c.NoContent(http.StatusBadRequest)
	}

	return c.NoContent(http.StatusCreated)
}

var (
	user = struct {
		name     string
		password string
		icon     string
	}{}
)

func saveUser(_ context.Context, name string, password string, iconReader io.Reader) error {
	user.name = name
	user.password = password

	sb := strings.Builder{}
	_, err := io.Copy(&sb, iconReader)
	if err != nil {
		return fmt.Errorf("failed to copy: %w", err)
	}
	user.icon = sb.String()

	return nil
}
