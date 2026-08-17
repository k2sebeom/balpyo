package soml

import (
	"testing"
	"testing/fstest"
)

func TestCollectOrdersByFilename(t *testing.T) {
	fsys := fstest.MapFS{
		"slides/010_architecture.slide":     {},
		"slides/001_title.slide":            {},
		"slides/900_end.slide":              {},
		"slides/002_agenda.slide":           {},
		"slides/02_deep-dive/010_api.slide": {},
		"slides/02_deep-dive/020_db.slide":  {},
	}
	files, err := Collect(fsys, "slides")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"001_title.slide",
		"002_agenda.slide",
		"010_architecture.slide",
		// 챕터는 자기 번호 자리에 들어간다 — `02_`는 `010_`보다 뒤, `900_`보다 앞.
		"02_deep-dive/010_api.slide",
		"02_deep-dive/020_db.slide",
		"900_end.slide",
	}
	assertPaths(t, files, want)
}

func TestCollectSkipsUnderscoreAndOtherExtensions(t *testing.T) {
	fsys := fstest.MapFS{
		"slides/001_title.slide":       {},
		"slides/_draft.slide":          {}, // 보류 슬라이드
		"slides/notes.md":              {}, // SOML 아님
		"slides/_wip/010_sketch.slide": {}, // 보류 챕터는 통째로 무시
		"slides/03_ok/010_real.slide":  {},
	}
	files, err := Collect(fsys, "slides")
	if err != nil {
		t.Fatal(err)
	}
	assertPaths(t, files, []string{"001_title.slide", "03_ok/010_real.slide"})
}

func TestCollectChapter(t *testing.T) {
	fsys := fstest.MapFS{
		"slides/001_title.slide":             {},
		"slides/02_deep/03_more/010_x.slide": {},
	}
	files, err := Collect(fsys, "slides")
	if err != nil {
		t.Fatal(err)
	}
	if got := files[0].Chapter; got != "" {
		t.Errorf("루트 직속 슬라이드의 Chapter = %q, want 빈 문자열", got)
	}
	if got, want := files[1].Chapter, "02_deep/03_more"; got != want {
		t.Errorf("Chapter = %q, want %q", got, want)
	}
}

func TestCollectEmptyDirIsNotAnError(t *testing.T) {
	// 슬라이드가 없는 것은 수집기의 에러가 아니다 — 빌드 파이프라인이 판단한다.
	fsys := fstest.MapFS{"slides/.keep": {}}
	files, err := Collect(fsys, "slides")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Errorf("got %d개, want 0개", len(files))
	}
}

func TestCollectMissingDir(t *testing.T) {
	if _, err := Collect(fstest.MapFS{}, "slides"); err == nil {
		t.Error("없는 디렉토리는 에러여야 한다")
	}
}

func TestCollectFileInsteadOfDir(t *testing.T) {
	fsys := fstest.MapFS{"slides": {Data: []byte("x")}}
	if _, err := Collect(fsys, "slides"); err == nil {
		t.Error("파일을 슬라이드 디렉토리로 준 경우는 에러여야 한다")
	}
}

func TestDerivedID(t *testing.T) {
	tests := []struct{ name, want string }{
		{"010_architecture.slide", "architecture"},
		{"001_title.slide", "title"},
		{"900-end.slide", "end"},
		{"02 intro.slide", "intro"},
		{"architecture.slide", "architecture"},
		{"Deep Dive.slide", "deep-dive"},
		{"a__b.slide", "a-b"},
		{"001.slide", "001"}, // 숫자만 있으면 접두를 떼지 않는다
		{"아키텍처.slide", "아키텍처"},
	}
	for _, tc := range tests {
		if got := (SlideFile{Name: tc.name}).DerivedID(); got != tc.want {
			t.Errorf("%q → %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestAspectStage(t *testing.T) {
	if s, err := Aspect16x9.Stage(); err != nil || s.Width != 1280 || s.Height != 720 {
		t.Errorf("16:9 = %+v, %v", s, err)
	}
	if s, err := Aspect4x3.Stage(); err != nil || s.Width != 960 || s.Height != 720 {
		t.Errorf("4:3 = %+v, %v", s, err)
	}
	if _, err := Aspect("21:9").Stage(); err == nil {
		t.Error("v1은 16:9와 4:3만 받는다")
	}
}

func assertPaths(t *testing.T, files []SlideFile, want []string) {
	t.Helper()
	if len(files) != len(want) {
		t.Fatalf("%d개 수집, want %d개: %v", len(files), len(want), paths(files))
	}
	for i := range want {
		if files[i].Path != want[i] {
			t.Fatalf("순서 불일치\ngot:  %v\nwant: %v", paths(files), want)
		}
	}
}

func paths(files []SlideFile) []string {
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = f.Path
	}
	return out
}
