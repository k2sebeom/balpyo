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

// 메모장은 UTF-8을 BOM과 함께 저장한다. 에디터에 보이지 않는 그 세 바이트 때문에 멀쩡해 보이는
// 설정이 "invalid character"로 거부되면 저자는 원인을 찾을 수 없다.
func TestLoadConfigIgnoresLeadingBOM(t *testing.T) {
	dir := writeConfig(t, "\uFEFF{\"title\": \"BOM\"}")
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("BOM으로 시작하는 balpyo.json도 읽혀야 한다: %v", err)
	}
	if cfg.Title != "BOM" {
		t.Errorf("Title = %q, want BOM", cfg.Title)
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
