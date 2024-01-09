package formstream

//go:generate go run github.com/golang/mock/mockgen -source=$GOFILE -destination=mock/${GOFILE} -package=mock

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"os"
)

func (p *Parser) Parse(r io.Reader) (err error) {
	wh := newHookSatisfactionChecker(p.hookMap)
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
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read next part: %w", err)
		}

		fileHeader := FileHeader{
			FileName: part.FileName(),
			Header:   part.Header,
		}
		p.valueHeaderMap[part.FormName()] = append(p.valueHeaderMap[part.FormName()], &fileHeader)

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

			p.Value[part.FormName()] = append(p.Value[part.FormName()], &Value{
				Body:       b.Bytes(),
				FileHeader: &fileHeader,
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
}

func newHookSatisfactionChecker(streamHooks map[string]streamHook) *hookSatisfactionChecker {
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
		unsatisfiedHooks: make(map[string][]*waitHook),
	}
}

func (w *hookSatisfactionChecker) runOrSetHook(name string, part *multipart.Part) error {
	if fn := w.satisfiedHooks[name]; fn != nil {
		err := fn(part, part.FileName(), part.Header)
		if err != nil {
			return fmt.Errorf("failed to execute hook: %w", err)
		}
		return nil
	}

	f, err := os.CreateTemp("", "formstream-")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	if hook, ok := w.hookMap[name]; ok {
		hook.callParams = &callParams{
			reader:   f,
			fileName: part.FileName(),
			header:   part.Header,
		}
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

		if hook.callParams == nil {
			w.satisfiedHooks[hook.name] = hook.fn
			continue
		}

		err := func() (err error) {
			defer func() {
				err := hook.callParams.reader.Close()
				if err != nil {
					errs = append(errs, err)
				}
			}()

			err = hook.fn(hook.callParams.reader, hook.callParams.fileName, hook.callParams.header)
			if err != nil {
				return fmt.Errorf("failed to execute hook of %s: %v", hook.name, err)
			}

			return nil
		}()
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) != 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (w *hookSatisfactionChecker) Close() error {
	var errs []error
	for _, hook := range w.hookMap {
		if hook.unsatisfiedCount > 0 && hook.callParams != nil {
			err := hook.callParams.reader.Close()
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) != 0 {
		return errors.Join(errs...)
	}

	return nil
}

type waitHook struct {
	name             string
	fn               StreamHookFunc
	callParams       *callParams
	unsatisfiedCount int
}

type callParams struct {
	reader   io.ReadCloser
	fileName string
	header   textproto.MIMEHeader
}
