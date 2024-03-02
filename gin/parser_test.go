package ginform_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mazrean/formstream"
	ginform "github.com/mazrean/formstream/gin"
)

func TestExample(t *testing.T) {
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
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")

	rec := httptest.NewRecorder()

	router := gin.Default()
	router.POST("/user", createUserHandler)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status code is wrong: expected: %d, actual: %d\n", http.StatusCreated, rec.Code)
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

func createUserHandler(c *gin.Context) {
	parser, err := ginform.NewParser(c)
	if err != nil {
		log.Println(err)
		c.Status(http.StatusBadRequest)
		return
	}

	err = parser.Register("icon", func(r io.Reader, header formstream.Header) error {
		name, _, _ := parser.Value("name")
		password, _, _ := parser.Value("password")

		return saveUser(c.Request.Context(), name, password, r)
	}, formstream.WithRequiredPart("name"), formstream.WithRequiredPart("password"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to register",
		})
		log.Println(err)
		return
	}

	err = parser.Parse()
	if err != nil {
		log.Println(err)
		c.Status(http.StatusBadRequest)
		return
	}

	c.Status(http.StatusCreated)
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
