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

func TestLoadConfigRejectsUnimplementedPPTXMode(t *testing.T) {
	// 구현되지 않은 모드를 받아 주면 저자는 네이티브와 같은 바이트를 받고도 모른다.
	dir := writeConfig(t, `{"targets": {"pptx": {"mode": "image"}}}`)
	_, err := LoadConfig(dir)
	if err == nil {
		t.Fatal(`mode "image"는 구현되지 않았으므로 에러여야 한다`)
	}
	want := `targets.pptx.mode: unknown mode "image" ("native" is the only mode)`
	if !strings.Contains(err.Error(), want) {
		t.Errorf("에러 = %q, want %q 포함", err, want)
	}
}

func TestLoadConfigSlidesDirStaysInsideProject(t *testing.T) {
	for _, tc := range []struct{ body, want string }{
		{`{"slidesDir": "deck/slides"}`, "deck/slides"},
		{`{"slidesDir": "./talk/"}`, "talk"},
	} {
		cfg, err := LoadConfig(writeConfig(t, tc.body))
		if err != nil {
			t.Errorf("%s: %v", tc.body, err)
			continue
		}
		if cfg.SlidesDir != tc.want {
			t.Errorf("%s: SlidesDir = %q, want %q", tc.body, cfg.SlidesDir, tc.want)
		}
	}
	for _, dir := range []string{"../t1/slides", "slides/../../x", "/abs/slides"} {
		_, err := LoadConfig(writeConfig(t, `{"slidesDir": "`+dir+`"}`))
		if err == nil {
			t.Errorf("slidesDir %q는 프로젝트 밖이므로 에러여야 한다", dir)
			continue
		}
		want := `slidesDir: "` + dir + `" must be a relative path inside the project`
		if !strings.Contains(err.Error(), want) {
			t.Errorf("에러 = %q, want %q 포함", err, want)
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

// 설정 파일의 에러는 고칠 자리(줄:칸)와 JSON의 말로 나온다 — Go 구조체·타입 이름이 아니다.
func TestLoadConfigErrorsPointIntoTheFile(t *testing.T) {
	tests := []struct {
		body string
		want string
	}{
		{"{\n  \"title\": 1\n}", "balpyo.json:2:12: title must be a string, not number"},
		{"{\n  \"tilte\": \"x\"\n}", `balpyo.json:2:3: unknown key "tilte" at tilte`},
		// BOM은 에디터에 보이지 않으므로 칸을 세지 않는다.
		{"\uFEFF{\"tilte\": \"x\"}", `balpyo.json:1:2: unknown key "tilte" at tilte`},
		{`{"targets": {"pptx": {"mdoe": "image"}}}`, `balpyo.json:1:23: unknown key "mdoe" at targets.pptx.mdoe`},
		// 테마 파일의 `$comment` 주석 규칙은 balpyo.json에 없다 — 이 파일은 닫혀 있다.
		{`{"$comment": "x"}`, `balpyo.json:1:2: unknown key "$comment"`},
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
