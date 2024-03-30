package formstream_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/textproto"
	"os"
	"strings"
	"testing"

	"github.com/mazrean/formstream"
	"github.com/mazrean/formstream/internal/myio"
)

func ExampleNewParser() {
	buf := strings.NewReader(`
--boundary
Content-Disposition: form-data; name="field"

value
--boundary
Content-Disposition: form-data; name="stream"; filename="file.txt"
Content-Type: text/plain

large file contents
--boundary--`)

	parser := formstream.NewParser("boundary")

	err := parser.Register("stream", func(r io.Reader, header formstream.Header) error {
		fmt.Println("---stream---")
		fmt.Printf("file name: %s\n", header.FileName())
		fmt.Printf("Content-Type: %s\n", header.ContentType())
		fmt.Println()

		_, err := io.Copy(os.Stdout, r)
		if err != nil {
			return fmt.Errorf("failed to copy: %w", err)
		}

		return nil
	}, formstream.WithRequiredPart("field"))
	if err != nil {
		log.Fatal(err)
	}

	err = parser.Parse(buf)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n\n")
	fmt.Println("---field---")
	content, _, _ := parser.Value("field")
	fmt.Println(content)

	// Output:
	// ---stream---
	// file name: file.txt
	// Content-Type: text/plain
	//
	// large file contents
	//
	// ---field---
	// value
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

	mbData := make([]byte, formstream.MB)
	for i := 0; i < int(fileSize/formstream.MB); i++ {
		_, err := pw.Write(mbData)
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
	r, err := sampleForm(fileSize, boundary, reverse)
	if err != nil {
		b.Fatal(err)
	}
	defer r.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		_, err := r.Seek(0, io.SeekStart)
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()

		parser := formstream.NewParser(boundary)

		err = parser.Register("stream", func(r io.Reader, _ formstream.Header) error {
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

		err = parser.Parse(r)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStdMultipartNextPart(b *testing.B) {
	b.Run("1MB", func(b *testing.B) {
		benchmarkStdMultipartNextPart(b, 1*formstream.MB)
	})
	b.Run("10MB", func(b *testing.B) {
		benchmarkStdMultipartNextPart(b, 10*formstream.MB)
	})
	b.Run("100MB", func(b *testing.B) {
		benchmarkStdMultipartNextPart(b, 100*formstream.MB)
	})
	b.Run("1GB", func(b *testing.B) {
		benchmarkStdMultipartNextPart(b, 1*formstream.GB)
	})
	b.Run("5GB", func(b *testing.B) {
		if testing.Short() {
			b.Skip("skipping test in short mode.")
		}
		benchmarkStdMultipartNextPart(b, 5*formstream.GB)
	})
	b.Run("10GB", func(b *testing.B) {
		if testing.Short() {
			b.Skip("skipping test in short mode.")
		}
		benchmarkStdMultipartNextPart(b, 10*formstream.GB)
	})
}

func benchmarkStdMultipartNextPart(b *testing.B, fileSize formstream.DataSize) {
	r, err := sampleForm(fileSize, boundary, false)
	if err != nil {
		b.Fatal(err)
	}
	defer r.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		_, err := r.Seek(0, io.SeekStart)
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()

		func() {
			mr := multipart.NewReader(r, boundary)

			for {
				p, err := mr.NextPart()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					b.Fatal(err)
				}

				if p.FormName() == "field" {
					sb := &strings.Builder{}
					_, err := io.Copy(sb, p)
					if err != nil {
						b.Fatal(err)
					}

					_ = sb.String()
				} else {
					_, err := io.Copy(io.Discard, p)
					if err != nil {
						b.Fatal(err)
					}
				}

				_, err = io.Copy(io.Discard, p)
				if err != nil {
					b.Fatal(err)
				}
			}
		}()
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

	r, err := sampleForm(fileSize, boundary, false)
	if err != nil {
		b.Fatal(err)
	}
	defer r.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		_, err := r.Seek(0, io.SeekStart)
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()

		func() {
			mr := multipart.NewReader(r, boundary)
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
		}()
	}
}
