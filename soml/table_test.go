package soml

import (
	"strings"
	"testing"
)

// 표의 정상 트리 — 태그(table/tr/th/td)·병합 속성·외관 속성·셀 안 인라인 서식이 파싱을
// 그대로 통과한다. dumpNodes의 표현은 subset_test.go와 공유한다.
func TestParseTable(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "헤더 행 + 본문 행",
			src:  `<slide><table><tr><th>지역</th><th>Q1</th></tr><tr><td>남미</td><td>18%</td></tr></table></slide>`,
			want: `table(tr(th("지역") th("Q1")) tr(td("남미") td("18%")))`,
		},
		{
			name: "병합 속성 — colspan/rowspan",
			src:  `<slide><table><tr><td colspan="2" rowspan="2">병합</td></tr></table></slide>`,
			want: `table(tr(td[colspan=2 rowspan=2]("병합")))`,
		},
		{
			name: "셀 외관 속성",
			src:  `<slide><table><tr><th fill="accent1" border="1 solid muted" color="bg" align="center" anchor="middle">머리</th></tr></table></slide>`,
			want: `table(tr(th[fill=accent1 border=1 solid muted color=bg align=center anchor=middle]("머리")))`,
		},
		{
			name: "셀 안 인라인 서식 — 텍스트 전용 콘텐츠 모델",
			src:  `<slide><table><tr><td>매출 <strong>+18%</strong></td></tr></table></slide>`,
			want: `table(tr(td("매출 " strong("+18%"))))`,
		},
		{
			name: "표 프레임의 variant와 자식 배치 — 레이아웃 컨테이너의 자식",
			src:  `<slide><col><table variant="striped" grow="1"><tr><td>x</td></tr></table></col></slide>`,
			want: `col(table[variant=striped grow=1](tr(td("x"))))`,
		},
		{
			name: "표는 슬롯을 채울 수 있다",
			src:  `<slide layout="content"><table slot="body"><tr><td>x</td></tr></table></slide>`,
			want: `table[slot=body](tr(td("x")))`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, diags := Parse("slides/001_table.slide", tc.src)
			if len(diags) != 0 {
				t.Fatalf("진단이 없어야 한다:\n%s", diags.Err())
			}
			if got := dumpNodes(doc.Content); got != tc.want {
				t.Errorf("트리 = %q\nwant %q", got, tc.want)
			}
		})
	}
}

// 표 요소의 종류 — 네 층이 KindOf로 갈라지는 값이다 (subset.go의 카탈로그).
func TestTableKinds(t *testing.T) {
	cases := map[string]ElementKind{
		TableElement:           KindTable,
		TableRowElement:        KindTableRow,
		TableHeaderCellElement: KindTableCell,
		TableDataCellElement:   KindTableCell,
	}
	for name, want := range cases {
		if got := KindOf(name); got != want {
			t.Errorf("KindOf(%q) = %v, want %v", name, got, want)
		}
	}
	if !IsHeaderCell(TableHeaderCellElement) || IsHeaderCell(TableDataCellElement) {
		t.Error("IsHeaderCell은 <th>에만 참이어야 한다")
	}
	if !IsTableCell(TableHeaderCellElement) || !IsTableCell(TableDataCellElement) || IsTableCell(TableRowElement) {
		t.Error("IsTableCell은 <th>/<td>에만 참이어야 한다")
	}
}

// 셀 속성 목록과 순서 — 진단 메시지가 이 순서를 그대로 싣는다 (ShapeAttrs의 회귀 시험과
// 같은 태도). 병합 → 외관 순, 자식 배치·슬롯은 붙지 않는다.
func TestTableCellAttrsOrder(t *testing.T) {
	want := "colspan, rowspan, fill, border, color, align, anchor"
	if got := strings.Join(TableCellAttrs(), ", "); got != want {
		t.Errorf("TableCellAttrs() = %q\nwant %q", got, want)
	}
	// 셀은 slot·grow 같은 배치 채널도, 프레임이 소유하는 variant도 받지 않는다.
	for _, banned := range []string{SlotAttr, "grow", "span", StyleAttr, "variant"} {
		for _, a := range TableCellAttrs() {
			if a == banned {
				t.Errorf("셀에 %q가 열려선 안 된다", banned)
			}
		}
	}
}

// 표 프레임은 슬롯·자식 배치 대상이다 — withChildPlacement가 slot을 잇는다.
func TestTableFrameAttrsIncludeSlot(t *testing.T) {
	spec := elementSpecs[TableElement]
	var hasSlot, hasVariant bool
	for _, a := range spec.attrs {
		switch a {
		case SlotAttr:
			hasSlot = true
		case "variant":
			hasVariant = true
		}
	}
	if !hasSlot {
		t.Error("<table>는 slot 속성을 받아야 한다 (슬롯을 채울 수 있다)")
	}
	if !hasVariant {
		t.Error("<table>는 variant 속성을 받아야 한다 (테마 표 스타일)")
	}
}

// 배치 위반 — 진단의 line:col과 문구까지 단정한다 (<li> 규칙의 선례와 같은 태도).
func TestParseTablePlacementErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "tr은 table 밖에 올 수 없다",
			src:  `<slide><tr><td>x</td></tr></slide>`,
			want: []string{"1:9 <tr>는 <table>의 자식으로만 쓸 수 있다"},
		},
		{
			name: "td는 tr 밖에 올 수 없다",
			src:  `<slide><table><td>x</td></table></slide>`,
			want: []string{"1:16 <th>/<td>는 <tr>의 자식으로만 쓸 수 있다"},
		},
		{
			name: "table의 자식은 tr뿐이다",
			src:  `<slide><table><p>본문</p></table></slide>`,
			want: []string{"1:16 <table>의 자식으로 <p>는 올 수 없다 (쓸 수 있는 태그: <tr>)"},
		},
		{
			name: "셀 안에는 블록이 올 수 없다 (텍스트 전용)",
			src:  `<slide><table><tr><td><p>본문</p></td></tr></table></slide>`,
			want: []string{"1:24 <td> 안에 블록 태그 <p>는 올 수 없다"},
		},
		// 문자 데이터의 조언은 **부모가** 정한다 — `<text>로 감싼다`는 표 안에서 따를 수 없는
		// 말이다 (`<table>`의 자식은 행뿐이고 `<tr>`의 자식은 셀뿐이다).
		{
			name: "table 안의 문자 데이터는 행으로 감싸라고 말한다",
			src:  `<slide><table>지역</table></slide>`,
			want: []string{"1:15 <table> 안에 문자 데이터를 직접 쓸 수 없다 (<tr>로 감싼다)"},
		},
		{
			name: "tr 안의 문자 데이터는 셀로 감싸라고 말한다",
			src:  `<slide><table><tr>지역</tr></table></slide>`,
			want: []string{"1:19 <tr> 안에 문자 데이터를 직접 쓸 수 없다 (<td>로 감싼다)"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := Parse("slides/001_table.slide", tc.src)
			if len(diags) != len(tc.want) {
				t.Fatalf("진단 %d개, want %d개:\n%s", len(diags), len(tc.want), diags.Err())
			}
			for i, want := range tc.want {
				pos, message, _ := strings.Cut(want, " ")
				message = strings.TrimPrefix(want, pos+" ")
				if got := diags[i].Span.Start.String(); got != pos {
					t.Errorf("진단[%d] 위치 = %s, want %s (%s)", i, got, pos, diags[i].Message)
				}
				if !strings.Contains(diags[i].Message, message) {
					t.Errorf("진단[%d] 메시지 = %q, want %q 포함", i, diags[i].Message, message)
				}
			}
		})
	}
}
