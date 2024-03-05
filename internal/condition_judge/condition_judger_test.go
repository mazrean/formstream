package conditionjudge_test

import (
	"errors"
	"fmt"
	"testing"

	conditionjudge "github.com/mazrean/formstream/internal/condition_judge"
)

var errTest = errors.New("test error")

type mockHook struct {
	requirements      []string
	err               error
	lastNormalValue   string
	lastAbnormalValue string
}

func (h *mockHook) NormalPath(value string) error {
	if h.err != nil {
		return h.err
	}

	h.lastNormalValue = value
	return nil
}

func (h *mockHook) AbnormalPath(value string) error {
	if h.err != nil {
		return h.err
	}

	h.lastAbnormalValue = value
	return nil
}

func (h *mockHook) Requirements() []string {
	return h.requirements
}

func preProcessFunc(value string) (string, error) {
	return fmt.Sprintf("abnormal:%s", value), nil
}

func TestConditionJudger(t *testing.T) {
	t.Parallel()

	type event struct {
		eventType string
		key       string
		value     string
		err       error
		runHook   string
		runType   string
		runValue  string
	}
	tests := map[string]struct {
		hooks  map[string]*mockHook
		events []event
	}{
		"normal": {
			hooks: map[string]*mockHook{
				"stream": {
					requirements: []string{"field"},
				},
			},
			events: []event{
				{"key", "field", "", nil, "", "", ""},
				{"hook", "stream", "one", nil, "stream", "normal", "one"},
			},
		},
		"error": {
			hooks: map[string]*mockHook{
				"stream": {
					err:          errTest,
					requirements: []string{"field"},
				},
			},
			events: []event{
				{"key", "field", "", nil, "", "", ""},
				{"hook", "stream", "one", errTest, "stream", "normal", "one"},
			},
		},
		"no hook": {
			hooks: map[string]*mockHook{},
			events: []event{
				{"key", "field", "", nil, "", "", ""},
				{"hook", "stream", "one", conditionjudge.ErrNoHooks, "", "", ""},
			},
		},
		"no call": {
			hooks: map[string]*mockHook{
				"stream": {
					requirements: []string{"field"},
				},
			},
			events: []event{
				{"key", "field", "", nil, "", "", ""},
			},
		},
		"abnormal": {
			hooks: map[string]*mockHook{
				"stream": {
					requirements: []string{"field"},
				},
			},
			events: []event{
				{"hook", "stream", "one", nil, "", "", ""},
				{"key", "field", "", nil, "stream", "abnormal", "abnormal:one"},
			},
		},
		"abnormal error": {
			hooks: map[string]*mockHook{
				"stream": {
					err:          errTest,
					requirements: []string{"field"},
				},
			},
			events: []event{
				{"hook", "stream", "one", nil, "", "", ""},
				{"key", "field", "", errTest, "stream", "abnormal", "abnormal:one"},
			},
		},
		"no requirements": {
			hooks: map[string]*mockHook{
				"stream": {},
			},
			events: []event{
				{"hook", "stream", "one", nil, "stream", "normal", "one"},
			},
		},
		"multiple requirements": {
			hooks: map[string]*mockHook{
				"stream": {
					requirements: []string{"field", "field2"},
				},
			},
			events: []event{
				{"key", "field", "", nil, "", "", ""},
				{"key", "field2", "", nil, "", "", ""},
				{"hook", "stream", "one", nil, "stream", "normal", "one"},
			},
		},
		"multiple hooks": {
			hooks: map[string]*mockHook{
				"stream": {
					requirements: []string{"field"},
				},
				"stream2": {
					requirements: []string{"field"},
				},
			},
			events: []event{
				{"key", "field", "", nil, "", "", ""},
				{"hook", "stream", "one", nil, "stream", "normal", "one"},
				{"hook", "stream2", "two", nil, "stream2", "normal", "two"},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			hookMap := make(map[string]conditionjudge.Hook[string, string, string], len(tt.hooks))
			for key, hook := range tt.hooks {
				hookMap[key] = hook
			}
			cj := conditionjudge.NewConditionJudger(hookMap, preProcessFunc)

		EVENT_LOOP:
			for _, event := range tt.events {
				switch event.eventType {
				case "key":
					err := cj.KeyEvent(event.key)
					if !errors.Is(err, event.err) {
						t.Errorf("unexpected error: %s", err)
					}
					if err != nil {
						break EVENT_LOOP
					}
				case "hook":
					ok, err := cj.HookEvent(event.key, event.value)
					if !errors.Is(err, event.err) {
						t.Errorf("unexpected error: %s", err)
					}
					if err != nil {
						break EVENT_LOOP
					}

					if ok != (event.runType == "normal") {
						t.Errorf("unexpected normal run: %t", ok)
					}
				default:
					t.Fatalf("unknown event type: %s", event.eventType)
				}

				switch event.runType {
				case "normal":
					if tt.hooks[event.runHook].lastNormalValue != event.runValue {
						t.Errorf("unexpected normal value: %s", tt.hooks[event.runHook].lastNormalValue)
					}
				case "abnormal":
					if tt.hooks[event.runHook].lastAbnormalValue != event.runValue {
						t.Errorf("unexpected abnormal value: %s", tt.hooks[event.runHook].lastAbnormalValue)
					}
				}
			}
		})
	}
}
