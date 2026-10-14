package core

import (
	"strings"
	"testing"
	"testing/fstest"
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
		{"최상위가 객체가 아니다", "", "\n  [1]",
			"themes/corp/components.json", "2:3", "the top-level value must be an object, not array"},
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

// 병합 결과를 검사하다 난 에러(값 어휘 · 완전성)도 고칠 파일과 그 안의 줄:칸을 가리킨다 — 이름을
// 요청한 `balpyo.json`이 아니다. `extends` 체인에서는 틀린 선언을 **실제로 적은** 테마의 파일이다:
// 자식이 덮어 쓴 이름은 자식의 것, 물려받은 이름은 부모의 것.
func TestThemeValueErrorsNameTheDeclaringFile(t *testing.T) {
	badVariant := "{\n  \"shape\": {\n    \"variants\": [\n      {\"name\": \"card-body\", \"fill\": \"nope2\"}\n    ]\n  }\n}"
	cases := []struct {
		name, request   string
		project         fstest.MapFS
		file, pos, want string
	}{
		{"자기 파일의 variant 색", "corp",
			localTheme("corp", `{"name":"corp","extends":"minimal"}`, "", badVariant, nil),
			"themes/corp/components.json", "4:37", `shape variant "card-body": fill: color token "nope2" is not in the theme`},
		{"부모가 적은 variant 색", "b", mergeFS(
			localTheme("b", `{"name":"b","extends":"brine"}`, "", `{"text":{"variants":[{"name":"lead","size":30}]}}`, nil),
			localTheme("brine", `{"name":"brine","extends":"minimal"}`, "", badVariant, nil)),
			"themes/brine/components.json", "4:37", `shape variant "card-body"`},
		{"자식이 덮어 쓴 variant 색", "b", mergeFS(
			localTheme("b", `{"name":"b","extends":"brine"}`, "", badVariant, nil),
			localTheme("brine", `{"name":"brine","extends":"minimal"}`, "",
				`{"shape":{"variants":[{"name":"card-body","fill":"surface"}]}}`, nil)),
			"themes/b/components.json", "4:37", `shape variant "card-body"`},
		{"역할 색", "corp", localTheme("corp", `{"name":"corp","extends":"minimal"}`, "",
			"{\"text\": {\"roles\": [\n  {\"name\": \"title\", \"font\": \"heading\", \"size\": 60, \"weight\": 700, \"lineHeight\": 1.1, \"color\": \"brand\", \"align\": \"start\"}\n]}}", nil),
			"themes/corp/components.json", "2:95", `role "title" color "brand" is not a color token of this theme`},
		{"겹친 clrScheme 슬롯", "corp", localTheme("corp", `{"name":"corp","extends":"minimal"}`,
			"{\"color\": [\n  {\"name\": \"brand\", \"rgb\": \"445566\", \"slot\": \"accent1\", \"ref\": \"accent1\"}\n]}", "", nil),
			"themes/corp/tokens.json", "2:46", `clrScheme slot "accent1" is shared`},
		{"크롬 선언", "corp", localTheme("corp",
			"{\"name\": \"corp\", \"extends\": \"minimal\",\n \"chrome\": [{\"kind\": \"footer\", \"align\": \"middle\", \"role\": \"caption\"}]}", "", "", nil),
			"themes/corp/theme.json", "2:41", `unknown align "middle"`},
		{"부모가 적은 rgb", "b", mergeFS(
			localTheme("b", `{"name":"b","extends":"brine"}`, "", "", nil),
			localTheme("brine", `{"name":"brine","extends":"minimal"}`,
				"{\"color\": [\n  {\"name\": \"accent1\", \"rgb\": \"#123456\", \"slot\": \"accent1\", \"ref\": \"accent1\"}\n]}", "", nil)),
			"themes/brine/tokens.json", "2:30", `has rgb "#123456"`},
		{"variant 굵기", "corp", localTheme("corp", `{"name":"corp","extends":"minimal"}`, "",
			"{\"text\": {\"variants\": [\n  {\"name\": \"lead\", \"weight\": 6000}\n]}}", nil),
			"themes/corp/components.json", "2:30", `text variant "lead" weight 6000 is out of range`},
		{"fontScheme 자리", "corp", localTheme("corp", `{"name":"corp","extends":"minimal"}`,
			"{\"font\": [\n  {\"name\": \"heading\", \"latin\": \"Inter\", \"scheme\": \"mid\"}\n]}", "", nil),
			"themes/corp/tokens.json", "2:51", `unknown fontScheme slot "mid"`},
		{"상속 없는 테마의 빠진 토큰", "corp", localTheme("corp", `{"name":"corp","layouts":["content"]}`,
			`{"color":[{"name":"fg","rgb":"000000","slot":"dk1","ref":"tx1"}]}`, fullComponents,
			map[string]string{"content.css": ""}),
			"themes/corp/tokens.json", "1:10", `tokens.json has no color token "bg"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := themeDeckIn(t, tc.project, tc.request, themeProbe)
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1: %v", len(diags), diags)
			}
			d := diags[0]
			if d.File != tc.file || d.Span.Start.String() != tc.pos {
				t.Errorf("자리 = %s:%s, want %s:%s (%s)", d.File, d.Span.Start, tc.file, tc.pos, d.Message)
			}
			if !strings.Contains(d.Message, tc.want) {
				t.Errorf("진단이 %q를 말하지 않는다: %s", tc.want, d.Message)
			}
		})
	}
}

// 설치한 테마의 값 에러는 홈 기준 경로라 프로젝트의 파일 자리로 찍지 않는다 — 대신 테마 이름과
// 그 테마 안의 파일 · 줄:칸을 메시지에 담는다.
func TestInstalledThemeValueErrorsNameTheThemeAndFile(t *testing.T) {
	home := localTheme("acme", `{"name":"acme","extends":"minimal"}`, "",
		"{\n  \"shape\": {\"variants\": [{\"name\": \"card\", \"fill\": \"nope2\"}]}\n}", nil)
	_, diags := themeDeckIn(t, WithInstalledThemes(fstest.MapFS{}, home), "acme", themeProbe)
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1: %v", len(diags), diags)
	}
	if want := `theme "acme": themes/acme/components.json:2:51: shape variant "card"`; !strings.Contains(diags[0].Message, want) {
		t.Errorf("진단이 %q를 말하지 않는다: %s", want, diags[0].Message)
	}
	if _, err := CheckInstalledTheme(home, "acme"); err == nil || !strings.Contains(err.Error(), "themes/acme/components.json:2:51") {
		t.Errorf("CheckInstalledTheme = %v — 파일과 줄:칸을 말해야 한다", err)
	}
}

// 테마 파일의 색은 토큰 참조만 받는다 — 모르는 토큰 진단이 `#RRGGBB`를 권하면 그 권고를 따른 값이
// 다음 에러("is not a color token reference of this theme")로 막힌다.
func TestThemeColorErrorsDoNotSuggestLiterals(t *testing.T) {
	project := localTheme("corp", `{"name":"corp","extends":"minimal"}`, "",
		`{"text":{"variants":[{"name":"kicker","color":"nope2"}]}}`, nil)
	_, diags := themeDeckIn(t, project, "corp", themeProbe)
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1: %v", len(diags), diags)
	}
	if msg := diags[0].Message; !strings.Contains(msg, "available tokens:") || strings.Contains(msg, "#RRGGBB") {
		t.Errorf("테마 파일의 색 진단은 토큰 이름만 권해야 한다: %s", msg)
	}
	// 슬라이드의 색은 리터럴을 받으므로 권고가 남는다.
	_, diags = themeDeck(t, "minimal", `<slide id="s"><text color="nope2">x</text></slide>`)
	if len(diags) != 1 || !strings.Contains(diags[0].Message, "or #RRGGBB") {
		t.Errorf("슬라이드 색 진단은 #RRGGBB를 권해야 한다: %v", diags)
	}
}

// theme.json도 닫힌 어휘다 — 크롬 선언의 `skipLayout`(s 빠짐) 같은 오타를 조용히 버리면 푸터가
// 빼려던 레이아웃에 그대로 실리고, 빌드는 아무 말도 하지 않는다. 자리는 다른 테마 파일과 같이
// 그 파일의 줄:칸과 JSON 경로로 말하고, `$comment*` 키는 여기서도 문서 주석이다.
func TestThemeManifestUnknownKeysAreErrors(t *testing.T) {
	cases := []struct {
		name, manifest, pos, want string
	}{
		{"크롬 선언의 오타",
			"{\"name\": \"corp\", \"extends\": \"minimal\",\n  \"chrome\": [{\"kind\": \"footer\", \"align\": \"start\", \"role\": \"caption\", \"skipLayout\": [\"title\"]}]}",
			"2:70", `unknown key "skipLayout" at chrome[0].skipLayout`},
		{"배경 선언의 모르는 키",
			"{\"name\": \"corp\", \"extends\": \"minimal\",\n  \"backgrounds\": [{\"fill\": \"bg\", \"bogus\": 1}]}",
			"2:34", `unknown key "bogus" at backgrounds[0].bogus`},
		{"최상위의 모르는 키",
			"{\"$comment\": \"a\", \"name\": \"corp\", \"extends\": \"minimal\",\n  \"background\": [{\"fill\": \"bg\"}]}",
			"2:3", `unknown key "background" at background`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			project := localTheme("corp", tc.manifest, "", "", nil)
			_, diags := themeDeckIn(t, project, "corp", themeProbe)
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1: %v", len(diags), diags)
			}
			d := diags[0]
			if d.File != "themes/corp/theme.json" || d.Span.Start.String() != tc.pos {
				t.Errorf("자리 = %s:%s, want themes/corp/theme.json:%s", d.File, d.Span.Start, tc.pos)
			}
			if !strings.Contains(d.Message, tc.want) {
				t.Errorf("진단이 %q를 말하지 않는다: %s", tc.want, d.Message)
			}
		})
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
