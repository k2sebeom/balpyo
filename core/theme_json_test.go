package core

import (
	"strings"
	"testing"
)

// 테마 JSON 파일의 틀린 자리는 그 파일의 줄:칸과 JSON 안의 경로로 말한다 — 이름을 요청한
// `balpyo.json`을 가리키면 저자가 항목을 하나씩 지워 가며 찾게 된다. 배열 원소는 이름이 있으면
// 이름으로 부른다.
func TestThemeJSONErrorsNameTheFileAndPath(t *testing.T) {
	cases := []struct {
		name, tokens, comps string
		file, pos, want     string
	}{
		{"components.json의 모르는 키", "",
			"{\n  \"text\": {\n    \"variants\": [\n      {\"name\": \"stat\", \"lineHeight\": 1.1}\n    ]\n  }\n}",
			"themes/corp/components.json", "4:24", `unknown key "lineHeight" at text.variants["stat"].lineHeight`},
		{"tokens.json의 모르는 키", "{\n  \"colour\": []\n}", "",
			"themes/corp/tokens.json", "2:3", `unknown key "colour" at colour`},
		{"이름 없는 원소는 순번", "", `{"text": {"componentDefaults": [{"tag": "h1"}, {"tag": "h2", "font": "body"}]}}`,
			"themes/corp/components.json", "1:62", `unknown key "font" at text.componentDefaults[1].font`},
		{"값의 타입", "", "{\n  \"shape\": {\"variants\": [{\"name\": \"card\", \"fill\": 3}]}\n}",
			"themes/corp/components.json", "2:51", `shape.variants["card"].fill must be a string, not number`},
		{"문법 오류", "", "{\n  \"text\": {,}\n}",
			"themes/corp/components.json", "2:12", "invalid JSON"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			project := localTheme("corp", `{"name":"corp","extends":"minimal"}`, tc.tokens, tc.comps, nil)
			_, diags := themeDeckIn(t, project, "corp", themeProbe)
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1: %v", len(diags), diags)
			}
			d := diags[0]
			if d.File != tc.file || d.Span.Start.String() != tc.pos {
				t.Errorf("자리 = %s:%s, want %s:%s", d.File, d.Span.Start, tc.file, tc.pos)
			}
			if !strings.Contains(d.Message, tc.want) {
				t.Errorf("진단이 %q를 말하지 않는다: %s", tc.want, d.Message)
			}
		})
	}
}

// `$comment`로 시작하는 키는 두 파일 모두, 어느 객체에서나 문서 주석이다 — tokens.json의
// `$comment-accent`처럼 주석을 값 옆에 여럿 둘 수 있어야 하고, 그 관례가 components.json에서만
// 에러이면 같은 테마 안에서 규칙이 둘이 된다.
func TestThemeJSONCommentKeysAreAllowedEverywhere(t *testing.T) {
	tokens := `{"$comment": "a", "$comment-color": "b", "color": [{"$comment": "c", "name": "accent1", "rgb": "123456", "slot": "accent1", "ref": "accent1"}]}`
	comps := `{"$comment-defaults": "a", "text": {"$comment-variants": "b",
	  "variants": [{"$comment": "c", "name": "lead", "size": 30}]}}`
	project := localTheme("corp", `{"name":"corp","extends":"minimal"}`, tokens, comps, nil)
	if _, diags := themeDeckIn(t, project, "corp", themeProbe); len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
}

// 메모장이 저장한 테마 파일은 UTF-8 BOM으로 시작한다. 에디터에 보이지 않는 그 바이트 때문에 테마가
// "invalid character"로 깨지면 안 되고, 진단의 칸도 에디터가 보여 주는 대로 BOM을 세지 않는다.
func TestThemeJSONIgnoresLeadingBOM(t *testing.T) {
	const bom = "\uFEFF"
	project := localTheme("corp", bom+`{"name":"corp","extends":"minimal"}`,
		bom+`{"$comment": "a"}`, bom+`{"$comment": "b"}`, nil)
	if _, diags := themeDeckIn(t, project, "corp", themeProbe); len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	if h, err := ReadThemeHeader(project, "themes/corp"); err != nil || h.Name != "corp" {
		t.Fatalf("ReadThemeHeader = %+v, %v; want corp", h, err)
	}
	// 레이아웃 CSS는 `<style>`에 그대로 인라인되므로 BOM이 남으면 첫 선택자의 일부가 된다.
	project = localTheme("corp", `{"name":"corp","extends":"minimal"}`, "", "",
		map[string]string{"title.css": bom + "display: grid;"})
	if got := mustLoadLocal(t, project, "corp").layoutStyles()["title"]; strings.Contains(got, bom) {
		t.Errorf("title 배치에 BOM이 남았다: %q", got)
	}

	project = localTheme("corp", `{"name":"corp","extends":"minimal"}`, bom+`{"colour": []}`, "", nil)
	_, diags := themeDeckIn(t, project, "corp", themeProbe)
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1: %v", len(diags), diags)
	}
	if got := diags[0].Span.Start.String(); got != "1:2" {
		t.Errorf("자리 = %s, want 1:2", got)
	}
}
