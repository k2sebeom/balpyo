package soml

import (
	"strings"
	"testing"
)

// 파일 전체를 가리키는 진단(빈 Span)은 위치 없이 포맷된다.
func TestDiagnosticString(t *testing.T) {
	span := SpanOf(Pos{Offset: 10, Line: 3, Col: 5}, Pos{Offset: 15, Line: 3, Col: 10})
	for _, tc := range []struct {
		diag Diagnostic
		want string
	}{
		{Errorf("slides/001_title.slide", span, "알 수 없는 태그 %q", "sldie"), `slides/001_title.slide:3:5: error: 알 수 없는 태그 "sldie"`},
		{Warningf("balpyo.json", Span{}, "테마를 찾지 못해 기본 테마로 폴백했다"), "balpyo.json: warning: 테마를 찾지 못해 기본 테마로 폴백했다"},
	} {
		if got := tc.diag.String(); got != tc.want {
			t.Errorf("got  %q\nwant %q", got, tc.want)
		}
	}
}

func TestDiagnosticsErr(t *testing.T) {
	var ds Diagnostics
	if ds.HasErrors() || ds.Err() != nil {
		t.Error("빈 목록은 에러가 없어야 한다")
	}

	ds = Diagnostics{
		Warningf("a.slide", Span{}, "경고 하나"),
		Errorf("b.slide", Span{}, "에러 하나"),
	}
	if !ds.HasErrors() {
		t.Error("에러가 있어야 한다")
	}
	// Err은 에러만 모은다 — 경고는 종료 코드에 영향을 주지 않는다.
	if got, want := ds.Err().Error(), "b.slide: error: 에러 하나"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDiagnosticsStrictPromotesWarnings(t *testing.T) {
	ds := Diagnostics{Warningf("a.slide", Span{}, "미정의 variant")}
	if ds.Strict().Err() == nil {
		t.Error("--strict는 경고를 에러로 승격해야 한다")
	}
	if ds.HasErrors() {
		t.Error("원본 목록은 변하지 않아야 한다")
	}
}

// 층 순서로 쌓인 진단(뒤 파일의 파서 진단이 앞 파일의 값 층 진단보다 먼저)이 파일 → 줄 → 열로
// 펴진다. 파일 순서는 슬라이드 수집 순서다 — 장 디렉토리는 같은 깊이의 파일 뒤가 아니라
// 이름 순서에 선다. 같은 파일에서 위치 없는 진단이 앞이고, 같은 자리는 원래 순서를 지킨다.
func TestDiagnosticsSorted(t *testing.T) {
	at := func(line, col int) Span { return Span{Start: Pos{Line: line, Col: col}} }
	in := Diagnostics{
		Errorf("slides/002_b.slide", at(5, 1), "b5"),
		Errorf("slides/002_b.slide", at(3, 9), "b3:9"),
		Errorf("slides/002_b.slide", at(3, 2), "b3:2-first"),
		Errorf("slides/002_b.slide", at(3, 2), "b3:2-second"),
		Warningf("slides/001_a.slide", at(2, 4), "a2"),
		Errorf("slides/001_a.slide", Span{}, "a-whole"),
		Errorf("slides/001_ch/001_x.slide", at(1, 1), "ch"),
	}
	var got []string
	for _, d := range in.Sorted() {
		got = append(got, d.Message)
	}
	want := []string{"a-whole", "a2", "ch", "b3:2-first", "b3:2-second", "b3:9", "b5"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("순서\n got: %v\nwant: %v", got, want)
	}
	if in[0].Message != "b5" {
		t.Error("Sorted가 원래 목록을 바꿨다")
	}
}
