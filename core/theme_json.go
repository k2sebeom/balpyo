package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/k2sebeom/balpyo/soml"
)

// themeFileError는 테마 파일 하나의 읽기 에러다 — 진단이 `balpyo.json`이 아니라 고칠 파일
// (`themes/<이름>/components.json`)과 그 안의 줄:칸을 가리키게 한다. 테마 이름을 요청한 자리는
// `balpyo.json`이지만, 저자가 열어야 하는 것은 틀린 키가 있는 파일이다.
type themeFileError struct {
	file string
	pos  soml.Pos
	msg  string
}

func (e *themeFileError) Error() string {
	if e.pos.IsValid() {
		return fmt.Sprintf("%s:%s: %s", e.file, e.pos, e.msg)
	}
	return fmt.Sprintf("%s: %s", e.file, e.msg)
}

// diagnostic은 에러를 그 파일의 진단으로 옮긴다.
func (e *themeFileError) diagnostic() soml.Diagnostic {
	return soml.Errorf(e.file, soml.Span{Start: e.pos, End: e.pos}, "%s", e.msg)
}

// decodeThemeJSON은 테마 JSON 파일 하나를 v에 읽는다. closed면 v의 스키마에 없는 키를 에러로 낸다
// — 조용히 무시하면 저자가 오버라이드가 먹었다고 착각한다. `$comment`로 시작하는 키는 어느 객체에서나
// 문서 주석이다 (`$comment-accent`처럼 주석을 값 옆에 여럿 둘 수 있게).
//
// 키 검사를 encoding/json의 DisallowUnknownFields에 맡기지 않는 이유: 그 에러는 키 이름만 말하고
// JSON 안의 경로도 줄 번호도 없어서, 같은 이름(`lineHeight`)이 여러 절에 있으면 저자가 항목을 하나씩
// 지워 가며 찾아야 한다. 그리고 접두사 규칙(`$comment*`)을 표현할 수 없다.
func decodeThemeJSON(data []byte, file string, v any, closed bool) error {
	if closed {
		if err := checkJSONKeys(data, 0, reflect.TypeOf(v).Elem(), "", data, true); err != nil {
			var fe *themeFileError
			if errors.As(err, &fe) {
				fe.file = file
				return fe
			}
			return &themeFileError{file: file, msg: err.Error()}
		}
	}
	if err := json.Unmarshal(data, v); err != nil {
		return jsonDecodeError(data, file, err)
	}
	return nil
}

// jsonDecodeError는 encoding/json의 디코드 에러를 file 안의 줄:칸과 JSON 저자의 말로 옮긴다 —
// 원래 문구(`cannot unmarshal number into Go struct field Config.title of type string`)는 Go의
// 구조체·타입 이름을 말하고 자리를 말하지 않아, 파일을 고치는 사람에게 쓸모가 없다.
func jsonDecodeError(data []byte, file string, err error) *themeFileError {
	fe := &themeFileError{file: file, msg: err.Error()}
	var syntax *json.SyntaxError
	var typ *json.UnmarshalTypeError
	switch {
	case errors.As(err, &syntax):
		// Offset은 틀린 바이트를 읽은 **뒤**다.
		fe.pos = offsetPos(data, int(syntax.Offset)-1)
		fe.msg = "invalid JSON: " + syntax.Error()
	case errors.As(err, &typ):
		label, offset := locateValue(data, int(typ.Offset), typ.Field)
		fe.pos = offsetPos(data, offset)
		fe.msg = fmt.Sprintf("%s must be %s, not %s", label, typeWord(typ.Type), typ.Value)
	case errors.Is(err, io.ErrUnexpectedEOF):
		// json.Decoder는 끝나지 않은 값을 SyntaxError가 아니라 이 에러로 낸다.
		fe.msg = "invalid JSON: unexpected end of file"
	}
	return fe
}

// checkJSONKeys는 raw(파일 전체 data의 base 오프셋에서 시작하는 값)를 t의 스키마와 대조한다.
// comments면 `$comment`로 시작하는 키를 문서 주석으로 건너뛴다 — 테마 파일의 규칙이고, `balpyo.json`은
// 그 규칙 없이 닫혀 있다. 문법 오류는 여기서 말하지 않는다 — 뒤의 디코딩이 위치와 함께 말한다.
func checkJSONKeys(raw []byte, base int, t reflect.Type, at string, data []byte, comments bool) error {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Struct:
		dec := json.NewDecoder(bytes.NewReader(raw))
		if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
			return nil
		}
		for dec.More() {
			tok, err := dec.Token()
			if err != nil {
				return nil
			}
			key, _ := tok.(string)
			keyEnd := int(dec.InputOffset())
			keyStart := bytes.LastIndexByte(raw[:keyEnd-1], '"')
			var value json.RawMessage
			if err := dec.Decode(&value); err != nil {
				return nil
			}
			if comments && strings.HasPrefix(key, "$comment") {
				continue
			}
			field, ok := jsonField(t, key)
			if !ok {
				return &themeFileError{pos: offsetPos(data, base+keyStart),
					msg: fmt.Sprintf("unknown key %q at %s", key, joinPath(at, key))}
			}
			valueStart := base + int(dec.InputOffset()) - len(value)
			if err := checkJSONKeys(value, valueStart, field.Type, joinPath(at, key), data, comments); err != nil {
				return err
			}
		}
	case reflect.Slice:
		dec := json.NewDecoder(bytes.NewReader(raw))
		if tok, err := dec.Token(); err != nil || tok != json.Delim('[') {
			return nil
		}
		for i := 0; dec.More(); i++ {
			var value json.RawMessage
			if err := dec.Decode(&value); err != nil {
				return nil
			}
			valueStart := base + int(dec.InputOffset()) - len(value)
			if err := checkJSONKeys(value, valueStart, t.Elem(), at+elementLabel(value, i), data, comments); err != nil {
				return err
			}
		}
	}
	return nil
}

// jsonField는 key에 대응하는 필드를 찾는다 — encoding/json과 같이 태그 이름이 정확히 같은 것을
// 먼저, 없으면 대소문자를 무시하고 찾는다 (검사가 실제 디코딩보다 엄격하거나 느슨하면 안 된다).
func jsonField(t reflect.Type, key string) (reflect.StructField, bool) {
	var fold reflect.StructField
	found := false
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" {
			continue
		}
		if name == "" {
			name = f.Name
		}
		if name == key {
			return f, true
		}
		if !found && strings.EqualFold(name, key) {
			fold, found = f, true
		}
	}
	return fold, found
}

// elementLabel은 배열 원소의 경로 조각이다 — 이름이 있는 원소(variant · 역할 · 토큰)는 이름으로
// 부른다: `variants["stat"]`이 `variants[3]`보다 파일에서 찾기 쉽다.
func elementLabel(value json.RawMessage, i int) string {
	var named struct {
		Name string `json:"name"`
	}
	if json.Unmarshal(value, &named) == nil && named.Name != "" {
		return fmt.Sprintf("[%q]", named.Name)
	}
	return fmt.Sprintf("[%d]", i)
}

// locateValue는 오프셋 offset에서 끝나는(또는 그것을 품은) 가장 깊은 값을 찾아 키 검사와 같은 표기의 경로
// (`shape.variants["card"].fill`)와 그 값이 시작하는 오프셋을 돌려준다. 못 찾으면 fallback 경로와 -1이다.
//
// encoding/json의 UnmarshalTypeError.Offset은 틀린 값을 읽은 **뒤**의 오프셋이다. 경로를 Field 문자열에서
// 읽지 않는 이유는 Go 버전마다 그 모양이 다르다는 것이다 — 배열 순번을 담는 버전(`shape.variants.0.fill`)과
// 담지 않는 버전(`shape.variants.fill`)이 있어, 순번 없이는 어느 원소인지 정할 수 없다. 오프셋은 두 버전에서 같다.
func locateValue(data []byte, offset int, fallback string) (string, int) {
	at, raw, base := "", json.RawMessage(data), 0
	found := false
	for {
		dec := json.NewDecoder(bytes.NewReader(raw))
		tok, err := dec.Token()
		if err != nil {
			break
		}
		descended := false
		switch tok {
		case json.Delim('{'):
			for dec.More() && !descended {
				keyTok, err := dec.Token()
				if err != nil {
					break
				}
				var value json.RawMessage
				if dec.Decode(&value) != nil {
					break
				}
				end := base + int(dec.InputOffset())
				if start := end - len(value); start < offset && offset <= end {
					key, _ := keyTok.(string)
					at, raw, base, descended = joinPath(at, key), value, start, true
				}
			}
		case json.Delim('['):
			for i := 0; dec.More() && !descended; i++ {
				var value json.RawMessage
				if dec.Decode(&value) != nil {
					break
				}
				end := base + int(dec.InputOffset())
				if start := end - len(value); start < offset && offset <= end {
					at, raw, base, descended = at+elementLabel(value, i), value, start, true
				}
			}
		}
		if !descended {
			break
		}
		found = true
	}
	if !found {
		return fallback, -1
	}
	return at, base
}

// typeWord는 Go 타입을 JSON 저자의 말로 옮긴다.
func typeWord(t reflect.Type) string {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String:
		return "a string"
	case reflect.Bool:
		return "true or false"
	case reflect.Int, reflect.Int64, reflect.Int32:
		return "an integer"
	case reflect.Float64, reflect.Float32:
		return "a number"
	case reflect.Slice:
		return "an array"
	case reflect.Struct, reflect.Map:
		return "an object"
	}
	return t.String()
}

func joinPath(at, key string) string {
	if at == "" {
		return key
	}
	return at + "." + key
}

// offsetPos는 바이트 오프셋을 줄:칸으로 옮긴다 — 칸은 소스 진단과 같이 룬으로 센다.
func offsetPos(data []byte, offset int) soml.Pos {
	if offset < 0 || offset > len(data) {
		return soml.Pos{}
	}
	pos := soml.StartPos()
	before := data[:offset]
	if nl := bytes.LastIndexByte(before, '\n'); nl >= 0 {
		pos.Line += bytes.Count(before, []byte{'\n'})
		before = before[nl+1:]
	}
	pos.Col += utf8.RuneCount(before)
	pos.Offset = offset
	return pos
}
