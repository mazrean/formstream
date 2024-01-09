package formstream_test

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

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
	fmt.Println(parser.Value("field"))

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
