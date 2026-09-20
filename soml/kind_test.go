package soml

import (
	"strings"
	"testing"
)

// 카탈로그는 스스로 닫혀 있어야 한다 — 이름 색인(elementKinds)에 있는 종류가 ElementKinds()에
// 빠지면 렌더러 대조 테스트(render 패키지)가 그 종류를 아예 보지 못한다. 카탈로그를 한 곳에
// 둔 이유가 무너지는 자리이므로 목록을 서로 대조한다.
func TestElementKindsCoverCatalog(t *testing.T) {
	listed := make(map[ElementKind]bool, len(ElementKinds()))
	for _, kind := range ElementKinds() {
		if kind == KindNone {
			t.Error("ElementKinds()에 KindNone이 있다 — 카탈로그 밖의 값이다")
		}
		if listed[kind] {
			t.Errorf("ElementKinds()에 %v가 두 번 있다", kind)
		}
		listed[kind] = true
	}
	for name, kind := range elementKinds {
		if !listed[kind] {
			t.Errorf("<%s>의 종류 %v가 ElementKinds()에 없다", name, kind)
		}
	}
	// 종류마다 이름이 하나 이상 있어야 한다 — 이름 없는 종류는 어휘가 아니다.
	for _, kind := range ElementKinds() {
		if len(ElementNamesOf(kind)) == 0 {
			t.Errorf("%v에 해당하는 태그 이름이 없다", kind)
		}
		if kind.String() == "Unknown" {
			t.Errorf("%d에 이름표가 없다 (elementKindNames)", int(kind))
		}
	}
}

// ElementNamesOf는 카탈로그 순서를 지킨다 — 진단 메시지가 이 순서를 그대로 싣는다.
func TestElementNamesOf(t *testing.T) {
	tests := []struct {
		kind ElementKind
		want string
	}{
		{KindBlock, "h1 h2 h3 p blockquote pre"},
		{KindListContainer, "ul ol"},
		{KindListItem, "li"},
		{KindInline, "strong em code a br span"},
		{KindTable, "table"},
		{KindTableRow, "tr"},
		{KindTableCell, "th td"},
	}
	for _, tc := range tests {
		if got := strings.Join(ElementNamesOf(tc.kind), " "); got != tc.want {
			t.Errorf("ElementNamesOf(%v) = %q, want %q", tc.kind, got, tc.want)
		}
	}
	if got := ElementNamesOf(KindNone); got != nil {
		t.Errorf("ElementNamesOf(KindNone) = %v, want nil (카탈로그 밖이다)", got)
	}
}

// 지금 그리지 못하는 종류는 표 3종뿐이다 — 이 목록이 줄어드는 것이 P2.1의 완료 신호다.
func TestKindRendered(t *testing.T) {
	for _, kind := range ElementKinds() {
		want := true
		switch kind {
		case KindTable, KindTableRow, KindTableCell:
			want = false
		}
		if got := KindRendered(kind); got != want {
			t.Errorf("KindRendered(%v) = %v, want %v", kind, got, want)
		}
	}
	// 카탈로그 밖(<slide>·<text>·<note>)은 각자의 규칙을 따른다.
	if !KindRendered(KindNone) {
		t.Error("KindRendered(KindNone)은 참이어야 한다")
	}
}
