package soml

import "testing"

// 태그 → 배치 흐름 대응은 soml이 소유한다 — core가 자기 표를 들면 두 벌이 되고, 갈라진
// 쪽은 "flex인데 grid로 그려지는" 조용한 실패가 된다.
func TestFlowOfCoversLayoutContainers(t *testing.T) {
	cases := []struct {
		name string
		want ContainerFlow
	}{
		{RowElement, FlowRow},
		{ColElement, FlowColumn},
		{GridElement, FlowGrid},
		{StackElement, FlowStack},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if KindOf(tc.name) != KindLayoutContainer {
				t.Fatalf("%q는 레이아웃 컨테이너가 아니다", tc.name)
			}
			if got := FlowOf(tc.name); got != tc.want {
				t.Errorf("FlowOf(%q) = %d, want %d", tc.name, got, tc.want)
			}
		})
	}
	// 카탈로그의 컨테이너 전부가 위 표에 있어야 한다 — 컨테이너를 추가하고 흐름을 잊으면
	// FlowRow(제로값)로 조용히 떨어진다.
	for _, name := range layoutContainerElements {
		found := false
		for _, tc := range cases {
			found = found || tc.name == name
		}
		if !found {
			t.Errorf("컨테이너 %q의 흐름 대응 테스트가 없다", name)
		}
	}
}

// 접근자는 해석 전(Appearance == nil)에도 안전해야 한다 — 렌더러가 두 단계 nil 검사를
// 쓰지 않는 것이 이 접근자의 존재 이유다 (TextStyle과 같은 계약).
func TestPlacementAccessorsBeforeResolve(t *testing.T) {
	elem := &Element{Name: RowElement}
	if elem.Container() != nil || elem.Placement() != nil || elem.TextStyle() != nil {
		t.Fatal("해석 전 요소에 배치·외관이 있다")
	}
	elem.Appearance = &Appearance{Container: &ContainerLayout{Flow: FlowGrid}}
	if elem.Container() == nil || elem.Container().Flow != FlowGrid {
		t.Errorf("컨테이너 배치를 읽지 못했다: %+v", elem.Appearance)
	}
	if elem.Placement() != nil {
		t.Error("컨테이너 배치가 자식 배치로 읽혔다")
	}
}

// 공간 토큰 조회는 색·폰트와 같은 모양이다 (진단이 나열할 어휘가 결정적이어야 한다).
func TestThemeSpaceLookup(t *testing.T) {
	theme := Theme{Spaces: []SpaceToken{{Name: "sm", Px: 12}, {Name: "lg", Px: 36}}}
	if token, ok := theme.Space("lg"); !ok || token.Px != 36 {
		t.Errorf("Space(\"lg\") = %+v, %v", token, ok)
	}
	if _, ok := theme.Space("huge"); ok {
		t.Error("없는 토큰이 찾아졌다")
	}
	names := theme.SpaceNames()
	if len(names) != 2 || names[0] != "sm" || names[1] != "lg" {
		t.Errorf("SpaceNames = %v, want 선언 순서 그대로", names)
	}
}

// Span.Slice는 값 하나가 여러 조각을 담는 속성의 진단 정밀도다 (`pad="sm md"`, `style`).
func TestSpanSlice(t *testing.T) {
	doc, diags := Parse("slides/001_layout"+Ext, "<slide>\n  <row pad=\"sm md\"><p>내용</p></row>\n</slide>\n")
	if len(diags) != 0 {
		t.Fatalf("파서 진단이 없어야 한다: %v", diags)
	}
	row, ok := doc.Content[0].(*Element)
	if !ok {
		t.Fatalf("첫 콘텐츠가 요소가 아니다: %T", doc.Content[0])
	}
	pad, ok := row.Attr("pad")
	if !ok {
		t.Fatal("pad 속성이 없다")
	}
	// "sm md"의 둘째 조각 — 값 시작(2:13)에서 세 칸 뒤다.
	second := pad.ValueSpan.Slice(pad.Value, 3, 5)
	if second.Start.Line != 2 || second.Start.Col != 16 {
		t.Errorf("둘째 조각 위치 = %s, want 2:16", second.Start)
	}
	if got := second.Text("<slide>\n  <row pad=\"sm md\"><p>내용</p></row>\n</slide>\n"); got != "md" {
		t.Errorf("둘째 조각 = %q, want %q", got, "md")
	}

	// 줄바꿈을 담은 값도 옳게 센다 — 진단이 행을 잃으면 에디터가 엉뚱한 줄을 가리킨다.
	multi := SpanOf(Pos{Offset: 10, Line: 2, Col: 3}, Pos{Offset: 20, Line: 3, Col: 5})
	if got := multi.Slice("a: 1;\nb: 2", 6, 10); got.Start.Line != 3 || got.Start.Col != 1 {
		t.Errorf("줄바꿈 뒤 조각 위치 = %s, want 3:1", got.Start)
	}
	// 범위를 벗어난 요청은 원래 구간이다 — 진단이 위치를 잃는 것보다 덜 정밀한 위치가 낫다.
	if got := multi.Slice("a: 1", 0, 99); got != multi {
		t.Errorf("범위 밖 Slice = %+v, want 원래 구간", got)
	}
}
