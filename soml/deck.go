// Package soml은 SOML 파서와 덱 모델을 담는다.
//
// 덱 모델(Deck)이 이 제품의 중심이다 — SOML은 저작 문법이고, 각 타깃 렌더러는 덱 모델을
// 소비한다. 그래서 이 패키지는 외부에서 임포트 가능한 공개 API로 유지한다 (생성기·에이전트
// 같은 제3자 도구가 재사용한다).
package soml

import "fmt"

// Aspect는 슬라이드 종횡비다. v1은 2종만 받는다 (PRD §13-9).
type Aspect string

const (
	// Aspect16x9는 논리 스테이지 1280×720이다.
	Aspect16x9 Aspect = "16:9"
	// Aspect4x3은 논리 스테이지 960×720이다.
	Aspect4x3 Aspect = "4:3"
)

// DefaultAspect는 aspect를 지정하지 않은 덱의 비율이다.
const DefaultAspect = Aspect16x9

// Stage는 논리 스테이지 크기다. 단위는 px 하나뿐이며(PRD §13-5) 저자·웹 렌더러·지오메트리
// 패스가 모두 이 좌표계를 쓴다. 타깃 단위(EMU, pt) 변환은 렌더러가 소유한다.
type Stage struct {
	Width  int
	Height int
}

// Stage는 종횡비에 대응하는 논리 스테이지를 돌려준다. 알 수 없는 비율은 에러다.
//
// 두 비율은 높이 720을 공유한다 — 폰트 크기와 수직 리듬이 비율을 넘어 그대로 통하고,
// px에 상수 9525를 곱하면 PowerPoint 표준 페이지 크기(13.333×7.5in / 10×7.5in)가
// 정확히 나온다 (docs/decisions/0002-logical-stage-size.md).
func (a Aspect) Stage() (Stage, error) {
	switch a {
	case Aspect16x9:
		return Stage{Width: 1280, Height: 720}, nil
	case Aspect4x3:
		return Stage{Width: 960, Height: 720}, nil
	default:
		return Stage{}, fmt.Errorf("알 수 없는 종횡비 %q (%q 또는 %q)", string(a), Aspect16x9, Aspect4x3)
	}
}

// Deck은 정규화된 덱 모델이다. 렌더러가 보는 유일한 입력이며, 여기 도달한 시점에는
// 합성 컴포넌트가 전개되고 외관이 해석되어 있다.
type Deck struct {
	Title  string
	Author string
	Lang   string
	Aspect Aspect
	Stage  Stage
	Slides []*Slide
}

// Slide는 슬라이드 하나다 — `.slide` 파일 하나에 대응한다.
type Slide struct {
	// Index는 0-based 렌더 순서다 (Skip된 슬라이드는 애초에 Deck에 없다).
	Index int
	// ID는 딥링크 앵커다. 생략 시 파일명에서 유도한다 (SlideFile.DerivedID).
	ID string
	// Source는 프로젝트 루트 기준 상대 경로다. 진단이 가리키는 파일.
	Source string
	// Chapter는 슬라이드가 속한 하위 디렉토리 경로다. 루트 직속이면 빈 문자열이다.
	Chapter string
	// Notes는 발표자 노트다 (`<note>`). PPTX에서는 노트 슬라이드가 된다.
	Notes string
	// Content는 슬라이드 콘텐츠 트리다 (`<note>`는 여기 없다 — Notes로 간다).
	Content []Node
	// Tokens는 이 슬라이드를 만든 원문 토큰 열이다. format이 원문을 되돌려 쓰고,
	// 진단이 위치를 찾는 근거다.
	Tokens Tokens
}

// Node는 슬라이드 콘텐츠 트리의 노드다. 구현체는 Element와 Text다 (node.go).
type Node interface {
	// NodeSpan은 이 노드를 만든 원문 구간이다 — 렌더 단계의 진단도 위치를 갖는다.
	NodeSpan() Span
}
