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
		`{"lang": "korean"}`,
		`{"lang": "ko-"}`,
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

// lang은 BCP 47 언어 태그다. 이름("korean")은 `<html lang>`에 그대로 실리면 언어별 줄바꿈
// 규칙(`:lang(ko)`)을 조용히 끄므로 설정을 읽는 자리에서 멈춘다.
func TestLoadConfigRejectsNonBCP47Lang(t *testing.T) {
	_, err := LoadConfig(writeConfig(t, `{"lang": "korean"}`))
	if err == nil {
		t.Fatal(`"korean"을 언어 태그로 받아들였다`)
	}
	if !strings.Contains(err.Error(), "lang") || !strings.Contains(err.Error(), `"korean"`) ||
		!strings.Contains(err.Error(), `"ko"`) {
		t.Errorf("에러가 키·값·고칠 예를 말해야 한다: %v", err)
	}
}

// 올바른 태그는 정규형으로 바뀐다 — 대소문자·밑줄·세 글자 코드가 달라도 `:lang(ko)`가 맞는다.
func TestLoadConfigCanonicalizesLang(t *testing.T) {
	for body, want := range map[string]string{
		`{"lang": "ko"}`:         "ko",
		`{"lang": "ko-KR"}`:      "ko-KR",
		`{"lang": "EN-us"}`:      "en-US",
		`{"lang": "ko_KR"}`:      "ko-KR",
		`{"lang": "kor"}`:        "ko",
		`{"lang": "zh-Hant-TW"}`: "zh-Hant-TW",
	} {
		cfg, err := LoadConfig(writeConfig(t, body))
		if err != nil {
			t.Errorf("%s: %v", body, err)
			continue
		}
		if cfg.Lang != want {
			t.Errorf("%s: Lang = %q, want %q", body, cfg.Lang, want)
		}
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
