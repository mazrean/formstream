package httpform_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"strings"
	"testing"

	"github.com/mazrean/formstream"
	httpform "github.com/mazrean/formstream/http"
	"github.com/mazrean/formstream/internal/myio"
	"golang.org/x/sync/errgroup"
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

	createUserHandler(rec, req)

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

func createUserHandler(res http.ResponseWriter, req *http.Request) {
	parser, err := httpform.NewParser(req)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	err = parser.Register("icon", func(r io.Reader, header formstream.Header) error {
		name, _, _ := parser.Value("name")
		password, _, _ := parser.Value("password")

		return saveUser(req.Context(), name, password, r)
	}, formstream.WithRequiredPart("name"), formstream.WithRequiredPart("password"))
	if err != nil {
		log.Printf("failed to register: %s\n", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = parser.Parse()
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	res.WriteHeader(http.StatusCreated)
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

const boundary = "boundary"

func sampleForm(fileSize formstream.DataSize, boundary string, reverse bool) (io.ReadSeekCloser, error) {
	if fileSize > 1*formstream.GB {
		f, err := os.CreateTemp("", "formstream-test-form-")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp file: %w", err)
		}

		err = createSampleForm(f, fileSize, boundary, reverse)
		if err != nil {
			return nil, fmt.Errorf("failed to create sample form: %w", err)
		}

		return f, nil
	}

	buf := bytes.NewBuffer(nil)

	err := createSampleForm(buf, fileSize, boundary, reverse)
	if err != nil {
		return nil, fmt.Errorf("failed to create sample form: %w", err)
	}

	return myio.NopSeekCloser(bytes.NewReader(buf.Bytes())), nil
}

func createSampleForm(w io.Writer, fileSize formstream.DataSize, boundary string, reverse bool) error {
	mw := multipart.NewWriter(w)
	defer mw.Close()

	err := mw.SetBoundary(boundary)
	if err != nil {
		return fmt.Errorf("failed to set boundary: %w", err)
	}

	if !reverse {
		err := mw.WriteField("field", "value")
		if err != nil {
			return fmt.Errorf("failed to write field: %w", err)
		}
	}

	mh := make(textproto.MIMEHeader)
	mh.Set("Content-Disposition", `form-data; name="stream"; filename="file.txt"`)
	mh.Set("Content-Type", "text/plain")
	pw, err := mw.CreatePart(mh)
	if err != nil {
		return fmt.Errorf("failed to create part: %w", err)
	}
	for i := 0; i < int(fileSize/formstream.MB); i++ {
		_, err := pw.Write([]byte(strings.Repeat("a", int(formstream.MB))))
		if err != nil {
			return fmt.Errorf("failed to write: %w", err)
		}
	}

	if reverse {
		err := mw.WriteField("field", "value")
		if err != nil {
			return fmt.Errorf("failed to write field: %w", err)
		}
	}

	return nil
}

func TestSlowWriter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parser, err := httpform.NewParser(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = parser.Register("stream", func(r io.Reader, header formstream.Header) error {
			// get field value
			_, _, _ = parser.Value("field")

			_, err := io.Copy(myio.SlowWriter(), r)
			if err != nil {
				return fmt.Errorf("failed to copy: %w", err)
			}

			return nil
		}, formstream.WithRequiredPart("field"))
		if err != nil {
			t.Fatal(err)
		}

		err = parser.Parse()
		if err != nil {
			t.Fatal(err)
		}

		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	f, err := os.CreateTemp("", "formstream-test-form-")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	err = createSampleForm(f, 1*formstream.GB, boundary, false)
	if err != nil {
		t.Fatal(err)
	}

	eg := &errgroup.Group{}
	for i := 0; i < 100; i++ {
		f, err := os.Open(f.Name())
		if err != nil {
			t.Fatal(err)
		}

		req, err := http.NewRequest(http.MethodPost, srv.URL, f)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", fmt.Sprintf("multipart/form-data; boundary=%s", boundary))

		eg.Go(func() error {
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("failed to request: %w", err)
			}

			if res.StatusCode != http.StatusCreated {
				return fmt.Errorf("status code is wrong: expected: %d, actual: %d", http.StatusCreated, res.StatusCode)
			}

			return nil
		})
	}

	err = eg.Wait()
	if err != nil {
		t.Error(err)
	}
}

func BenchmarkFormStreamFastPath(b *testing.B) {
	b.Run("1MB", func(b *testing.B) {
		benchmarkFormStream(b, 1*formstream.MB, false)
	})
	b.Run("10MB", func(b *testing.B) {
		benchmarkFormStream(b, 10*formstream.MB, false)
	})
	b.Run("100MB", func(b *testing.B) {
		benchmarkFormStream(b, 100*formstream.MB, false)
	})
	b.Run("1GB", func(b *testing.B) {
		benchmarkFormStream(b, 1*formstream.GB, false)
	})
	b.Run("5GB", func(b *testing.B) {
		if testing.Short() {
			b.Skip("skipping test in short mode.")
		}
		benchmarkFormStream(b, 5*formstream.GB, false)
	})
	b.Run("10GB", func(b *testing.B) {
		if testing.Short() {
			b.Skip("skipping test in short mode.")
		}
		benchmarkFormStream(b, 10*formstream.GB, false)
	})
}

func BenchmarkFormStreamSlowPath(b *testing.B) {
	b.Run("1MB", func(b *testing.B) {
		benchmarkFormStream(b, 1*formstream.MB, true)
	})
	b.Run("10MB", func(b *testing.B) {
		benchmarkFormStream(b, 10*formstream.MB, true)
	})
	b.Run("100MB", func(b *testing.B) {
		benchmarkFormStream(b, 100*formstream.MB, true)
	})
	b.Run("1GB", func(b *testing.B) {
		benchmarkFormStream(b, 1*formstream.GB, true)
	})
	b.Run("5GB", func(b *testing.B) {
		if testing.Short() {
			b.Skip("skipping test in short mode.")
		}
		benchmarkFormStream(b, 5*formstream.GB, true)
	})
	b.Run("10GB", func(b *testing.B) {
		if testing.Short() {
			b.Skip("skipping test in short mode.")
		}
		benchmarkFormStream(b, 10*formstream.GB, true)
	})
}

func benchmarkFormStream(b *testing.B, fileSize formstream.DataSize, reverse bool) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parser, err := httpform.NewParser(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = parser.Register("stream", func(r io.Reader, header formstream.Header) error {
			// get field value
			_, _, _ = parser.Value("field")

			_, err := io.Copy(io.Discard, r)
			if err != nil {
				return fmt.Errorf("failed to copy: %w", err)
			}

			return nil
		}, formstream.WithRequiredPart("field"))
		if err != nil {
			b.Fatal(err)
		}

		err = parser.Parse()
		if err != nil {
			b.Fatal(err)
		}

		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	r, err := sampleForm(fileSize, boundary, reverse)
	if err != nil {
		b.Fatal(err)
	}
	defer r.Close()

	req, err := http.NewRequest(http.MethodPost, srv.URL, r)
	if err != nil {
		b.Fatal(err)
	}
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/form-data; boundary=%s", boundary))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		_, err := r.Seek(0, io.SeekStart)
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()

		_, err = http.DefaultClient.Do(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStdMultipartReadForm(b *testing.B) {
	b.Run("1MB", func(b *testing.B) {
		benchmarkStdMultipartReadForm(b, 1*formstream.MB)
	})
	b.Run("10MB", func(b *testing.B) {
		benchmarkStdMultipartReadForm(b, 10*formstream.MB)
	})
	b.Run("100MB", func(b *testing.B) {
		benchmarkStdMultipartReadForm(b, 100*formstream.MB)
	})
	b.Run("1GB", func(b *testing.B) {
		benchmarkStdMultipartReadForm(b, 1*formstream.GB)
	})
	b.Run("5GB", func(b *testing.B) {
		if testing.Short() {
			b.Skip("skipping test in short mode.")
		}
		benchmarkStdMultipartReadForm(b, 5*formstream.GB)
	})
	b.Run("10GB", func(b *testing.B) {
		if testing.Short() {
			b.Skip("skipping test in short mode.")
		}
		benchmarkStdMultipartReadForm(b, 10*formstream.GB)
	})
}

func benchmarkStdMultipartReadForm(b *testing.B, fileSize formstream.DataSize) {
	// default value in http package
	const maxMemory = 32 * formstream.MB

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mr := multipart.NewReader(r.Body, boundary)
		form, err := mr.ReadForm(int64(maxMemory))
		if err != nil {
			b.Fatal(err)
		}
		defer func() {
			err := form.RemoveAll()
			if err != nil {
				b.Fatal(err)
			}
		}()

		f, err := form.File["stream"][0].Open()
		if err != nil {
			b.Fatal(err)
		}
		defer f.Close()

		_, err = io.Copy(io.Discard, f)
		if err != nil {
			b.Fatal(err)
		}

		// get field value
		_ = form.Value["field"][0]

		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	r, err := sampleForm(fileSize, boundary, false)
	if err != nil {
		b.Fatal(err)
	}
	defer r.Close()

	req, err := http.NewRequest(http.MethodPost, srv.URL, r)
	if err != nil {
		b.Fatal(err)
	}
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/form-data; boundary=%s", boundary))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		_, err := r.Seek(0, io.SeekStart)
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()

		_, err = http.DefaultClient.Do(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}
