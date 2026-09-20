package core

import (
	"reflect"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// 텍스트 요소의 **가로 크기**가 확정 박스(soml.Appearance.Box)에 실린다 (ADR 0043).
//
// 값 문법은 컨테이너·도형과 **같은 헬퍼**(boxAttr)가 해석하므로 같은 문자열이 같은 값이
// 된다 — TestResolveContainerLayout의 크기 행들과 대조하면 표가 한 벌인 것이 보인다.
func TestResolveTextBoxSize(t *testing.T) {
	cases := []struct {
		name string
		attr string
		tag  string
		src  string
		want soml.Box
	}{
		{"text w 수치", "w", "text", `<text w="140">라벨</text>`, soml.Box{Width: ptr(soml.Px(140))}},
		{"text w 백분율", "w", "text", `<text w="40%">라벨</text>`, soml.Box{Width: ptr(soml.Percent(40))}},
		// `fill`은 해당 축의 100%로 눕는다 (ADR 0015) — 컨테이너·도형과 같은 계약이다.
		{"text w fill", "w", "text", `<text w="fill">라벨</text>`, soml.Box{Width: ptr(soml.Percent(100))}},
		{"블록 max-w", "max-w", "p", `<p max-w="50%">본문</p>`, soml.Box{MaxWidth: ptr(soml.Percent(50))}},
		{"리스트 컨테이너 min-w", "min-w", "ul", `<ul min-w="200"><li>항목</li></ul>`,
			soml.Box{MinWidth: ptr(soml.Px(200))}},
		{"셋을 함께", "", "text", `<text w="140" min-w="80" max-w="240">라벨</text>`,
			soml.Box{Width: ptr(soml.Px(140)), MinWidth: ptr(soml.Px(80)), MaxWidth: ptr(soml.Px(240))}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			elem := resolvedElement(t, layoutSlide(tc.src), tc.tag)
			box := elem.Box()
			if box == nil {
				t.Fatalf("<%s>에 확정 박스가 없다", tc.tag)
			}
			if !reflect.DeepEqual(*box, tc.want) {
				t.Errorf("박스 = %+v, want %+v", *box, tc.want)
			}
			// 크기는 컨테이너 배치도 자식 배치도 아니다 — 세 채널이 섞이면 렌더러가 자기
			// 폭을 부모의 배치로 낸다.
			if elem.Container() != nil {
				t.Errorf("컨테이너 배치가 붙었다: %+v", *elem.Container())
			}
			if elem.Placement() != nil {
				t.Errorf("자식 배치가 붙었다: %+v", *elem.Placement())
			}
		})
	}

	// 열린 속성 전부에 해석 테스트가 있다 — 카탈로그가 연 속성을 값 층이 해석하지 않으면
	// 저자가 쓴 폭이 조용히 사라진다 (TestResolveContainerLayout의 커버리지 판정과 짝이다).
	covered := map[string]bool{}
	for _, tc := range cases {
		covered[tc.attr] = true
	}
	for _, attr := range soml.TextSizeAttrs() {
		if !covered[attr] {
			t.Errorf("텍스트 크기 속성 %q의 해석 테스트가 없다", attr)
		}
	}
}

// 크기 속성을 쓰지 않은 요소의 박스는 **nil로 남는다** — 배치 해석이 붙었다는 이유로
// 산출물이 달라지지 않는다는 계약(resolveLayout의 nil)이 새 필드에도 그대로 적용된다.
func TestResolveTextBoxNilWithoutSize(t *testing.T) {
	cases := []struct {
		name string
		tag  string
		src  string
	}{
		{"속성 없음", "text", `<text>라벨</text>`},
		{"외관만", "text", `<text size="16" align="center">라벨</text>`},
		{"자식 배치만", "p", `<row><p grow="1">내용</p></row>`},
		// 컨테이너와 도형의 크기는 여기 실리지 않는다 — 각자 Container.Box·Shape.Box를 갖는다.
		// 한 값이 두 자리에 담기면 렌더러가 둘 다 읽어 선언이 두 번 나간다.
		{"컨테이너", "row", `<row w="320"><p>내용</p></row>`},
		{"도형", "shape", `<shape type="rect" w="140" h="56"><text>라벨</text></shape>`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			elem := resolvedElement(t, layoutSlide(tc.src), tc.tag)
			if box := elem.Box(); box != nil {
				t.Errorf("확정 박스가 붙었다: %+v", *box)
			}
		})
	}
	// 컨테이너·도형의 크기는 여전히 자기 자리에 실린다 (위 케이스가 "아무데도 없다"를 뜻하지
	// 않는다는 확인).
	if box := resolvedElement(t, layoutSlide(`<row w="320"><p>내용</p></row>`), "row").Container().Box; box.Width == nil {
		t.Error("<row>의 w가 컨테이너 박스에서 사라졌다")
	}
	if shape := resolvedElement(t, layoutSlide(`<shape type="rect" w="140"><text>라벨</text></shape>`), "shape").ShapeStyle(); shape.Width == nil {
		t.Error("<shape>의 w가 도형 박스에서 사라졌다")
	}
}

// 값 진단도 컨테이너와 같은 문구를 쓴다 — 같은 헬퍼가 해석하므로 저자가 `<row w="md">`와
// `<text w="md">`에서 다른 설명을 읽을 이유가 없다. 틀린 값은 확정 박스에 실리지 않는다.
func TestResolveTextBoxDiagnostics(t *testing.T) {
	cases := []struct {
		name     string
		tag      string
		content  string
		col      int
		contains string
	}{
		{"크기에 쓴 공간 토큰", "text", `<text w="md">라벨</text>`, 12, "gap·pad에만"},
		{"단위 접미사", "text", `<text w="140px">라벨</text>`, 12, "단위 없는 논리 px"},
		{"음수", "p", `<p max-w="-10">본문</p>`, 13, "0보다 작을 수 없다"},
		{"값 없음", "text", `<text w>라벨</text>`, 9, "값이 필요하다"},
		{"빈 값", "ul", `<ul min-w=""><li>항목</li></ul>`, 14, "비어 있다"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			deck, diags := resolveSlide(t, layoutSlide(tc.content))
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
			}
			diag := diags[0]
			if diag.Severity != soml.SeverityError || !strings.Contains(diag.Message, tc.contains) {
				t.Errorf("진단 = %q, want %q 포함", diag.String(), tc.contains)
			}
			if diag.Span.Start.Line != 2 || diag.Span.Start.Col != tc.col {
				t.Errorf("위치 = %s, want 2:%d", diag.Span.Start, tc.col)
			}
			if box := findElement(t, deck, tc.tag).Box(); box != nil {
				t.Errorf("무효한 크기가 실렸다: %+v", *box)
			}
		})
	}
}
