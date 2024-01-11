package conditionjudge

import (
	"errors"
	"fmt"
)

type IConditionJudger[K comparable, S any, T any] interface {
	HookEvent(key K, value S) (bool, error)
	KeyEvent(key K) error
}

type ConditionJudger[K comparable, S any, T any] struct {
	preProcessFunc   PreProcessFunc[S, T]
	satisfiedHooks   map[K]func(S) error
	unsatisfiedHooks map[K][]*waitHook[K, S, T]
	hooks            map[K]*waitHook[K, S, T]
}

type PreProcessFunc[S any, T any] func(S) (T, error)

type waitHook[K comparable, S any, T any] struct {
	key              K
	normalPathFunc   func(S) error
	abnormalPathFunc func(T) error
	callParams       []T
	unsatisfiedCount int
}

type Hook[K comparable, S any, T any] interface {
	NormalPath(S) error
	AbnormalPath(T) error
	Requirements() []K
}

func NewConditionJudger[K comparable, S any, T any](hookMap map[K]Hook[K, S, T], prepreProcessFunc PreProcessFunc[S, T]) *ConditionJudger[K, S, T] {
	satisfiedHooks := make(map[K]func(S) error, len(hookMap))
	unsatisfiedHooks := make(map[K][]*waitHook[K, S, T])
	hooks := make(map[K]*waitHook[K, S, T], len(hookMap))
	for key, hook := range hookMap {
		requirements := hook.Requirements()

		if len(requirements) == 0 {
			satisfiedHooks[key] = hook.NormalPath
			continue
		}

		hookValue := &waitHook[K, S, T]{
			key:              key,
			normalPathFunc:   hook.NormalPath,
			abnormalPathFunc: hook.AbnormalPath,
			unsatisfiedCount: len(requirements),
		}
		hooks[key] = hookValue
		for _, requirePart := range requirements {
			unsatisfiedHooks[requirePart] = append(unsatisfiedHooks[requirePart], hookValue)
		}
	}

	return &ConditionJudger[K, S, T]{
		preProcessFunc:   prepreProcessFunc,
		satisfiedHooks:   satisfiedHooks,
		unsatisfiedHooks: unsatisfiedHooks,
		hooks:            hooks,
	}
}

func (w *ConditionJudger[K, S, T]) HookEvent(key K, value S) (bool, error) {
	if fn := w.satisfiedHooks[key]; fn != nil {
		err := fn(value)
		if err != nil {
			return false, fmt.Errorf("failed to execute hook: %w", err)
		}

		return true, nil
	}

	callParam, err := w.preProcessFunc(value)
	if err != nil {
		return false, fmt.Errorf("failed to pre-process: %w", err)
	}

	hookValue := w.hooks[key]
	hookValue.callParams = append(hookValue.callParams, callParam)

	return false, nil
}

func (w *ConditionJudger[K, S, T]) KeyEvent(key K) error {
	hooks := w.unsatisfiedHooks[key]

	var errs []error
	for _, hook := range hooks {
		if hook.unsatisfiedCount <= 0 {
			continue
		}
		hook.unsatisfiedCount--
		if hook.unsatisfiedCount > 0 {
			continue
		}

		w.satisfiedHooks[hook.key] = hook.normalPathFunc

		for _, param := range hook.callParams {
			err := hook.abnormalPathFunc(param)
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to execute hook(%v): %w", key, err))
			}
		}
		hook.callParams = nil
	}
	if len(errs) != 0 {
		return errors.Join(errs...)
	}

	return nil
}
