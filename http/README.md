# Wrapper for [net/http](https://pkg.go.dev/net/http)

## Usage

<details>
<summary>Example data</summary>

```text
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
--boundary--
```
</details>

```go
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
```
