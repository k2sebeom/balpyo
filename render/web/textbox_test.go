package web

import (
	"strings"
	"testing"
)

// 텍스트 요소의 가로 크기가 인라인 style로 나간다 — 값 컴파일은 컨테이너·도형과
// 같은 함수(boxSizeCSS)가 하므로 `<row w="140">`과 `<text w="140">`이 같은 선언을 낸다.
func TestRenderTextBoxSizeDecls(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"수치", `<text w="140">라벨</text>`, "width: 140px"},
		{"백분율", `<text w="40%">라벨</text>`, "width: 40%"},
		{"fill은 100%로 눕는다", `<text w="fill">라벨</text>`, "width: 100%"},
		{"백분율 max-w", `<text max-w="50%">라벨</text>`, "max-width: 50%"},
		{"블록", `<p min-w="200">본문</p>`, "min-width: 200px"},
		{"리스트 컨테이너", `<ul w="480"><li>항목</li></ul>`, "width: 480px"},
		{"셋을 함께 — 순서는 w → min-w → max-w다", `<text w="140" max-w="240" min-w="80">라벨</text>`,
			"width: 140px; min-width: 80px; max-width: 240px"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := renderString(t, subsetDeck(t, tc.body))
			if tag := openTagOf(t, out, "s0.0"); !strings.Contains(tag, tc.want) {
				t.Errorf("여는 태그 = %s\nwant %q 포함", tag, tc.want)
			}
		})
	}
}

// 선언 순서가 속성 표의 순서를 따라간다: **외관 → 박스(가로) → 슬롯 → 자식 배치 → 탈출구**.
// 순서가 섞이면 두 채널을 함께 쓴 요소의 골든을 읽을 수 없고, 같은 프로퍼티가 겹칠 때
// 탈출구가 맨 끝이라는 계약도 흔들린다.
func TestRenderTextBoxDeclOrder(t *testing.T) {
	out := renderString(t, layoutDeck(t, "title",
		`<text slot="title" align="center" w="140" grow="1" style="opacity: 0.5">라벨</text>`))
	tag := openTagOf(t, out, "s0.0")
	order := []string{"text-align: center", "width: 140px", "grid-area: title", "flex-grow: 1", "opacity: 0.5"}
	at := 0
	for _, decl := range order {
		found := strings.Index(tag[at:], decl)
		if found < 0 {
			t.Fatalf("선언 %q가 %q 뒤에 없다:\n%s", decl, order[0], tag)
		}
		at += found + len(decl)
	}
}

// 슬롯 영역 자식의 폭 리셋이 **슬롯 덱에만** 나간다 (세 번째 리셋 문맥).
//
// 이 규칙이 없으면 테마가 `*.slots.css`에 쓴 정렬(`justify-items: center`)이 base 층
// `.balpyo-text { width: 100% }`에 지워진다 — 저자가 슬롯 자식마다 `align="center"`를 손으로
// 다는 것이 그 증상이다.
func TestSlotModeResetsTextWidth(t *testing.T) {
	const rule = ".balpyo-stage.balpyo-slots > :is(.balpyo-text, .balpyo-list) { width: auto; }"

	slots := between(t, renderString(t, layoutDeck(t, "title", `<h1 slot="title">제목</h1>`)), "<style>", "</style>")
	if !strings.Contains(slots, rule) {
		t.Errorf("슬롯 덱에 폭 리셋 규칙 %q가 없다:\n%s", rule, slots)
	}
	// base 층의 확정 폭은 그대로 있다 — 리셋은 그 규칙을 **문맥에서만** 되돌린다.
	if !strings.Contains(slots, "width: 100%") {
		t.Error("base 층의 .balpyo-text 확정 폭이 사라졌다")
	}

	// 흐름 모드 덱(같은 레이아웃, 슬롯 없음)과 레이아웃 없는 덱에는 나가지 않는다.
	flow := between(t, renderString(t, layoutDeck(t, "title", `<h1>제목</h1>`)), "<style>", "</style>")
	if strings.Contains(flow, rule) {
		t.Errorf("흐름 모드 덱에 폭 리셋 규칙이 나갔다:\n%s", flow)
	}
	plain := between(t, renderString(t, subsetDeck(t, `<p>본문</p>`)), "<style>", "</style>")
	if strings.Contains(plain, rule) {
		t.Errorf("레이아웃 없는 덱에 폭 리셋 규칙이 나갔다:\n%s", plain)
	}
}
