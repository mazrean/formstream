package formstream

import (
	"fmt"
)

// Register registers a stream hook with the given name.
func (p *Parser) Register(name string, fn StreamHookFunc, options ...RegisterOption) error {
	if _, ok := p.hookMap[name]; ok {
		return DuplicateHookNameError{Name: name}
	}

	c := &registerConfig{}
	for _, opt := range options {
		opt(c)
	}

	p.hookMap[name] = streamHook{
		fn:           fn,
		requireParts: c.requireParts,
	}

	return nil
}

type DuplicateHookNameError struct {
	Name string
}

func (e DuplicateHookNameError) Error() string {
	return fmt.Sprintf("duplicate hook name: %s", e.Name)
}

type registerConfig struct {
	requireParts []string
}

type RegisterOption func(*registerConfig)

// WithRequiredPart sets the required part names for the stream hook.
func WithRequiredPart(name string) RegisterOption {
	return func(c *registerConfig) {
		c.requireParts = append(c.requireParts, name)
	}
}
