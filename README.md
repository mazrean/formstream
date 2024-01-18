# FormStream

[![GitHub release](https://img.shields.io/github/release/mazrean/formstream.svg)](https://github.com/mazrean/formstream/releases/)
![CI main](https://github.com/mazrean/formstream/actions/workflows/ci.yaml/badge.svg)
[![codecov](https://codecov.io/gh/mazrean/formstream/branch/master/graph/badge.svg)](https://codecov.io/gh/mazrean/formstream)
[![Go Reference](https://pkg.go.dev/badge/github.com/mazrean/formstream.svg)](https://pkg.go.dev/github.com/mazrean/formstream)

FormStream is a Golang streaming parser for multipart.

## Features

- Streaming parser, no need to store the whole file in memory or on disk in most cases
- Extremely low memory usage
- Fast, fast, fast!

## Benchmarks

For all file sizes, FormStream is faster and uses less memory than [`mime/multipartform`](https://pkg.go.dev/mime/multipart).

<details>
<summary>Environment</summary>

- OS:
- CPU:
- RAM:
- Disk:
- Go version:
</details>

![](./docs/images/memory.png)
![](./docs/images/time.png)

> [!NOTE]
> FormStream is extremely fast by using a stream when parsing a multipart that satisfies certain conditions (FastPath in the graph).
> It is fast enough even when no conditions are met (SlowPath in the graph), but only slightly faster than `mime/multipart`.
> If you want to know more details, see [How it works](./#how-it-works).

## Install

```sh
go get github.com/mazrean/formstream@latest
```

## Usage

### Basic

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
parser, err := formstream.NewParser(r)
if err != nil {
    return err
}

err = parser.Register("icon", func(r io.Reader, header formstream.Header) error {
    name, _, _ := parser.Value("name")
    password, _, _ := parser.Value("password")

    return saveUser(r.Context(), name, password, r)
}, formstream.WithRequiredPart("name"), formstream.WithRequiredPart("password"))
if err != nil {
    return err
}

err = parser.Parse()
if err != nil {
    return err
}
```

### `net/http`, `Echo`, `Gin`

Use wrapper for each library.

- [net/http](./http)
- [Echo](./echo)
- [Gin](./gin)

## How it works

multipart is data in the following format.
```text
--boundary
Content-Disposition: form-data; name="description"

file description
--boundary
Content-Disposition: form-data; name="file"; filename="large.png"
Content-Type: image/png

large png data...
--boundary--
```
In this case, the file part is large and should be streamed as much as possible.
In the case of the above data, stream processing can be performed by processing the data in order from the beginning.
Such processing can be achieved by the `(*Reader).NextPart` method of `mime/multipart`.

In `mime/multipart`, the `(*Reader).ReadForm` method can also parse a multipart.
This method does not do stream processing, but saves the data once 
in memory or a file and is slow.
However, `net/http`, `Echo`, and `Gin` use the `(*Reader).ReadForm` method to parse multipart.
This is because even if you need the data in description to process a file, you need to process the following data.
```text
--boundary
Content-Disposition: form-data; name="file"; filename="large.png"
Content-Type: image/png

large png data...
--boundary
Content-Disposition: form-data; name="description"

file description
--boundary--
```
NextPart` can only process in order from the beginning, so in such cases, the data must be saved once to disk or memory.

In summary, the multipart format can be processed efficiently and safely as follows
- When the data necessary to process a Part is available, perform Stream processing as in the `(*Reader).NextPart` method.
- If data necessary to process a Part is not available, write it to disk or memory and process it when it is available, as in the `(*Reader).ReadForm` method.

FormStream achieves this behavior faster than the `(*Reader).ReadForm` method, and unlike the `(*Reader).NextPart` method, it can process arbitrary multipart data.
