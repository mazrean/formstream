# Wrapper for [echo](https://echo.labstack.com/)

## Usage
```go
func createUserHandler(c echo.Context) error {
	parser, err := echoform.NewParser(c)
	if err != nil {
		return c.NoContent(http.StatusBadRequest)
	}

	err = parser.Register("icon", func(r io.Reader, header formstream.Header) error {
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
```
