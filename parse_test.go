package formstream

import (
	"errors"
	"strings"
	"testing"

	"github.com/mazrean/formstream/internal/condition_judge/mock"
	"go.uber.org/mock/gomock"
)

var errTest = errors.New("test error")

func TestParser_Parse(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		inputFormData  string
		outputValueMap map[string]Value
		mockSetup      func(*mock.MockIConditionJudger[string, *normalParam, *abnormalParam])
		err            error
	}{
		{
			name: "value only",
			inputFormData: "--boundary\n" +
				"Content-Disposition: form-data; name=\"field1\"\n" +
				"\n" +
				"field1Value\n" +
				"--boundary--\n",
			outputValueMap: map[string]Value{
				"field1": {
					content: []byte("field1Value"),
					header:  newHeader(map[string][]string{"Content-Disposition": {"form-data; name=\"field1\""}}),
				},
			},
			mockSetup: func(m *mock.MockIConditionJudger[string, *normalParam, *abnormalParam]) {
				m.EXPECT().IsHookExist("field1").Return(false)
				m.EXPECT().KeyEvent("field1").Return(nil)
			},
		},
		{
			name: "stream only",
			inputFormData: "--boundary\n" +
				"Content-Disposition: form-data; name=\"stream1\"; filename=\"test.txt\"\n" +
				"Content-Type: text/plain\n" +
				"\n" +
				"stream1Value\n" +
				"--boundary--\n",
			outputValueMap: map[string]Value{},
			mockSetup: func(m *mock.MockIConditionJudger[string, *normalParam, *abnormalParam]) {
				m.EXPECT().IsHookExist("stream1").Return(true)
				m.EXPECT().HookEvent("stream1", gomock.Any()).Return(true, nil)
				m.EXPECT().KeyEvent("stream1").Return(nil)
			},
		},
		{
			name: "value and stream",
			inputFormData: "--boundary\n" +
				"Content-Disposition: form-data; name=\"field1\"\n" +
				"\n" +
				"field1Value\n" +
				"--boundary\n" +
				"Content-Disposition: form-data; name=\"stream1\"; filename=\"test.txt\"\n" +
				"Content-Type: text/plain\n" +
				"\n" +
				"stream1Value\n" +
				"--boundary--\n",
			outputValueMap: map[string]Value{
				"field1": {
					content: []byte("field1Value"),
					header:  newHeader(map[string][]string{"Content-Disposition": {"form-data; name=\"field1\""}}),
				},
			},
			mockSetup: func(m *mock.MockIConditionJudger[string, *normalParam, *abnormalParam]) {
				m.EXPECT().IsHookExist("field1").Return(false)
				m.EXPECT().KeyEvent("field1").Return(nil)
				m.EXPECT().IsHookExist("stream1").Return(true)
				m.EXPECT().HookEvent("stream1", gomock.Any()).Return(true, nil)
				m.EXPECT().KeyEvent("stream1").Return(nil)
			},
		},
		{
			name: "key error",
			inputFormData: "--boundary\n" +
				"Content-Disposition: form-data; name=\"field1\"\n" +
				"\n" +
				"field1Value\n" +
				"--boundary--\n",
			outputValueMap: map[string]Value{
				"field1": {
					content: []byte("field1Value"),
					header:  newHeader(map[string][]string{"Content-Disposition": {"form-data; name=\"field1\""}}),
				},
			},
			mockSetup: func(m *mock.MockIConditionJudger[string, *normalParam, *abnormalParam]) {
				m.EXPECT().IsHookExist("field1").Return(false)
				m.EXPECT().KeyEvent("field1").Return(errTest)
			},
			err: errTest,
		},
		{
			name: "hook error",
			inputFormData: "--boundary\n" +
				"Content-Disposition: form-data; name=\"stream1\"; filename=\"test.txt\"\n" +
				"Content-Type: text/plain\n" +
				"\n" +
				"stream1Value\n" +
				"--boundary--\n",
			outputValueMap: map[string]Value{},
			mockSetup: func(m *mock.MockIConditionJudger[string, *normalParam, *abnormalParam]) {
				m.EXPECT().IsHookExist("stream1").Return(true)
				m.EXPECT().HookEvent("stream1", gomock.Any()).Return(true, errTest)
			},
			err: errTest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockJudger := mock.NewMockIConditionJudger[string, *normalParam, *abnormalParam](ctrl)
			tc.mockSetup(mockJudger)

			parser := &Parser{
				boundary: "boundary",
				valueMap: make(map[string][]Value),
			}
			err := parser.parse(strings.NewReader(tc.inputFormData), mockJudger)

			if !errors.Is(err, tc.err) {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
