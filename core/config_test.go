package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

func TestLoadConfigDefaultsWhenMissing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "my-talk")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("balpyo.json이 없어도 빌드는 가능해야 한다: %v", err)
	}
	if cfg.Title != "my-talk" {
		t.Errorf("Title = %q, want 디렉토리 이름", cfg.Title)
	}
	if cfg.Theme != "minimal" || cfg.Aspect != soml.Aspect16x9 ||
		cfg.SlidesDir != "slides" || cfg.OutDir != "dist" ||
		cfg.Targets.PPTX.Mode != PPTXNative {
		t.Errorf("기본값 불일치: %+v", cfg)
	}
}

func TestLoadConfigOverridesOnlyGivenFields(t *testing.T) {
	dir := writeConfig(t, `{"title": "2026 아키텍처 리뷰", "aspect": "4:3"}`)
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Title != "2026 아키텍처 리뷰" || cfg.Aspect != soml.Aspect4x3 {
		t.Errorf("지정한 필드가 반영되지 않았다: %+v", cfg)
	}
	// 지정하지 않은 필드는 기본값을 유지한다.
	if cfg.OutDir != "dist" || cfg.Targets.PPTX.Mode != PPTXNative {
		t.Errorf("기본값이 지워졌다: %+v", cfg)
	}
}

func TestLoadConfigRejectsUnknownField(t *testing.T) {
	dir := writeConfig(t, `{"tilte": "오타"}`)
	_, err := LoadConfig(dir)
	if err == nil {
		t.Fatal("알 수 없는 필드는 에러여야 한다")
	}
	if !strings.Contains(err.Error(), "tilte") {
		t.Errorf("에러가 무엇이 잘못됐는지 말해야 한다: %v", err)
	}
}

func TestLoadConfigRejectsClosedVocabularyViolations(t *testing.T) {
	for _, body := range []string{
		`{"aspect": "21:9"}`,
		`{"targets": {"pptx": {"mode": "vector"}}}`,
		`{`,
	} {
		dir := writeConfig(t, body)
		if _, err := LoadConfig(dir); err == nil {
			t.Errorf("%s는 에러여야 한다", body)
		}
	}
}

func TestValidateFillsBlanks(t *testing.T) {
	cfg := Config{Title: "t"}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if cfg.Aspect != soml.DefaultAspect || cfg.Lang != "en" || cfg.Theme != "minimal" {
		t.Errorf("빈 필드가 채워지지 않았다: %+v", cfg)
	}
}

// 설정 파일의 에러는 고칠 자리(줄:칸)와 JSON의 말로 나온다 — Go 구조체·타입 이름이 아니다.
func TestLoadConfigErrorsPointIntoTheFile(t *testing.T) {
	tests := []struct {
		body string
		want string
	}{
		{"{\n  \"title\": 1\n}", "balpyo.json:2:12: title must be a string, not number"},
		{`{"pageNumber": "yes"}`, "balpyo.json:1:16: pageNumber must be true or false, not string"},
		{`{"targets": {"pptx": {"mode": 3}}}`, "targets.pptx.mode must be a string, not number"},
		{`{"title": "x",}`, "balpyo.json:1:15: invalid JSON"},
		{`{`, "balpyo.json: invalid JSON: unexpected end of file"},
		{`{"title":"x"} {"a":1}`, "balpyo.json:1:15: unexpected content after the top-level object"},
		{"{\"title\":\"x\"}\n]", "balpyo.json:2:1: unexpected content after the top-level object"},
	}
	for _, tc := range tests {
		dir := writeConfig(t, tc.body)
		_, err := LoadConfig(dir)
		if err == nil {
			t.Errorf("%s는 에러여야 한다", tc.body)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: 에러 = %q, want %q 포함", tc.body, err, tc.want)
		}
		for _, leak := range []string{"Go struct", "core.", "cannot unmarshal"} {
			if strings.Contains(err.Error(), leak) {
				t.Errorf("%s: 에러에 내부 표기 %q가 새어 나왔다: %v", tc.body, leak, err)
			}
		}
	}
	// 뒤에 공백만 있는 파일은 정상이다.
	if _, err := LoadConfig(writeConfig(t, "{\"title\": \"x\"}\n\n")); err != nil {
		t.Errorf("뒤의 공백은 에러가 아니다: %v", err)
	}
}

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ConfigFile), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}
