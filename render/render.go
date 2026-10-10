// Package render는 타깃 렌더러가 공유하는 계약을 담는다.
//
// 새 타깃을 붙이는 일은 Renderer 하나를 더 만드는 일이다 — 덱 모델을 읽고 파일 몇 개를
// 낸다. 파일 쓰기와 순서 결정은 파이프라인이 하므로 렌더러는 순수 함수로 유지된다
// (테스트가 디스크를 건드리지 않는다).
package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// Target은 렌더 타깃 이름이다. CLI `--target` 값과 같은 문자열을 쓴다.
type Target string

const (
	// TargetHTML은 웹 프레젠테이션이다.
	TargetHTML Target = "html"
	// TargetPPTX는 편집 가능한 네이티브 PPTX다.
	TargetPPTX Target = "pptx"
)

// DefaultTargets는 `--target`을 주지 않았을 때 렌더하는 타깃이다.
func DefaultTargets() []Target { return []Target{TargetHTML} }

// ParseTargets는 `html,pptx` 형태의 목록을 파싱한다. 중복은 접고, 알 수 없는 이름은
// 에러다 — 오타가 "아무것도 안 만들어졌다"로 나타나면 안 된다.
func ParseTargets(list string) ([]Target, error) {
	var out []Target
	seen := map[Target]bool{}
	for _, raw := range strings.Split(list, ",") {
		name := Target(strings.TrimSpace(raw))
		if name == "" {
			continue
		}
		switch name {
		case TargetHTML, TargetPPTX:
		default:
			return nil, fmt.Errorf("unknown target %q (%q, %q)", string(name), TargetHTML, TargetPPTX)
		}
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no target given")
	}
	return out, nil
}

// Files는 산출 파일 묶음이다 — 출력 디렉토리 기준 상대 경로에서 내용으로.
type Files map[string][]byte

// Paths는 정렬된 경로 목록이다. 골든 비교와 로그 출력이 실행마다 같은 순서를 보게 한다.
func (f Files) Paths() []string {
	paths := make([]string, 0, len(f))
	for p := range f {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}

// Renderer는 덱 모델을 타깃 산출물로 바꾼다.
type Renderer interface {
	// Target은 이 렌더러가 담당하는 타깃이다.
	Target() Target
	// Render는 덱 모델을 읽어 산출 파일을 만든다. 디스크를 건드리지 않는다.
	Render(deck *soml.Deck) (Files, error)
}
