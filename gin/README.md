# Wrapper for [Gin](https://gin-gonic.com/)

## Usage
```go
func createUserHandler(ctx *gin.Context) {
	parser, err := ginform.NewParser(ctx)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	err = parser.Register("icon", func(r io.Reader, header formstream.Header) error {
		name, _, _ := parser.Value("name")
		password, _, _ := parser.Value("password")

		return saveUser(ctx.Request.Context(), name, password, r)
	}, formstream.WithRequiredPart("name"), formstream.WithRequiredPart("password"))
	if err != nil {
		ctx.Error(err)
		return
	}

	err = parser.Parse()
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	ctx.Status(http.StatusCreated)
}
```
