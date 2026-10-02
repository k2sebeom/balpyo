package pptx

import (
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
	if err := core.Resolve(deck, nil).Err(); err != nil {
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

// 텍스트 박스의 골격: 채움·선 없는 rect, 박스 고정 + 넘치면 줄여 맞춤(ADR 0064), 인셋 0.
// 줄높이는 절대 줄간격이다 — title 56px × 1.2 × 0.75 = 50.4pt (LineSpacing의 근거). 종류·rect·
// 런은 패리티가 본다.
func TestRenderTextBody(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <text role="title">제목</text>
</slide>`)
	xml := slideXML(t, deck, measuredGeometry(t, deck, geom.Rect{X: 64, Y: 64, W: 1152, H: 67.2}))

	for _, want := range []string{
		`<p:cNvSpPr txBox="1"/>`,
		`<a:prstGeom prst="rect">`,
		"<a:noFill/>",
		`<a:bodyPr wrap="square" lIns="0" tIns="0" rIns="0" bIns="0" anchor="t">`,
		"<a:normAutofit/>",
		`<a:spcPts val="5040"/>`,
	} {
		if !strings.Contains(xml, want) {
			t.Errorf("slide1.xml에 %q가 없다", want)
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

// 원문의 문자 참조는 해석된 뒤 `a:t`에서 **한 번** 이스케이프된다 (ADR 0037) — 두 번이면
// PowerPoint에 `&lt;`가 보이고, 안 하면 파일이 열리지 않는다.
func TestRenderTextEscapesXMLOnce(t *testing.T) {
	tests := []struct{ src, want string }{
		{`R&D 로드맵 > 상반기`, "<a:t>R&amp;D 로드맵 &gt; 상반기</a:t>"},
		{`&lt;slide id=&quot;x&quot;&gt; &amp; a &lt; b`, `<a:t>&lt;slide id=&quot;x&quot;&gt; &amp; a &lt; b</a:t>`},
	}
	for _, tc := range tests {
		deck := resolvedDeck(t, "<slide>\n  <text>"+tc.src+"</text>\n</slide>")
		xml := slideXML(t, deck, measuredGeometry(t, deck, geom.Rect{X: 64, Y: 64, W: 1152, H: 36}))
		if !strings.Contains(xml, tc.want) {
			t.Errorf("slide1.xml에 %q가 없다\n%s", tc.want, xml)
		}
	}
}

// 렌더러는 해석·검증을 지나지 않은 모델을 기본값으로 그리지 않고 멈춘다 — 웹만 그럴듯하게
// 나오고 PPTX와 갈라지는 대신 (웹 렌더러와 같은 계약). 파서가 막는 모양도 직접 만든다.
func TestRenderRejectsUnresolvedModel(t *testing.T) {
	tests := []struct {
		name, src string
		breakIt   func(content []soml.Node)
		want      []string
	}{
		{"매핑 없는 태그", `<slide><text>본문</text></slide>`, func(c []soml.Node) {
			c[0] = &soml.Element{Name: "box"}
		}, []string{"매핑이 없다"}},
		{"해석 안 된 외관", `<slide><text>본문</text></slide>`, func(c []soml.Node) {
			c[0].(*soml.Element).Appearance = nil
		}, []string{"core.Resolve"}},
		{"해석 안 된 인라인", `<slide><p><strong>강조</strong></p></slide>`, func(c []soml.Node) {
			c[0].(*soml.Element).Children[0].(*soml.Element).Appearance = nil
		}, []string{"core.Resolve"}},
		{"Link 없는 a", `<slide><p><a href="https://example.com">링크</a></p></slide>`, func(c []soml.Node) {
			c[0].(*soml.Element).Children[0].(*soml.Element).Appearance.Link = ""
		}, []string{"core.Resolve"}},
		// 문자 데이터는 측정 대상이 아니라 지오메트리는 그대로 유효하다 — 순회가 건너뛰면 글자가 사라진다.
		{"컨테이너 안 문자 데이터", `<slide><row><text>본문</text></row></slide>`, func(c []soml.Node) {
			row := c[0].(*soml.Element)
			row.Children = append(row.Children, &soml.Text{Value: "떠 있는 글자"})
		}, []string{"요소가 아닌 노드", "s0.0.1"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			deck := resolvedDeck(t, tc.src)
			tc.breakIt(deck.Slides[0].Content)
			rects := make([]geom.Rect, len(geom.Refs(deck)))
			for i := range rects {
				rects[i] = geom.Rect{W: 10, H: 10}
			}
			g := measuredGeometry(t, deck, rects...)
			_, err := NewWithGeometry(g).Render(deck)
			if err == nil {
				t.Fatal("렌더가 멈추지 않았다")
			}
			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("에러에 %q가 없다: %v", want, err)
				}
			}
		})
	}
}

// renderedKinds는 이 렌더러가 다루는 종류의 얼굴이고(RenderedKinds — render 패키지의 대조가
// 읽는다) shapeWriterFor는 그중 **도형이 되는** 종류의 실제 동작이다. 둘의 관계는 웹과
// 다르다: 포함이지 일치가 아니다 — `<li>`와 인라인은 부모 도형에 흡수되고 레이아웃
// 컨테이너·`<spacer/>`는 사라진다. 그 비대칭이 의도라는 것을 여기서 고정한다 (TASKS P0.2).
func TestRenderedKindsMatchShapeDispatch(t *testing.T) {
	listed := make(map[soml.ElementKind]bool, len(renderedKinds))
	for _, kind := range RenderedKinds() {
		listed[kind] = true
	}
	// 도형이 되는 종류는 정확히 다섯이다 — 이 목록이 늘어나는 것이 새 산출 요소가 생겼다는
	// 뜻이다 (표가 p:graphicFrame으로 다섯째가 됐다: 행·셀은 그 안에 접히므로 여기 없다).
	shapeKinds := map[soml.ElementKind]bool{
		soml.KindBlock:         true,
		soml.KindListContainer: true,
		soml.KindShape:         true,
		soml.KindImage:         true,
		soml.KindTable:         true,
	}
	for _, kind := range soml.ElementKinds() {
		if got, want := shapeWriterFor(kind) != nil, shapeKinds[kind]; got != want {
			t.Errorf("%v: shapeWriterFor 있음=%v, want %v", kind, got, want)
		}
		if shapeKinds[kind] && !listed[kind] {
			t.Errorf("%v는 도형이 되는데 renderedKinds 목록에 없다", kind)
		}
	}
	// `<text>`는 카탈로그 밖이지만 블록과 같은 텍스트 박스가 된다 (writeShapeNode) —
	// 이름으로 다루므로 종류 표에는 자기 자리가 없다.
	if shapeWriterFor(soml.KindOf(soml.TextElement)) != nil {
		t.Error("shapeWriterFor(KindNone)이 함수를 준다 — <text>는 writeShapeNode가 이름으로 다룬다")
	}
}
