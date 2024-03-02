package formstream

import (
	"bytes"
	"errors"
	"io"
	"strconv"
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
		outputValueMap map[string][]Value
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
			outputValueMap: map[string][]Value{
				"field1": {
					{
						content: []byte("field1Value"),
						header:  newHeader(map[string][]string{"Content-Disposition": {"form-data; name=\"field1\""}}),
					},
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
			outputValueMap: map[string][]Value{},
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
			outputValueMap: map[string][]Value{
				"field1": {
					{
						content: []byte("field1Value"),
						header:  newHeader(map[string][]string{"Content-Disposition": {"form-data; name=\"field1\""}}),
					},
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
			outputValueMap: map[string][]Value{
				"field1": {
					{
						content: []byte("field1Value"),
						header:  newHeader(map[string][]string{"Content-Disposition": {"form-data; name=\"field1\""}}),
					},
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
			outputValueMap: map[string][]Value{},
			mockSetup: func(m *mock.MockIConditionJudger[string, *normalParam, *abnormalParam]) {
				m.EXPECT().IsHookExist("stream1").Return(true)
				m.EXPECT().HookEvent("stream1", gomock.Any()).Return(true, errTest)
			},
			err: errTest,
		},
		{
			name: "too many parts",
			inputFormData: "--boundary\n" +
				"Content-Disposition: form-data; name=\"field1\"\n" +
				"\n" +
				"field1Value\n" +
				"--boundary\n" +
				"Content-Disposition: form-data; name=\"field2\"\n" +
				"\n" +
				"field2Value\n" +
				"--boundary\n" +
				"Content-Disposition: form-data; name=\"field3\"\n" +
				"\n" +
				"field3Value\n" +
				"--boundary\n" +
				"Content-Disposition: form-data; name=\"field4\"\n" +
				"\n" +
				"field4Value\n" +
				"--boundary\n" +
				"Content-Disposition: form-data; name=\"field5\"\n" +
				"\n" +
				"field5Value\n" +
				"--boundary\n" +
				"Content-Disposition: form-data; name=\"field6\"\n" +
				"\n" +
				"field6Value\n" +
				"--boundary--\n",
			outputValueMap: map[string][]Value{
				"field1": {
					{
						content: []byte("field1Value"),
						header:  newHeader(map[string][]string{"Content-Disposition": {"form-data; name=\"field1\""}}),
					},
				},
				"field2": {
					{
						content: []byte("field2Value"),
						header:  newHeader(map[string][]string{"Content-Disposition": {"form-data; name=\"field2\""}}),
					},
				},
				"field3": {
					{
						content: []byte("field3Value"),
						header:  newHeader(map[string][]string{"Content-Disposition": {"form-data; name=\"field3\""}}),
					},
				},
				"field4": {
					{
						content: []byte("field4Value"),
						header:  newHeader(map[string][]string{"Content-Disposition": {"form-data; name=\"field4\""}}),
					},
				},
				"field5": {
					{
						content: []byte("field5Value"),
						header:  newHeader(map[string][]string{"Content-Disposition": {"form-data; name=\"field5\""}}),
					},
				},
			},
			mockSetup: func(m *mock.MockIConditionJudger[string, *normalParam, *abnormalParam]) {
				m.EXPECT().IsHookExist("field1").Return(false)
				m.EXPECT().KeyEvent("field1").Return(nil)
				m.EXPECT().IsHookExist("field2").Return(false)
				m.EXPECT().KeyEvent("field2").Return(nil)
				m.EXPECT().IsHookExist("field3").Return(false)
				m.EXPECT().KeyEvent("field3").Return(nil)
				m.EXPECT().IsHookExist("field4").Return(false)
				m.EXPECT().KeyEvent("field4").Return(nil)
				m.EXPECT().IsHookExist("field5").Return(false)
				m.EXPECT().KeyEvent("field5").Return(nil)
			},
			err: ErrTooManyParts,
		},
		{
			name: "too many headers",
			inputFormData: "--boundary\n" +
				"Content-Disposition: form-data; name=\"field1\"\n" +
				"a: field2\n" +
				"b: field3\n" +
				"c: field4\n" +
				"d: field5\n" +
				"e: field6\n" +
				"\n" +
				"field1Value\n" +
				"--boundary--\n",
			outputValueMap: map[string][]Value{},
			mockSetup:      func(_ *mock.MockIConditionJudger[string, *normalParam, *abnormalParam]) {},
			err:            ErrTooManyHeaders,
		},
		{
			name: "too large form",
			inputFormData: "--boundary\n" +
				"Content-Disposition: form-data; name=\"field1\"\n" +
				"\n" +
				strings.Repeat("a", 1024) + "\n" +
				"--boundary--\n",
			outputValueMap: map[string][]Value{},
			mockSetup: func(m *mock.MockIConditionJudger[string, *normalParam, *abnormalParam]) {
				m.EXPECT().IsHookExist("field1").Return(false)
			},
			err: ErrTooLargeForm,
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
				parserConfig: parserConfig{
					maxParts:       5,
					maxHeaders:     5,
					maxMemSize:     1024,
					maxMemFileSize: 1024,
				},
			}
			err := parser.parse(strings.NewReader(tc.inputFormData), mockJudger)

			for k, v := range tc.outputValueMap {
				if len(parser.valueMap[k]) != len(v) {
					t.Errorf("unexpected value count: expected %d, actual %d", len(v), len(parser.valueMap[k]))
				}

				for i, v := range parser.valueMap[k] {
					if string(v.content) != string(tc.outputValueMap[k][i].content) {
						t.Errorf("unexpected value: expected %s, actual %s", string(tc.outputValueMap[k][i].content), string(v.content))
					}

					if v.header.Name() != tc.outputValueMap[k][i].header.Name() {
						t.Errorf("unexpected value: expected %s, actual %s", tc.outputValueMap[k][i].header.Name(), v.header.Name())
					}

					if v.header.FileName() != tc.outputValueMap[k][i].header.FileName() {
						t.Errorf("unexpected value: expected %s, actual %s", tc.outputValueMap[k][i].header.FileName(), v.header.FileName())
					}

					if v.header.ContentType() != tc.outputValueMap[k][i].header.ContentType() {
						t.Errorf("unexpected value: expected %s, actual %s", tc.outputValueMap[k][i].header.ContentType(), v.header.ContentType())
					}
				}
			}

			if !errors.Is(err, tc.err) {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestPreProcessor_run(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		value       []byte
		header      Header
		readerTypes []string
		close       bool
		err         error
	}{
		{
			name:  "on memory",
			value: []byte("value"),
			header: Header{
				dispositionParams: map[string]string{
					"name":     "field",
					"filename": "file.txt",
				},
				header: map[string][]string{
					"Content-Type":        {"text/plain"},
					"Content-Disposition": {"form-data; name=\"field\"; filename=\"file.txt\""},
				},
			},
			readerTypes: []string{"memory"},
		},
		{
			name:  "on disk",
			value: []byte(strings.Repeat("a", 64)),
			header: Header{
				dispositionParams: map[string]string{
					"name":     "field",
					"filename": "file.txt",
				},
				header: map[string][]string{
					"Content-Type":        {"text/plain"},
					"Content-Disposition": {"form-data; name=\"field\"; filename=\"file.txt\""},
				},
			},
			readerTypes: []string{"disk"},
		},
		{
			name:  "on memory(max)",
			value: []byte(strings.Repeat("a", 32)),
			header: Header{
				dispositionParams: map[string]string{
					"name":     "field",
					"filename": "file.txt",
				},
				header: map[string][]string{
					"Content-Type":        {"text/plain"},
					"Content-Disposition": {"form-data; name=\"field\"; filename=\"file.txt\""},
				},
			},
			readerTypes: []string{"memory"},
		},
		{
			name:  "on disk(min)",
			value: []byte(strings.Repeat("a", 33)),
			header: Header{
				dispositionParams: map[string]string{
					"name":     "field",
					"filename": "file.txt",
				},
				header: map[string][]string{
					"Content-Type":        {"text/plain"},
					"Content-Disposition": {"form-data; name=\"field\"; filename=\"file.txt\""},
				},
			},
			readerTypes: []string{"disk"},
		},
		{
			name:  "on memory and disk",
			value: []byte(strings.Repeat("a", 17)),
			header: Header{
				dispositionParams: map[string]string{
					"name":     "field",
					"filename": "file.txt",
				},
				header: map[string][]string{
					"Content-Type":        {"text/plain"},
					"Content-Disposition": {"form-data; name=\"field\"; filename=\"file.txt\""},
				},
			},
			readerTypes: []string{"memory", "disk", "disk"},
		},
		{
			name:  "on memory(close)",
			value: []byte(strings.Repeat("a", 17)),
			header: Header{
				dispositionParams: map[string]string{
					"name":     "field",
					"filename": "file.txt",
				},
				header: map[string][]string{
					"Content-Type":        {"text/plain"},
					"Content-Disposition": {"form-data; name=\"field\"; filename=\"file.txt\""},
				},
			},
			readerTypes: []string{"memory", "memory"},
			close:       true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pp := &preProcessor{
				config: &parserConfig{
					maxMemSize:     32,
					maxMemFileSize: 32,
				},
			}

			for i, readerType := range tt.readerTypes {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					r, err := pp.run(&normalParam{
						r: strings.NewReader(string(tt.value)),
						h: tt.header,
					})
					if !errors.Is(err, tt.err) {
						t.Errorf("unexpected error: %v", err)
					}
					if err != nil {
						return
					}

					if r.content == nil {
						t.Error("unexpected nil")
					}

					b := bytes.NewBuffer(nil)
					_, err = io.Copy(b, r.content)
					if err != nil {
						t.Fatalf("failed to copy: %s", err)
					}
					if b.String() != string(tt.value) {
						t.Errorf("unexpected value: expected %s, actual %s", string(tt.value), b.String())
					}

					_, ok := r.content.(customReadCloser)
					switch readerType {
					case "memory":
						if !ok {
							t.Errorf("unexpected reader type: %T", r.content)
						}
					case "disk":
						if ok {
							t.Errorf("unexpected reader type: %T", r.content)
						}
					}

					if tt.close {
						err = r.content.Close()
						if err != nil {
							t.Errorf("unexpected error: %v", err)
						}
					}
				})
			}
		})
	}
}
