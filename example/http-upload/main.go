package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mazrean/formstream"
	httpform "github.com/mazrean/formstream/http"
)

const iconDir = "icons"

func main() {
	err := os.MkdirAll(iconDir, 0755)
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		parser, err := httpform.NewParser(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = parser.Register("icon", func(r io.Reader, header formstream.Header) error {
			if header.ContentType() != "image/png" {
				http.Error(w, "content type is not supported", http.StatusBadRequest)
				return fmt.Errorf("content type is not supported")
			}

			id, _, _ := parser.Value("id")
			iconPath := filepath.Join(iconDir, id)

			_, err := os.Stat(iconPath)
			if err != nil && !os.IsNotExist(err) {
				http.Error(w, "failed to check file existence", http.StatusInternalServerError)
				return fmt.Errorf("failed to check file existence: %w", err)
			}

			if err == nil {
				http.Error(w, "user already exists", http.StatusConflict)
				return fmt.Errorf("user already exists")
			}

			file, err := os.Create(iconPath)
			if err != nil {
				http.Error(w, "failed to create file", http.StatusInternalServerError)
				return fmt.Errorf("failed to create file: %w", err)
			}
			defer file.Close()

			_, err = io.Copy(file, r)
			if err != nil {
				http.Error(w, "failed to copy", http.StatusInternalServerError)
				return fmt.Errorf("failed to copy: %w", err)
			}

			return nil
		}, formstream.WithRequiredPart("id"))
		if err != nil {
			http.Error(w, "failed to register hook", http.StatusInternalServerError)
			return
		}

		err = parser.Parse()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
	})
	mux.Handle("/icons/", http.StripPrefix("/icons/", http.FileServer(http.Dir(iconDir))))

	err = http.ListenAndServe(":8080", mux)
	if err != nil {
		panic(err)
	}
}
