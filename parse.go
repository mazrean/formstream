package formstream

//go:generate go run github.com/golang/mock/mockgen -source=$GOFILE -destination=mock/${GOFILE} -package=mock

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
)

func (p *Parser) Parse(r io.Reader) (err error) {
	wh := newHookSatisfactionChecker(p.hookMap, p.maxMemFileSize)
	defer func() {
		deferErr := wh.Close()
		// capture the error of Close()
		if deferErr != nil {
			if err != nil {
				err = errors.Join(err, deferErr)
			} else {
				err = deferErr
			}
		}
	}()

	err = p.parse(r, wh)

	return err
}

func (p *Parser) parse(r io.Reader, wh iHookSatisfactionChecker) error {
	mr := multipart.NewReader(r, p.boundary)
	for {
		var part *multipart.Part
		part, err := mr.NextPart()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("failed to read next part: %w", err)
		}

		if _, ok := p.hookMap[part.FormName()]; ok {
			err := wh.runOrSetHook(part.FormName(), part)
			if err != nil {
				return fmt.Errorf("failed to run or set hook: %w", err)
			}
		} else {
			b := bytes.NewBuffer(nil)
			if _, err := io.Copy(b, part); err != nil {
				return fmt.Errorf("failed to copy part: %w", err)
			}

			p.valueMap[part.FormName()] = append(p.valueMap[part.FormName()], Value{
				content: b.Bytes(),
				header:  newHeader(part.Header),
			})
		}

		err = wh.runSatisfiedHook(part.FormName())
		if err != nil {
			return fmt.Errorf("failed to run satisfied hook: %w", err)
		}
	}

	return nil
}

type iHookSatisfactionChecker interface {
	runOrSetHook(name string, part *multipart.Part) error
	runSatisfiedHook(name string) error
}

type hookSatisfactionChecker struct {
	satisfiedHooks   map[string]StreamHookFunc
	unsatisfiedHooks map[string][]*waitHook
	hookMap          map[string]*waitHook

	maxMemFileSize DataSize
	offset         int64
	file           *os.File
}

func newHookSatisfactionChecker(streamHooks map[string]streamHook, maxMemFileSize DataSize) *hookSatisfactionChecker {
	satisfiedHooks := make(map[string]StreamHookFunc, len(streamHooks))
	unsatisfiedHooks := make(map[string][]*waitHook)
	hookMap := make(map[string]*waitHook, len(streamHooks))
	for name, hook := range streamHooks {
		if len(hook.requireParts) == 0 {
			satisfiedHooks[name] = hook.fn
			continue
		}

		waitHookValue := waitHook{
			name:             name,
			fn:               hook.fn,
			unsatisfiedCount: len(hook.requireParts),
		}
		hookMap[name] = &waitHookValue
		for _, requirePart := range hook.requireParts {
			unsatisfiedHooks[requirePart] = append(unsatisfiedHooks[requirePart], &waitHookValue)
		}
	}

	return &hookSatisfactionChecker{
		satisfiedHooks:   satisfiedHooks,
		unsatisfiedHooks: unsatisfiedHooks,
		hookMap:          hookMap,
		maxMemFileSize:   maxMemFileSize,
	}
}

func (w *hookSatisfactionChecker) runOrSetHook(name string, part *multipart.Part) error {
	header := newHeader(part.Header)

	if fn := w.satisfiedHooks[name]; fn != nil {
		err := fn(part, header)
		if err != nil {
			return fmt.Errorf("failed to execute hook: %w", err)
		}
		return nil
	}

	buf := bytes.NewBuffer(nil)
	n, err := io.CopyN(buf, part, int64(w.maxMemFileSize)+1)
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("failed to copy: %w", err)
	}

	var cntnt content
	if n > int64(w.maxMemFileSize) {
		if w.file == nil {
			f, err := os.CreateTemp("", "formstream-")
			if err != nil {
				return fmt.Errorf("failed to create temp file: %w", err)
			}
			w.file = f
		}

		_, err := io.Copy(w.file, buf)
		if err != nil {
			return fmt.Errorf("failed to write: %w", err)
		}

		remainSize, err := io.Copy(w.file, part)
		if err != nil {
			return fmt.Errorf("failed to copy: %w", err)
		}

		cntnt = fileContent{
			file:   w.file,
			offset: w.offset,
			size:   int64(buf.Len()) + remainSize,
		}
	} else {
		cntnt = bytesContent{
			content: buf.Bytes(),
		}
		w.maxMemFileSize -= DataSize(buf.Len())
	}

	if hook, ok := w.hookMap[name]; ok {
		hook.callParams = append(hook.callParams, callParams{
			content: cntnt,
			header:  header,
		})
	} else {
		// actually not reached
		return fmt.Errorf("no such hook %s", name)
	}

	return nil
}

func (w *hookSatisfactionChecker) runSatisfiedHook(name string) error {
	var errs []error
	for _, hook := range w.unsatisfiedHooks[name] {
		if hook.unsatisfiedCount <= 0 {
			continue
		}
		hook.unsatisfiedCount--
		if hook.unsatisfiedCount > 0 {
			continue
		}

		w.satisfiedHooks[hook.name] = hook.fn

		for _, param := range hook.callParams {
			err := func() (err error) {
				err = hook.fn(param.content.Reader(), param.header)
				if err != nil {
					return fmt.Errorf("failed to execute hook of %s: %v", hook.name, err)
				}

				w.maxMemFileSize += param.content.FreeMemSize()

				return nil
			}()
			if err != nil {
				errs = append(errs, err)
			}
		}

		hook.callParams = nil
	}
	if len(errs) != 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (w *hookSatisfactionChecker) Close() error {
	if w.file != nil {
		err := w.file.Close()
		if err != nil {
			return fmt.Errorf("failed to close file: %w", err)
		}
	}

	return nil
}

type waitHook struct {
	name             string
	fn               StreamHookFunc
	callParams       []callParams
	unsatisfiedCount int
}

type callParams struct {
	content content
	header  Header
}

type content interface {
	Reader() io.Reader
	FreeMemSize() DataSize
}

type bytesContent struct {
	content []byte
}

func (c bytesContent) Reader() io.Reader {
	return bytes.NewReader(c.content)
}

func (c bytesContent) FreeMemSize() DataSize {
	return DataSize(len(c.content))
}

type fileContent struct {
	file   io.ReaderAt
	offset int64
	size   int64
}

func (c fileContent) Reader() io.Reader {
	return io.NewSectionReader(c.file, c.offset, c.size)
}

func (c fileContent) FreeMemSize() DataSize {
	return 0
}
