package soml

import "testing"

func TestDiagnosticString(t *testing.T) {
	span := SpanOf(Pos{Offset: 10, Line: 3, Col: 5}, Pos{Offset: 15, Line: 3, Col: 10})
	got := Errorf("slides/001_title.slide", span, "알 수 없는 태그 %q", "sldie").String()
	want := `slides/001_title.slide:3:5: error: 알 수 없는 태그 "sldie"`
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestDiagnosticStringWithoutPosition(t *testing.T) {
	// 파일 전체를 가리키는 진단은 위치 없이 포맷된다.
	got := Warningf("balpyo.json", Span{}, "테마를 찾지 못해 기본 테마로 폴백했다").String()
	want := "balpyo.json: warning: 테마를 찾지 못해 기본 테마로 폴백했다"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
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
