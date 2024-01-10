package formstream_test

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/textproto"
	"os"
	"strings"
	"testing"

	"github.com/mazrean/formstream"
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

	parser.Register("stream", func(r io.Reader, header formstream.Header) error {
		fmt.Println("---stream---")
		fmt.Printf("file name: %s\n", header.FileName())
		fmt.Printf("Content-Type: %s\n", header.ContentType())
		fmt.Println()

		_, err := io.Copy(os.Stdout, r)
		if err != nil {
			return fmt.Errorf("failed to copy: %w", err)
		}

		return nil
	})

	err := parser.Parse(buf)
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

func sampleForm(fileSize formstream.DataSize, boundary string, reverse bool) (io.Reader, error) {
	b := bytes.NewBuffer(nil)

	mw := multipart.NewWriter(b)
	defer mw.Close()

	mw.SetBoundary(boundary)

	if !reverse {
		mw.WriteField("field", "value")
	}

	mh := make(textproto.MIMEHeader)
	mh.Set("Content-Disposition", `form-data; name="stream"; filename="file.txt"`)
	mh.Set("Content-Type", "text/plain")
	w, err := mw.CreatePart(mh)
	if err != nil {
		return nil, fmt.Errorf("failed to create part: %w", err)
	}
	_, err = io.CopyN(w, strings.NewReader(strings.Repeat("a", int(fileSize))), int64(fileSize))
	if err != nil {
		return nil, fmt.Errorf("failed to copy: %w", err)
	}

	if reverse {
		mw.WriteField("field", "value")
	}

	return b, nil
}

func BenchmarkFormstream(b *testing.B) {
	b.Run("1MB", func(b *testing.B) {
		benchmarkFormstream(b, 1*formstream.MB, false)
	})
	b.Run("10MB", func(b *testing.B) {
		benchmarkFormstream(b, 10*formstream.MB, false)
	})
	b.Run("100MB", func(b *testing.B) {
		benchmarkFormstream(b, 100*formstream.MB, false)
	})
	b.Run("1GB", func(b *testing.B) {
		benchmarkFormstream(b, 1*formstream.GB, false)
	})

	b.Run("1MB Reverse", func(b *testing.B) {
		benchmarkFormstream(b, 1*formstream.MB, true)
	})
	b.Run("10MB Reverse", func(b *testing.B) {
		benchmarkFormstream(b, 10*formstream.MB, true)
	})
	b.Run("100MB Reverse", func(b *testing.B) {
		benchmarkFormstream(b, 100*formstream.MB, true)
	})
	b.Run("1GB Reverse", func(b *testing.B) {
		benchmarkFormstream(b, 1*formstream.GB, true)
	})
}

func benchmarkFormstream(b *testing.B, fileSize formstream.DataSize, reverse bool) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		r, err := sampleForm(fileSize, boundary, reverse)
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()

		parser := formstream.NewParser(boundary)

		parser.Register("stream", func(r io.Reader, header formstream.Header) error {
			// get field value
			_, _, _ = parser.Value("field")

			_, err := io.Copy(io.Discard, r)
			if err != nil {
				return fmt.Errorf("failed to copy: %w", err)
			}

			return nil
		}, formstream.WithRequiredPart("field"))

		err = parser.Parse(r)
		if err != nil {
			b.Fatal(err)
		}

	}
}

func BenchmarkStdMultipart_ReadForm(b *testing.B) {
	// default value in http package
	const maxMemory = 32 * formstream.MB

	b.Run("1MB", func(b *testing.B) {
		benchmarkStdMultipart_ReadForm(b, 1*formstream.MB, maxMemory)
	})
	b.Run("10MB", func(b *testing.B) {
		benchmarkStdMultipart_ReadForm(b, 10*formstream.MB, maxMemory)
	})
	b.Run("100MB", func(b *testing.B) {
		benchmarkStdMultipart_ReadForm(b, 100*formstream.MB, maxMemory)
	})
	b.Run("1GB", func(b *testing.B) {
		benchmarkStdMultipart_ReadForm(b, 1*formstream.GB, maxMemory)
	})
}

func benchmarkStdMultipart_ReadForm(b *testing.B, fileSize formstream.DataSize, maxMemory formstream.DataSize) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		r, err := sampleForm(fileSize, boundary, false)
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
			defer form.RemoveAll()

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
