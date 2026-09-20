package core

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 코드 하이라이팅의 값 층 계약이다 (P2.4 · ADR 0051): `<pre>`의 문자 데이터가
// **합성 인라인 런**이 되고, 색은 테마의 `code.tokens` 표에서 오며, 탭은 칼럼을 세어 펴진다.
//
// 렌더러를 손대지 않는 것이 이 설계의 값이므로 여기서 못 박는 모양이 곧 두 산출물의 입력이다.

// codeRuns는 `<pre>`의 자식을 (종류, 글자) 목록으로 눕힌다 — 색 있는 조각은 합성 `<span>`,
// plain은 문자 데이터다. 색은 토큰 이름으로 돌려준다 (리터럴 RGB로 눕히면 schemeClr 참조가
// 끊긴 것을 이 테스트가 못 본다).
func codeRuns(t *testing.T, pre *soml.Element) []string {
	t.Helper()
	var runs []string
	for _, child := range pre.Children {
		switch node := child.(type) {
		case *soml.Text:
			runs = append(runs, "plain "+node.Value)
		case *soml.Element:
			if node.Name != "span" {
				t.Fatalf("합성 자식이 <span>이 아니다: <%s>", node.Name)
			}
			style := node.TextStyle()
			if style == nil {
				t.Fatalf("합성 런의 외관이 해석되지 않았다 (%s)", node.Span)
			}
			if style.Color.Token == "" {
				t.Fatalf("합성 런의 색이 토큰 참조가 아니다: %#v", style.Color)
			}
			runs = append(runs, style.Color.Token+" "+node.TextContent())
		default:
			t.Fatalf("알 수 없는 노드 %T", child)
		}
	}
	return runs
}

// preOf는 해석된 덱의 첫 `<pre>`다.
func preOf(t *testing.T, src string) *soml.Element {
	t.Helper()
	deck, diags := resolveSlide(t, src)
	if err := diags.Err(); err != nil {
		t.Fatalf("진단이 없어야 한다: %v", err)
	}
	return findElement(t, deck, "pre")
}

// 토큰 하나하나가 런이 되지 않는다 — **인접한 같은 종류는 합친다**. Chroma는 공백마다
// 토큰을 끊으므로(아래 Go 다섯 줄이 30토큰이 넘는다) 합치지 않으면 `a:r`가 글자 수만큼 는다.
func TestHighlightCoalescesAdjacentClasses(t *testing.T) {
	pre := preOf(t, `<slide><pre lang="go">func warm(ids []string) error {
  n := 42 // 예열
  return nil
}</pre></slide>`)
	runs := codeRuns(t, pre)

	// 첫 세 런: 키워드 · plain 공백 · 함수 이름. 그 뒤 괄호는 punct(색 없음)이므로 plain과
	// 합쳐지지 않고 **따로** 남는다 — punct를 표에 적지 않은 것이 minimal의 선택이다.
	want := []string{"accent6 func", "plain ", "accent1 warm"}
	for i, w := range want {
		if i >= len(runs) || runs[i] != w {
			t.Fatalf("런[%d] = %q, want %q\n전체: %q", i, runAt(runs, i), w, runs)
		}
	}
	for _, w := range []string{"accent4 42", "muted // 예열", "accent6 return"} {
		if !hasRun(runs, w) {
			t.Errorf("런 %q가 없다\n전체: %q", w, runs)
		}
	}
	// 글자는 하나도 잃지 않는다 — 토큰 스트림의 왕복이 원문과 같아야 한다.
	if got := pre.TextContent(); !strings.Contains(got, "func warm(ids []string) error {") {
		t.Errorf("원문이 보존되지 않았다:\n%s", got)
	}
	// 런 수는 글자 수가 아니라 종류 경계의 수다 — 다섯 줄이 스무 런 아래여야 한다.
	if len(runs) > 20 {
		t.Errorf("런 %d개 — 인접 합치기가 없다\n%q", len(runs), runs)
	}
}

// 종류 → 색은 **테마**가 정한다 (components.json의 code.tokens). minimal의 표 전체를 여기서
// 못 박는다 — 번호는 위계가 아니므로(ADR 0050) 이 대응은 눈으로 고른 것이고, 바꾸려면
// 테스트를 함께 읽어야 한다.
func TestHighlightUsesThemeTokenTable(t *testing.T) {
	pre := preOf(t, `<slide><pre lang="xml"><![CDATA[<slide layout="content">
  <!-- 주석 -->
</slide>]]></pre></slide>`)
	runs := codeRuns(t, pre)
	for _, want := range []string{
		"accent1 <slide",      // NameTag → name
		"accent2 layout=",     // NameAttribute → attr
		`accent3 "content"`,   // LiteralString → string
		"muted <!-- 주석 -->", // Comment → comment (인접 합치기가 셋을 하나로 만든다)
	} {
		if !hasRun(runs, want) {
			t.Errorf("런 %q가 없다\n전체: %q", want, runs)
		}
	}
}

// plain은 **요소가 되지 않는다** — 문자 데이터로 남는다. 그래야 `<pre>`의 확정 색이 바닥에
// 남고(저자가 쓴 `color=`가 무시되지 않는다), 웹의 선두 개행 보호(firstText)도 그대로 산다.
func TestHighlightLeavesPlainAsCharacterData(t *testing.T) {
	pre := preOf(t, `<slide><pre lang="bash" color="muted">balpyo build --target html,pptx</pre></slide>`)
	if len(pre.Children) != 1 {
		t.Fatalf("자식 %d개, want 1개 (전부 plain이면 요소가 하나도 생기지 않는다): %q",
			len(pre.Children), codeRuns(t, pre))
	}
	if _, ok := pre.Children[0].(*soml.Text); !ok {
		t.Fatalf("plain 조각이 요소가 됐다: %T", pre.Children[0])
	}
	// 선두 개행도 plain이다 — 첫 자식이 요소가 되면 웹의 선두 개행 보호가 눈이 먼다.
	lead := preOf(t, `<slide><pre lang="go">

func main() {}</pre></slide>`)
	if text, ok := lead.Children[0].(*soml.Text); !ok || !strings.HasPrefix(text.Value, "\n") {
		t.Errorf("선두 개행이 문자 데이터로 남지 않았다: %#v", lead.Children[0])
	}
}

// 탭은 **칼럼을 세어** 펴진다 (D15). gofmt 출력을 그대로 붙여 넣을 수 있어야 하고, 폭은
// 테마가 정한다 (minimal은 2). 정지점은 "다음 배수까지"지 "N칸"이 아니다 — 글자 하나 뒤의
// 탭은 한 칸만 먹는다.
func TestHighlightExpandsTabsByColumn(t *testing.T) {
	pre := preOf(t, "<slide><pre>a\tb\n\t\tc\n</pre></slide>")
	want := "a b\n    c"
	if got := pre.TextContent(); got != want {
		t.Errorf("탭이 칼럼으로 펴지지 않았다: %q, want %q", got, want)
	}
}

// lang이 없는 `<pre>`도 탭을 편다 — 탭은 하이라이팅의 문제가 아니라 글자의 문제다.
// (그리고 하이라이팅이 없으므로 자식은 문자 데이터 하나로 남는다.)
func TestHighlightExpandsTabsWithoutLang(t *testing.T) {
	pre := preOf(t, "<slide><pre>func main() {\n\tprintln(1)\n}</pre></slide>")
	if got := pre.TextContent(); !strings.Contains(got, "\n  println(1)") {
		t.Errorf("lang 없는 pre의 탭이 펴지지 않았다: %q", got)
	}
	if len(pre.Children) != 1 {
		t.Errorf("자식 %d개, want 1개 — lang이 없으면 런을 만들지 않는다", len(pre.Children))
	}
}

// 탭은 하이라이팅 **전에** 펴진다 — 토크나이저가 보는 글자가 산출물의 글자와 같아야
// 왕복(토큰 값을 이어 붙인 것 = 원문)이 성립한다.
func TestHighlightExpandsTabsBeforeTokenising(t *testing.T) {
	pre := preOf(t, "<slide><pre lang=\"go\">func f() {\n\treturn nil\n}</pre></slide>")
	if got := pre.TextContent(); !strings.Contains(got, "\n  return nil") {
		t.Errorf("탭이 펴진 뒤 토큰화되지 않았다: %q", got)
	}
}

// lang 어휘는 **엔진**이 닫는다 (ADR 0051) — 목록 밖은 에러이고 진단이 쓸 수 있는 이름을
// 나열한다. variant처럼 폴백하지 않는 이유: variant 어휘는 테마의 사유물이라 다른 테마의
// 덱도 열려야 하지만, lang은 엔진 어휘이므로 목록 밖 이름은 오타이거나 미지원이다.
func TestHighlightLangVocabulary(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "목록 밖 언어",
			src:  `<slide><pre lang="typescript">let x = 1</pre></slide>`,
			want: `알 수 없는 lang "typescript" (쓸 수 있는 값: go, xml, soml, json, bash)`,
		},
		{
			name: "값 없는 lang",
			src:  `<slide><pre lang>x</pre></slide>`,
			want: "lang 속성에는 값이 필요하다",
		},
		{
			name: "빈 lang",
			src:  `<slide><pre lang="">x</pre></slide>`,
			want: "lang이 비어 있다",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := resolveSlide(t, tc.src)
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
			}
			if !strings.Contains(diags[0].Message, tc.want) {
				t.Errorf("진단 = %q, want %q 포함", diags[0].Message, tc.want)
			}
			if diags[0].Severity != soml.SeverityError {
				t.Errorf("심각도 = %v, want 에러", diags[0].Severity)
			}
		})
	}
}

// soml은 xml의 별칭이다 — 이 저장소의 덱이 자기 문법을 띄우는 것이 가장 흔한 코드 블록이고,
// `lang="soml"`이 저자가 먼저 쓸 이름이다.
func TestHighlightSomlIsAliasOfXML(t *testing.T) {
	pre := preOf(t, `<slide><pre lang="soml"><![CDATA[<text role="title">제목</text>]]></pre></slide>`)
	runs := codeRuns(t, pre)
	for _, want := range []string{"accent1 <text", "accent2 role=", `accent3 "title"`} {
		if !hasRun(runs, want) {
			t.Errorf("런 %q가 없다\n전체: %q", want, runs)
		}
	}
}

// 테마의 `code` 절은 검사를 받는다 — 테마는 저자가 고칠 수 없는 자리이므로 덱을 렌더하기
// 전에 멈춘다 (크롬 선언·슬롯 대조와 같은 태도).
func TestCodeComponentsAreValidated(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
	}{
		{
			name: "모르는 종류 이름",
			code: `{"tabSize":2,"tokens":[{"class":"kewyord","color":"accent6"}]}`,
			want: "알 수 없는 코드 토큰 종류",
		},
		{
			name: "테마에 없는 색 토큰",
			code: `{"tabSize":2,"tokens":[{"class":"keyword","color":"brand"}]}`,
			want: `색 토큰 "brand"`,
		},
		{
			name: "리터럴 hex",
			code: `{"tabSize":2,"tokens":[{"class":"keyword","color":"#DC2626"}]}`,
			want: "색 토큰 이름이어야 한다",
		},
		{
			name: "탭 폭이 0",
			code: `{"tabSize":0,"tokens":[]}`,
			want: "tabSize는 1 이상이어야 한다",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := loadTheme(themeFSWithCode(tc.code), "t")
			if err == nil {
				t.Fatalf("잘못된 code 절이 진단되지 않았다 (%s)", tc.code)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("에러 = %q, want %q 포함", err.Error(), tc.want)
			}
		})
	}
}

// themeFSWithCode는 `code` 절만 다른 최소 테마 패키지다 (themeFSWithChrome의 code 판).
// 색 토큰은 하나만 둔다 — "테마에 없는 색 토큰"을 판정하려면 있는 이름과 없는 이름이
// 갈려야 한다.
func themeFSWithCode(code string) fstest.MapFS {
	return fstest.MapFS{
		"t/theme.json":  {Data: []byte(`{"name":"t","layouts":["content"]}`)},
		"t/tokens.json": {Data: []byte(`{"color":[{"name":"accent6","rgb":"9333EA","slot":"accent6","ref":"accent6"}],"font":[],"space":[],"radius":[],"shadow":[]}`)},
		"t/components.json": {Data: []byte(
			`{"text":{"roles":[]},"code":` + code + `}`)},
		"t/layouts/content.css": {Data: []byte("")},
	}
}

func runAt(runs []string, i int) string {
	if i < len(runs) {
		return runs[i]
	}
	return "<없음>"
}

func hasRun(runs []string, want string) bool {
	for _, run := range runs {
		if run == want {
			return true
		}
	}
	return false
}
