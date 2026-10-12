package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path"
)

// 이 파일은 **병합 결과의 에러를 고칠 파일로 되돌리는 길**이다.
//
// 테마의 값 어휘 · 완전성 검사는 `extends` 체인을 병합한 **결과**에 한 번 돈다 (buildThemePackage).
// 병합 결과에는 파일이 없으므로 에러가 그대로 나가면 진단이 이름을 요청한 `balpyo.json`을 가리키고,
// 체인의 어느 테마가 틀렸는지도 말하지 못한다. 그래서 검사하는 자리가 에러에 **선언의 자리**(테마
// 루트 기준 파일 이름 + JSON 안의 경로)를 달고, 체인을 아는 로더가 그 경로를 실제로 적은 테마의
// 파일을 찾아 줄:칸으로 옮긴다.

// declStep은 JSON 안의 경로 한 걸음이다 — 객체의 키이거나, 배열에서 필드 값으로 고르는 원소다.
// 원소를 순번이 아니라 필드로 고르는 이유: 병합된 목록의 순번은 어느 파일의 순번도 아니다
// (이름 단위 병합이 자식의 항목을 부모 순서 사이에 끼운다).
type declStep struct {
	key   string
	match []string // 필드, 값, 필드, 값 … — 전부 같은 원소를 고른다
}

func atKey(key string) declStep { return declStep{key: key} }

// atEntry는 fields(필드, 값 쌍)가 모두 같은 배열 원소다. 값이 빈 필드는 "없음"과 같다.
func atEntry(fields ...string) declStep { return declStep{match: fields} }

// themeDeclError는 병합 결과를 검사하다 난 에러에 선언의 자리를 단 것이다. 메시지는 감싼 에러의
// 것 그대로다 — 자리는 진단의 파일 · 줄:칸이 말한다.
type themeDeclError struct {
	file string // 테마 루트 기준 파일 이름 (`components.json`)
	path []declStep
	err  error
}

func (e *themeDeclError) Error() string { return e.err.Error() }
func (e *themeDeclError) Unwrap() error { return e.err }

// declAt은 err의 선언 경로 **앞**에 steps를 붙인다 — 안쪽 함수가 필드(`fill`)를, 바깥 함수가 그
// 필드를 담은 항목(`shape.variants["card"]`)을 안다. 자리가 없는 에러에는 자리를 새로 단다.
func declAt(err error, steps ...declStep) error {
	if err == nil {
		return nil
	}
	var de *themeDeclError
	if errors.As(err, &de) {
		de.path = append(append([]declStep(nil), steps...), de.path...)
		return err
	}
	return &themeDeclError{path: steps, err: err}
}

// declIn은 err가 사는 파일을 정한다. 이미 파일이 정해진 에러는 그대로 둔다 (안쪽이 더 잘 안다).
func declIn(file string, err error) error {
	if err == nil {
		return nil
	}
	var de *themeDeclError
	if errors.As(err, &de) {
		if de.file == "" {
			de.file = file
		}
		return err
	}
	return &themeDeclError{file: file, err: err}
}

// buildThemeChain은 뿌리 → 잎 체인을 병합해 패키지로 만든다. 검사가 실패하면 에러를 선언한
// 테마의 파일로 옮긴다 (attributeThemeError).
func buildThemeChain(chain []rawTheme) (themePackage, error) {
	merged := chain[0]
	for _, child := range chain[1:] {
		merged = mergeRawTheme(merged, child)
	}
	pkg, err := buildThemePackage(merged)
	if err != nil {
		return pkg, attributeThemeError(chain, err)
	}
	return pkg, nil
}

// attributeThemeError는 선언의 자리가 달린 에러를 체인 안의 파일 하나로 옮긴다.
//
// 고르는 순서: 경로를 **끝까지** 담은 파일 중 잎에 가장 가까운 것 — 이름 단위 병합에서 그 이름을
// 이긴 선언이 거기 있다. 끝까지 담은 파일이 없으면(빠진 이름) 경로를 가장 깊이 담은 파일, 그것도
// 없으면 잎 테마의 그 파일 (줄:칸 없이) — 빠진 것을 더할 자리다. 내장 테마는 후보가 아니다: 내장
// 테마는 테스트가 완전성을 지키므로, 병합 결과가 틀렸다면 원인은 그 위에 얹은 테마다.
//
// 로컬 테마는 그 파일의 진단(themeFileError)이 되고, 설치한 테마는 홈 기준 경로라 프로젝트의 파일
// 자리로 찍지 않고 테마 이름과 파일 · 줄:칸을 메시지에 담는다 (readThemeChain과 같은 태도).
func attributeThemeError(chain []rawTheme, err error) error {
	var de *themeDeclError
	if !errors.As(err, &de) || de.file == "" {
		return err
	}
	owner, offset, best := -1, -1, -1
	for i := len(chain) - 1; i >= 0; i-- {
		raw := chain[i]
		if raw.origin == originBuiltin {
			continue
		}
		data, ok := raw.sources[de.file]
		if !ok {
			continue
		}
		at, depth := locateDecl(data, de.path)
		if depth > best {
			owner, offset, best = i, at, depth
		}
		if depth == len(de.path) {
			break
		}
	}
	if owner < 0 {
		// 파일을 가진 테마가 없다 — 잎(요청한 테마)이 그 파일을 더할 자리다.
		for i := len(chain) - 1; i >= 0; i-- {
			if chain[i].origin != originBuiltin {
				owner = i
				break
			}
		}
		if owner < 0 {
			return err
		}
	}
	raw := chain[owner]
	fe := &themeFileError{file: path.Join(raw.dir, de.file), msg: err.Error()}
	if data, ok := raw.sources[de.file]; ok && offset >= 0 {
		fe.pos = offsetPos(data, offset)
	}
	if raw.origin != originLocal {
		return fmt.Errorf("theme %q: %s", raw.manifest.Name, fe)
	}
	return fe
}

// locateDecl은 path를 따라 data 안으로 내려가 닿은 가장 깊은 값의 시작 오프셋과 내려간 걸음 수를
// 돌려준다. 첫 걸음도 못 가면 문서 전체(오프셋 -1 · 0걸음)다 — 파일은 맞으므로 파일만 가리킨다.
func locateDecl(data []byte, steps []declStep) (offset, depth int) {
	offset = -1
	raw, base := json.RawMessage(data), 0
	for _, step := range steps {
		start, value, ok := stepInto(raw, step)
		if !ok {
			break
		}
		raw, base = value, base+start
		offset, depth = base, depth+1
	}
	return offset, depth
}

// stepInto는 값 raw 안에서 한 걸음을 찾아 그 값과 raw 안의 시작 오프셋을 돌려준다.
func stepInto(raw json.RawMessage, step declStep) (int, json.RawMessage, bool) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil {
		return 0, nil, false
	}
	switch {
	case tok == json.Delim('{') && step.key != "":
		for dec.More() {
			keyTok, err := dec.Token()
			if err != nil {
				return 0, nil, false
			}
			var value json.RawMessage
			if dec.Decode(&value) != nil {
				return 0, nil, false
			}
			if key, _ := keyTok.(string); key == step.key {
				end := int(dec.InputOffset())
				return end - len(value), value, true
			}
		}
	case tok == json.Delim('[') && step.match != nil:
		for dec.More() {
			var value json.RawMessage
			if dec.Decode(&value) != nil {
				return 0, nil, false
			}
			if entryMatches(value, step.match) {
				end := int(dec.InputOffset())
				return end - len(value), value, true
			}
		}
	}
	return 0, nil, false
}

// entryMatches는 배열 원소 하나가 필드 쌍을 전부 만족하는지 본다. 문자열이 아닌 필드는 빈 값이다.
func entryMatches(value json.RawMessage, fields []string) bool {
	var entry map[string]json.RawMessage
	if json.Unmarshal(value, &entry) != nil {
		return false
	}
	for i := 0; i+1 < len(fields); i += 2 {
		var got string
		if v, ok := entry[fields[i]]; ok {
			_ = json.Unmarshal(v, &got)
		}
		if got != fields[i+1] {
			return false
		}
	}
	return true
}
