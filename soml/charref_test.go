package soml

import (
	"strings"
	"testing"
)

// textOf는 파싱 결과의 첫 콘텐츠 요소의 텍스트다.
func textOf(t *testing.T, src string) string {
	t.Helper()
	doc := parseOK(t, src)
	return elementAt(t, doc.Content, 0).TextContent()
}

// 이름 있는 참조 5종과 수치 참조가 풀린다 — 모델에는 **해석된 문자**가 담긴다 (ADR 0037).
func TestParseResolvesCharRefs(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"이름 참조 5종", `&lt;slide&gt; &amp; &quot;인용&quot; &apos;홑&apos;`, `<slide> & "인용" '홑'`},
		{"10진 수치 참조", `&#60;p&#62; &#8594;`, "<p> →"},
		{"16진 수치 참조", `&#x3C;p&#x3E; &#x27E8;범위&#x27E9;`, "<p> ⟨범위⟩"},
		{"대문자 X도 16진이다", `&#X3c;`, "<"},
		{"참조는 두 번 풀리지 않는다", `&amp;lt;`, "&lt;"},
		{"참조 모양이 아닌 &는 리터럴이다", `R&D 예산 ?a=1&b=2 100% & 그 외`, "R&D 예산 ?a=1&b=2 100% & 그 외"},
		{"참조로 쓴 공백은 접히지 않는다", `가&#32;&#32;나`, "가  나"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := textOf(t, "<slide><p>"+tc.body+"</p></slide>"); got != tc.want {
				t.Errorf("텍스트 = %q, want %q", got, tc.want)
			}
		})
	}
}

// 참조 모양인데 어휘 밖이면 에러다 — 지원하지 않는 이름을 리터럴로 흘리면 화면에
// `&nbsp;`가 보이는 것으로만 드러난다.
func TestParseRejectsUnknownCharRefs(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string // 진단 메시지에 들어 있어야 하는 조각
		col  int    // 진단이 가리키는 칸 (1-based)
	}{
		{"알 수 없는 이름", `공백&nbsp;하나`, "&nbsp;", 13},
		{"수치가 비었다", `&#;`, "&#;", 11},
		{"XML이 금지한 문자", `&#0;`, "&#0;", 11},
		{"서로게이트", `&#xD800;`, "&#xD800;", 11},
		{"범위를 넘는 코드포인트", `&#x110000;`, "&#x110000;", 11},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := Parse("slides/001.slide", "<slide><p>"+tc.body+"</p></slide>")
			if len(diags) != 1 {
				t.Fatalf("진단 %d건, want 1건:\n%s", len(diags), diags.Err())
			}
			if !strings.Contains(diags[0].Message, tc.want) {
				t.Errorf("메시지 = %q, want %q 포함", diags[0].Message, tc.want)
			}
			if diags[0].Span.Start.Col != tc.col {
				t.Errorf("진단 위치 = %v, want 1:%d (참조를 가리켜야 한다)", diags[0].Span.Start, tc.col)
			}
		})
	}
}

// 속성 값도 같은 규칙으로 푼다 — `alt="a &lt; b"`가 동작하고 `?a=1&b=2`는 그대로다.
func TestParseResolvesCharRefsInAttrValues(t *testing.T) {
	doc := parseOK(t, `<slide><image src="a.png" alt="a &lt; b &amp; c"/><p><a href="https://x/?a=1&b=2">링크</a></p></slide>`)
	img := elementAt(t, doc.Content, 0)
	alt, _ := img.Attr("alt")
	if want := "a < b & c"; alt.Value != want {
		t.Errorf("alt = %q, want %q", alt.Value, want)
	}
	link := elementAt(t, elementAt(t, doc.Content, 1).Children, 0)
	href, _ := link.Attr("href")
	if want := "https://x/?a=1&b=2"; href.Value != want {
		t.Errorf("href = %q, want %q (참조 모양이 아닌 &는 리터럴이다)", href.Value, want)
	}
}

func TestParseRejectsUnknownCharRefInAttrValue(t *testing.T) {
	_, diags := Parse("slides/001.slide", `<slide><image src="a.png" alt="&nbsp;"/></slide>`)
	if len(diags) != 1 {
		t.Fatalf("진단 %d건, want 1건:\n%s", len(diags), diags.Err())
	}
	if !strings.Contains(diags[0].Message, "&nbsp;") {
		t.Errorf("메시지 = %q", diags[0].Message)
	}
	// 값 안의 참조를 가리킨다 — 속성 이름이 아니라 값이다.
	if got := diags[0].Span.Start.Col; got != 32 {
		t.Errorf("진단 위치 = 1:%d, want 1:32", got)
	}
}

// 마크업이 아닌 `<`는 내용이다 — 비교 연산자와 수식이 그냥 쓰인다 (ADR 0037).
func TestParseKeepsNonMarkupAngleBracket(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"비교 연산자", "a < b", "a < b"},
		{"공백 없는 비교", "1<2", "1<2"},
		{"작거나 같다", "x <= y", "x <= y"},
		{"닫기만 있는 화살표", "a <- b", "a <- b"},
		{"문장 끝의 <", "끝은 <", "끝은 <"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := textOf(t, "<slide><p>"+tc.body+"</p></slide>"); got != tc.want {
				t.Errorf("텍스트 = %q, want %q", got, tc.want)
			}
		})
	}
}

// 글자가 뒤따르는 `<`는 여전히 태그다 — 오타를 리터럴로 흘리지 않는다.
func TestParseStillRejectsUnknownTagAfterAngleBracket(t *testing.T) {
	_, diags := Parse("slides/001.slide", "<slide><p>a <txt>b</txt></p></slide>")
	if len(diags) == 0 {
		t.Fatal("알 수 없는 태그 <txt>가 진단되지 않았다")
	}
	if !strings.Contains(diags[0].Message, "<txt>") {
		t.Errorf("첫 진단 = %q, want <txt> 언급", diags[0].Message)
	}
}

// CDATA는 원문을 그대로 담는다 — 마크업을 설명하는 덱이 자기 문법을 띄우는 자리다.
func TestParseCData(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "pre 안의 CDATA는 원문과 줄바꿈을 보존한다",
			src:  "<slide><pre><![CDATA[<slide id=\"x\">\n  <text>제목</text>\n</slide>]]></pre></slide>",
			want: "<slide id=\"x\">\n  <text>제목</text>\n</slide>",
		},
		{
			name: "CDATA 안에서는 참조를 풀지 않는다",
			src:  `<slide><pre><![CDATA[&lt;&amp;]]></pre></slide>`,
			want: `&lt;&amp;`,
		},
		{
			name: "인라인 code 안의 CDATA",
			src:  `<slide><p><code><![CDATA[a < b && c]]></code></p></slide>`,
			want: "a < b && c",
		},
		{
			name: "접는 모델에서는 CDATA의 공백도 접힌다",
			src:  "<slide><p><![CDATA[a   b\n  c]]></p></slide>",
			want: "a b c",
		},
		{
			name: "CDATA와 텍스트가 이어진다",
			src:  `<slide><p>식: <![CDATA[a < b]]> 끝</p></slide>`,
			want: "식: a < b 끝",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := textOf(t, tc.src); got != tc.want {
				t.Errorf("텍스트 = %q\nwant       %q", got, tc.want)
			}
		})
	}
}

func TestParseRejectsUnterminatedCData(t *testing.T) {
	_, diags := Parse("slides/001.slide", "<slide><pre><![CDATA[<slide>\n</pre></slide>")
	// 닫히지 않은 CDATA는 파일의 나머지를 삼키므로 열린 태그 진단이 함께 난다 (닫히지 않은
	// 주석과 같다). CDATA를 말하는 진단이 그 안에 있어야 한다.
	var found bool
	for _, d := range diags {
		if strings.Contains(d.Message, "CDATA") {
			found = true
		}
	}
	if !found {
		t.Errorf("닫히지 않은 CDATA가 진단되지 않았다:\n%s", diags.Err())
	}
}

// 요소만 담는 모델에서 CDATA는 문자 데이터와 같은 진단을 받는다 — 표기가 달라도 내용이다.
func TestParseRejectsCDataWhereCharDataIsNotAllowed(t *testing.T) {
	_, diags := Parse("slides/001.slide", `<slide><![CDATA[본문]]></slide>`)
	if len(diags) != 1 {
		t.Fatalf("진단 %d건, want 1건:\n%s", len(diags), diags.Err())
	}
	if !strings.Contains(diags[0].Message, "문자 데이터") {
		t.Errorf("진단 = %q", diags[0].Message)
	}
}

// scanCharRef의 경계를 단위로 고정한다 — 참조 모양 판정이 `;`에 걸려 있다는 결정의 핵이다.
func TestScanCharRef(t *testing.T) {
	tests := []struct {
		in     string
		r      rune
		length int
		ok     bool
	}{
		{"&lt;", '<', 4, true},
		{"&amp;x", '&', 5, true},
		{"&#39;", '\'', 5, true},
		{"&#x27E8;", '⟨', 8, true},
		{"&nbsp;", 0, 6, false}, // 모양은 맞고 어휘 밖이다 — 진단 대상.
		{"&#xZZ;", 0, 0, false}, // 16진이 아니다 — 모양 아님.
		{"&D 예산", 0, 0, false},  // `;`가 없다 — 리터럴 `&`.
		{"&", 0, 0, false},      // 끝의 `&`.
		{"&;", 0, 0, false},     // 빈 이름.
		{"& lt;", 0, 0, false},  // 공백이 끼면 참조가 아니다.
		{"&#x110000;", 0, 10, false},
	}
	for _, tc := range tests {
		r, length, ok := scanCharRef(tc.in)
		if length != tc.length || ok != tc.ok || (tc.ok && r != tc.r) {
			t.Errorf("scanCharRef(%q) = (%q, %d, %v), want (%q, %d, %v)",
				tc.in, r, length, ok, tc.r, tc.length, tc.ok)
		}
	}
}
