# Repository Guidelines — formstream

Streaming `multipart/form-data` parser for Go, with adapters for `echo`, `gin`,
and `net/http`.

> Agent configuration is managed via [apm](https://github.com/microsoft/apm).
> Common conventions live in `mazrean/apm-plackage/common`; Go-specific rules
> come from `mazrean/apm-plackage/go`. Run `apm install` to materialise locally.

## Project Structure

- Multi-module workspace: root + `echo/` + `gin/` + `http/` adapters.
- Each adapter is its own Go module to keep the dependency footprint small.

## Build & Test

- `go test -v ./...` — root module
- `cd echo && go test -v ./...` — echo adapter
- `cd gin && go test -v ./...` — gin adapter
- `cd http && go test -v ./...` — http adapter
- `golangci-lint run` — lint all modules

## Conventions

- Specs go under `specs/`; use `mazrean/agent-skills/skills/writing-*`.
- Commit using Conventional Commits (`committing-code` skill).
- Use the Go 1.24+ `tool` directive for build tools; see `using-go-tool-directive` skill.
