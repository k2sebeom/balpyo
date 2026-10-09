package soml

import "testing"

// textOf는 파싱 결과의 첫 콘텐츠 요소의 텍스트다.
func textOf(t *testing.T, src string) string {
	t.Helper()
	doc := parseOK(t, src)
	return elementAt(t, doc.Content, 0).TextContent()
}

// 이름 있는 참조 5종과 수치 참조가 풀리고 참조·마크업 모양이 아닌 `&`·`<`는 리터럴이다 — 모델에는 **해석된 문자**가 담긴다.
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
		// 마크업이 아닌 `<`는 내용이다 — 비교 연산자와 수식이 그냥 쓰인다.
		{"마크업이 아닌 <", "a < b 1<2 x <= y a <- b 끝은 <", "a < b 1<2 x <= y a <- b 끝은 <"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := textOf(t, "<slide><p>"+tc.body+"</p></slide>"); got != tc.want {
				t.Errorf("텍스트 = %q, want %q", got, tc.want)
			}
		})
	}
}

// 참조 모양인데 어휘 밖이면 에러다 — 리터럴로 흘리면 화면에 `&nbsp;`가 보이는 것으로만
// 드러난다. 진단은 참조(속성이면 값 안의 참조)를 가리킨다.
func TestParseCharRefAndCDataErrors(t *testing.T) {
	runDiagCases(t, []diagCase{
		{name: "알 수 없는 이름", src: `<slide><p>공백&nbsp;하나</p></slide>`, want: []string{"1:13 쓸 수 없는 문자 참조 &nbsp;"}},
		{name: "수치가 비었다", src: `<slide><p>&#;</p></slide>`, want: []string{"1:11 &#;"}},
		{name: "XML이 금지한 문자", src: `<slide><p>&#0;</p></slide>`, want: []string{"1:11 &#0;"}},
		{name: "서로게이트", src: `<slide><p>&#xD800;</p></slide>`, want: []string{"1:11 &#xD800;"}},
		{name: "범위를 넘는 코드포인트", src: `<slide><p>&#x110000;</p></slide>`, want: []string{"1:11 &#x110000;"}},
		{name: "속성 값 안의 참조", src: `<slide><image src="a.png" alt="&nbsp;"/></slide>`, want: []string{"1:32 쓸 수 없는 문자 참조 &nbsp;"}},
		// 글자가 뒤따르는 `<`는 여전히 태그다 — 오타를 리터럴로 흘리지 않는다.
		{name: "글자가 뒤따르는 <는 태그다", src: `<slide><p>a <txt>b</txt></p></slide>`, want: []string{"1:14 알 수 없는 태그 <txt>"}},
		{
			// 닫히지 않은 CDATA는 파일의 나머지를 삼키므로 열린 태그 진단이 함께 난다.
			name: "닫히지 않은 CDATA",
			src:  "<slide><pre><![CDATA[<slide>\n</pre></slide>",
			want: []string{
				"1:2 <slide> 태그가 닫히지 않았다",
				"1:9 <pre> 태그가 닫히지 않았다",
				"1:13 CDATA 섹션이 `]]>`로 닫히지 않았다",
			},
		},
		// 요소만 담는 모델에서 CDATA는 문자 데이터와 같은 진단을 받는다.
		{name: "슬라이드 직속 CDATA", src: `<slide><![CDATA[본문]]></slide>`, want: []string{"1:8 <slide> 안에 문자 데이터를 직접 쓸 수 없다 (<text>로 감싼다)"}},
	})
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
