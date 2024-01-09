package formstream

import (
	"fmt"
)

func (p *Parser) Register(name string, fn StreamHookFunc, options ...RegisterOption) error {
	if _, ok := p.hookMap[name]; ok {
		return ErrDuplicateHookName{Name: name}
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

type ErrDuplicateHookName struct {
	Name string
}

func (e ErrDuplicateHookName) Error() string {
	return fmt.Sprintf("duplicate hook name: %s", e.Name)
}

type registerConfig struct {
	requireParts []string
}

type RegisterOption func(*registerConfig)

func WithFieldRequirement(name string) RegisterOption {
	return func(c *registerConfig) {
		c.requireParts = append(c.requireParts, name)
	}
}
