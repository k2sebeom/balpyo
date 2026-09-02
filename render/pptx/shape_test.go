package pptx

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// resolvedDeck는 SOML 원문을 파이프라인과 같은 순서(파싱 → 외관 해석)로 덱에 담는다.
//
// 손으로 만든 덱만으로 테스트하면 파서·해석기가 실제로 채우는 모양과 조립기가 기대하는
// 모양이 갈라져도 알 수 없다 (render/web/fixture_test.go의 같은 판단). 테마는 core가
// 심는 최소 테마다 — testDeck의 testTheme와 다른 것은 의도다 (이쪽은 실제 해석 경로를 본다).
func resolvedDeck(t *testing.T, sources ...string) *soml.Deck {
	t.Helper()
	stage, err := soml.Aspect16x9.Stage()
	if err != nil {
		t.Fatal(err)
	}
	deck := &soml.Deck{Title: "text deck", Aspect: soml.Aspect16x9, Stage: stage}
	for i, src := range sources {
		file := fmt.Sprintf("slides/%03d.slide", i+1)
		doc, diags := soml.Parse(file, src)
		if err := diags.Err(); err != nil {
			t.Fatalf("%s 파싱 실패: %v", file, err)
		}
		deck.Slides = append(deck.Slides, &soml.Slide{
			Index: i, ID: fmt.Sprintf("s%d", i+1), Source: file,
			Notes: doc.Notes, Content: doc.Content, Tokens: doc.Tokens,
		})
	}
	if err := core.Resolve(deck).Err(); err != nil {
		t.Fatalf("외관 해석 실패: %v", err)
	}
	return deck
}

// measuredGeometry는 덱의 요소 순서에 rect를 하나씩 붙인 지오메트리다.
//
// geom.Synthetic은 모든 요소에 같은 rect를 주므로 좌표가 XML의 어느 도형으로 갔는지
// 구별할 수 없다 — xfrm 값을 단정하는 테스트는 요소마다 다른 rect가 필요하다.
func measuredGeometry(t *testing.T, deck *soml.Deck, rects ...geom.Rect) *geom.Geometry {
	t.Helper()
	refs := geom.Refs(deck)
	if len(refs) != len(rects) {
		t.Fatalf("rect %d개, 덱 요소 %d개", len(rects), len(refs))
	}
	boxes := make([]geom.Box, len(refs))
	for i, ref := range refs {
		boxes[i] = geom.Box{ID: ref.ID, El: ref.El, Slide: ref.Slide, Rect: rects[i]}
	}
	return &geom.Geometry{Stage: deck.Stage, Boxes: boxes}
}

// slideXML은 덱을 렌더해 첫 슬라이드 파트를 돌려준다.
func slideXML(t *testing.T, deck *soml.Deck, g *geom.Geometry) string {
	t.Helper()
	parts, _ := renderPackage(t, NewWithGeometry(g), deck)
	return parts[slidePart(1)]
}

// renderWebString은 같은 덱의 웹 산출물이다 — 두 렌더러의 결정을 대조하는 테스트용.
func renderWebString(t *testing.T, deck *soml.Deck) string {
	t.Helper()
	files, err := web.New().Render(deck)
	if err != nil {
		t.Fatal(err)
	}
	return string(files[web.IndexFile])
}

// `<text>` 하나가 p:sp 하나가 되고, 잰 rect가 그대로 EMU 좌표로 나온다. 문서 순서가
// spTree 순서다 — 겹침 z-순서의 근거.
func TestRenderTextShapes(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <text role="title">제목</text>
  <text>본문</text>
</slide>`)
	g := measuredGeometry(t, deck,
		geom.Rect{X: 64, Y: 64, W: 1152, H: 67.2},
		geom.Rect{X: 64, Y: 155.2, W: 1152, H: 36},
	)
	xml := slideXML(t, deck, g)

	if got := strings.Count(xml, "<p:sp>"); got != 2 {
		t.Fatalf("p:sp %d개, want 2\n%s", got, xml)
	}
	for _, want := range []string{
		// 결정적인 도형 id·이름 (id=1은 spTree 그룹).
		`<p:cNvPr id="2" name="text s0.0"/>`,
		`<p:cNvPr id="3" name="text s0.1"/>`,
		// 텍스트 박스 표시.
		`<p:cNvSpPr txBox="1"/>`,
		// 잰 rect + 안전 여유 → EMU (64px = 609600, 67.2px = 640080; 폭은 start 정렬이라
		// 오른쪽으로만 4% 넓어진다: 1152px × 1.04 = 1198.08px = 11411712).
		`<a:off x="609600" y="609600"/>`,
		`<a:ext cx="11411712" cy="640080"/>`,
		`<a:off x="609600" y="1478280"/>`,
		`<a:ext cx="11411712" cy="342900"/>`,
		// 사각형 프리셋 + 채움/선 없음.
		`<a:prstGeom prst="rect">`,
		"<a:noFill/>",
		// 박스 고정 + 오버플로 허용 (PRD §13-1), 인셋은 웹 박스 모델과 같은 0.
		`<a:bodyPr wrap="square" lIns="0" tIns="0" rIns="0" bIns="0" anchor="t">`,
		"<a:noAutofit/>",
	} {
		if !strings.Contains(xml, want) {
			t.Errorf("slide1.xml에 %q가 없다", want)
		}
	}
	// 문서 순서 = spTree 순서: 제목 도형이 본문 도형보다 앞이다.
	if strings.Index(xml, `name="text s0.0"`) > strings.Index(xml, `name="text s0.1"`) {
		t.Error("spTree 순서가 문서 순서가 아니다")
	}
}

// 해석된 외관이 rPr/pPr로 컴파일된다 (PRD §9.4) — Attrs를 다시 읽지 않으므로 역할
// 기본값과 인라인 속성의 조합이 웹과 같은 값으로 나와야 한다.
func TestRenderTextAppearance(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <text role="title" align="center">제목</text>
  <text role="code" color="#abcdef">코드</text>
</slide>`)
	g := measuredGeometry(t, deck,
		geom.Rect{X: 64, Y: 64, W: 1152, H: 67.2},
		geom.Rect{X: 64, Y: 155.2, W: 1152, H: 30},
	)
	xml := slideXML(t, deck, g)

	for _, want := range []string{
		// title: 56px → 42pt → sz=4200, weight 700 → b=1, 줄높이 1.2 → 1.2×56px×0.75 =
		// 50.4pt = spcPts 5040(절대 줄간격 — units.go LineSpacing의 근거),
		// 가운데 정렬, fg 토큰 → schemeClr tx1, heading 폰트 → major 참조.
		`<a:rPr sz="4200" b="1">`,
		`<a:pPr algn="ctr">`,
		`<a:spcPts val="5040"/>`,
		`<a:schemeClr val="tx1"/>`,
		`<a:latin typeface="+mj-lt"/>`,
		`<a:ea typeface="+mj-ea"/>`,
		// code: 20px → 15pt, weight 400 → b=0, 리터럴 색은 srgbClr(대문자 정규화),
		// mono는 fontScheme 자리가 없어 리터럴 typeface다.
		`<a:rPr sz="1500" b="0">`,
		`<a:srgbClr val="ABCDEF"/>`,
		`<a:latin typeface="Noto Sans Mono"/>`,
	} {
		if !strings.Contains(xml, want) {
			t.Errorf("slide1.xml에 %q가 없다", want)
		}
	}
}

// 토큰 색의 RGB는 슬라이드 파트에 나오면 안 된다 — 리터럴로 눕는 순간 그 글자만
// PowerPoint의 테마 색 편집을 따라가지 못한다 (웹의 CSS 변수 보존과 짝을 이루는 절반).
func TestRenderTextKeepsTokenColorsAsSchemeClr(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <text color="accent1">강조</text>
</slide>`)
	g := measuredGeometry(t, deck, geom.Rect{X: 64, Y: 64, W: 1152, H: 36})
	xml := slideXML(t, deck, g)

	if !strings.Contains(xml, `<a:schemeClr val="accent1"/>`) {
		t.Error("토큰 색이 schemeClr 참조로 나오지 않았다")
	}
	for _, token := range deck.Theme.Colors {
		if strings.Contains(xml, token.RGB) {
			t.Errorf("토큰 %s의 RGB(%s)가 슬라이드 파트에 있다 — 테마 파트에만 있어야 한다",
				token.Name, token.RGB)
		}
	}
}

// algn은 논리 이름의 OOXML 대응이다 — l/r은 rtl에서 뜻이 뒤집히는 논리 값이라
// start/end와 짝이 맞는다 (soml.TextAlign의 주석).
func TestAlgnValue(t *testing.T) {
	tests := []struct {
		align soml.TextAlign
		want  string
	}{
		{soml.AlignStart, "l"},
		{soml.AlignCenter, "ctr"},
		{soml.AlignEnd, "r"},
		{soml.AlignJustify, "just"},
	}
	for _, tc := range tests {
		got, err := algnValue(tc.align)
		if err != nil {
			t.Fatalf("algnValue(%q): %v", tc.align, err)
		}
		if got != tc.want {
			t.Errorf("algnValue(%q) = %q, want %q", tc.align, got, tc.want)
		}
	}
	// 해석되지 않은 빈 값이 algn=""으로 흘러가면 뷰어가 조용히 무시한다.
	if _, err := algnValue(""); err == nil {
		t.Error("빈 정렬은 에러여야 한다")
	}
}

// OOXML은 굵기 단계가 없어 임계값으로 눕힌다 — 경계는 CSS 폰트 매칭이 bold 면을 고르는
// 600이다 (웹이 굵게 그리는 지점과 같아야 한다).
func TestBoldValue(t *testing.T) {
	tests := []struct {
		weight int
		want   string
	}{
		{400, "0"}, {599, "0"}, {600, "1"}, {700, "1"},
	}
	for _, tc := range tests {
		if got := boldValue(tc.weight); got != tc.want {
			t.Errorf("boldValue(%d) = %q, want %q", tc.weight, got, tc.want)
		}
	}
}

// 문자 데이터의 &는 유효한 XML로 이스케이프되어야 한다 — 픽스처 003의 "질의응답 (Q and A)"
// 같은 실제 원문이 &를 쓰는 순간 파일이 열리지 않는 것을 막는다.
func TestRenderTextEscapesXML(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <text>R&D 로드맵 > 상반기</text>
</slide>`)
	g := measuredGeometry(t, deck, geom.Rect{X: 64, Y: 64, W: 1152, H: 36})
	xml := slideXML(t, deck, g)

	if want := "<a:t>R&amp;D 로드맵 &gt; 상반기</a:t>"; !strings.Contains(xml, want) {
		t.Errorf("slide1.xml에 %q가 없다\n%s", want, xml)
	}
}

// 원문 개행은 파서가 공백으로 접는다 — 웹과 PPTX가 **같은 문자열**을 받아야 줄 수가
// 갈라지지 않는다 (픽스처 002의 본문이 이 경우다). 한쪽이 a:br을 내고 다른 쪽이 접으면
// 두 타깃의 줄바꿈이 달라진다.
func TestRenderTextNewlinesAgreeWithWebRenderer(t *testing.T) {
	deck := resolvedDeck(t, "<slide>\n  <text>모놀리스 분리를 마쳤고,\n  관측성 스택을 도입했다.</text>\n</slide>")
	g := measuredGeometry(t, deck, geom.Rect{X: 64, Y: 64, W: 1152, H: 72})

	const collapsed = "모놀리스 분리를 마쳤고, 관측성 스택을 도입했다."
	xml := slideXML(t, deck, g)
	if !strings.Contains(xml, "<a:t>"+collapsed+"</a:t>") {
		t.Errorf("PPTX가 접힌 문자열을 내지 않았다\n%s", xml)
	}
	if strings.Contains(xml, "<a:br") {
		t.Error("PPTX가 a:br을 냈다 — 파서가 접은 개행이 되살아났다")
	}

	// 웹 렌더러도 같은 결정을 해야 한다. 두 렌더러가 같은 덱 모델을 받으므로 이 대조가
	// "결정은 파서 한 곳"이라는 계약을 못 박는다.
	html := renderWebString(t, deck)
	if !strings.Contains(html, ">"+collapsed+"</div>") {
		t.Errorf("웹이 접힌 문자열을 내지 않았다\n%s", html)
	}
	if strings.Contains(html, "<br") {
		t.Error("웹이 <br>을 냈다 — 두 렌더러의 개행 결정이 갈라졌다")
	}
}

// 빈 텍스트·크기 0 박스도 도형을 낸다 — 건너뛰면 spTree 순서와 도형 id가 내용에 따라
// 흔들리고, 저자는 빈 요소의 존재를 PowerPoint에서 확인할 수 없다 (writeTextShape의 정책).
func TestRenderEmptyTextStillEmitsShape(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <text></text>
</slide>`)
	g := measuredGeometry(t, deck, geom.Rect{X: 64, Y: 64, W: 0, H: 0})
	xml := slideXML(t, deck, g)

	if got := strings.Count(xml, "<p:sp>"); got != 1 {
		t.Fatalf("p:sp %d개, want 1\n%s", got, xml)
	}
	if !strings.Contains(xml, `<a:ext cx="0" cy="0"/>`) {
		t.Error("크기 0 박스가 그대로 나오지 않았다")
	}
	// 빈 텍스트는 런 없는 문단이다.
	if strings.Contains(xml, "<a:r>") {
		t.Error("빈 텍스트에 런이 있으면 안 된다")
	}
}

// 어휘에 태그를 추가하고 렌더 매핑을 잊으면 조용히 사라지는 대신 멈춘다 (웹과 같은 계약).
func TestRenderUnmappedElementFails(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <text>본문</text>
</slide>`)
	deck.Slides[0].Content = []soml.Node{&soml.Element{Name: "box"}}
	g := measuredGeometry(t, deck, geom.Rect{X: 0, Y: 0, W: 10, H: 10})

	_, err := NewWithGeometry(g).Render(deck)
	if err == nil {
		t.Fatal("매핑 없는 태그가 렌더되었다")
	}
	if !strings.Contains(err.Error(), "매핑이 없다") {
		t.Errorf("에러가 매핑 부재를 가리키지 않는다: %v", err)
	}
}

// 외관이 해석되지 않은 요소는 기본값으로 그리지 않는다 — 웹만 그럴듯하게 나오고 PPTX와
// 갈라지는 대신 멈춘다 (render/web.writeText와 같은 계약).
func TestRenderUnresolvedAppearanceFails(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <text>본문</text>
</slide>`)
	deck.Slides[0].Content[0].(*soml.Element).Appearance = nil
	g := measuredGeometry(t, deck, geom.Rect{X: 0, Y: 0, W: 10, H: 10})

	_, err := NewWithGeometry(g).Render(deck)
	if err == nil {
		t.Fatal("해석되지 않은 외관이 렌더되었다")
	}
	if !strings.Contains(err.Error(), "core.Resolve") {
		t.Errorf("에러가 해석 누락을 가리키지 않는다: %v", err)
	}
}

// 콘텐츠가 있는 덱도 같은 입력이 같은 바이트를 내야 한다 — 도형 조립에 맵 순회가 새면
// 여기서 잡힌다.
func TestRenderContentDeckIsByteIdentical(t *testing.T) {
	render := func() []byte {
		deck := resolvedDeck(t, `<slide>
  <text role="title" align="center">제목</text>
  <text color="accent1">본문</text>
</slide>`)
		g := measuredGeometry(t, deck,
			geom.Rect{X: 64, Y: 64, W: 1152, H: 67.2},
			geom.Rect{X: 64, Y: 155.2, W: 1152, H: 36},
		)
		files, err := NewWithGeometry(g).Render(deck)
		if err != nil {
			t.Fatal(err)
		}
		return files[FileName(deck)]
	}
	if !bytes.Equal(render(), render()) {
		t.Error("같은 덱이 다른 바이트를 냈다")
	}
}

// 안전 여유는 글자가 닻을 내린 변을 움직이지 않아야 한다 — start는 왼쪽 변, end는 오른쪽 변,
// center는 중심이 그대로여야 웹과 같은 자리에 그려진다 (TextBoxRect의 계약).
func TestTextBoxRect(t *testing.T) {
	measured := geom.Rect{X: 100, Y: 50, W: 1000, H: 40}
	tests := []struct {
		align soml.TextAlign
		want  geom.Rect
	}{
		{soml.AlignStart, geom.Rect{X: 100, Y: 50, W: 1040, H: 40}},
		{soml.AlignCenter, geom.Rect{X: 80, Y: 50, W: 1040, H: 40}},
		{soml.AlignEnd, geom.Rect{X: 60, Y: 50, W: 1040, H: 40}},
		// justify는 글줄을 박스 폭까지 늘이므로 넓히지 않는다.
		{soml.AlignJustify, measured},
	}
	for _, tc := range tests {
		got := TextBoxRect(measured, tc.align)
		if got != tc.want {
			t.Errorf("TextBoxRect(%s, %s) = %s, want %s", measured, tc.align, got, tc.want)
		}
		// 높이와 세로 위치는 어느 정렬에서도 그대로다 — 줄바꿈 여유는 가로 문제다.
		if got.Y != measured.Y || got.H != measured.H {
			t.Errorf("TextBoxRect(%s)가 세로를 건드렸다: %s", tc.align, got)
		}
	}

	// 크기 0 박스(빈 요소)는 여유도 0이다 — 위치가 흔들리면 안 된다.
	zero := geom.Rect{X: 64, Y: 64}
	if got := TextBoxRect(zero, soml.AlignCenter); got != zero {
		t.Errorf("크기 0 박스가 움직였다: %s", got)
	}

	// 음수 좌표에서도 안전 여유는 **부호와 무관**해야 한다 (항목 05 — 박스 밖으로 걸친
	// 오버레이 배지는 음수 x를 갖는다). 여유(slack)는 폭 W(양수)에서 나오고 정렬 방향으로만
	// x를 움직이므로, X의 부호가 계산에 끼면 안 된다: start는 오른쪽으로만, end는 왼쪽으로만,
	// center는 양쪽 반씩 — 양수 좌표와 같은 규칙이다.
	neg := geom.Rect{X: -50, Y: -30, W: 1000, H: 40}
	for _, tc := range []struct {
		align soml.TextAlign
		want  geom.Rect
	}{
		{soml.AlignStart, geom.Rect{X: -50, Y: -30, W: 1040, H: 40}},
		{soml.AlignCenter, geom.Rect{X: -70, Y: -30, W: 1040, H: 40}},
		{soml.AlignEnd, geom.Rect{X: -90, Y: -30, W: 1040, H: 40}},
		{soml.AlignJustify, neg},
	} {
		if got := TextBoxRect(neg, tc.align); got != tc.want {
			t.Errorf("음수 좌표 TextBoxRect(%s, %s) = %s, want %s", neg, tc.align, got, tc.want)
		}
	}
}
