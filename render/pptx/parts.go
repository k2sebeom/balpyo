package pptx

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/soml"
)

// OOXML 파트 경로. 관계 파일이 서로를 이 경로로 가리키므로 한곳에 모아 둔다.
const (
	contentTypesPart    = "[Content_Types].xml"
	rootRelsPart        = "_rels/.rels"
	presentationPart    = "ppt/presentation.xml"
	presRelsPart        = "ppt/_rels/presentation.xml.rels"
	masterPart          = "ppt/slideMasters/slideMaster1.xml"
	masterRelsPart      = "ppt/slideMasters/_rels/slideMaster1.xml.rels"
	layoutPart          = "ppt/slideLayouts/slideLayout1.xml"
	layoutRelsPart      = "ppt/slideLayouts/_rels/slideLayout1.xml.rels"
	themePart           = "ppt/theme/theme1.xml"
	notesMasterPart     = "ppt/notesMasters/notesMaster1.xml"
	notesMasterRelsPart = "ppt/notesMasters/_rels/notesMaster1.xml.rels"
	corePropsPart       = "docProps/core.xml"
	appPropsPart        = "docProps/app.xml"
)

// slidePart는 n번째(1-based) 슬라이드 파트 경로다.
func slidePart(n int) string { return fmt.Sprintf("ppt/slides/slide%d.xml", n) }

// slideRelsPart는 n번째 슬라이드의 관계 파일 경로다.
func slideRelsPart(n int) string { return fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", n) }

// notesSlidePart는 n번째 슬라이드의 노트 슬라이드 파트 경로다.
//
// **번호 규칙: N은 노트가 붙는 슬라이드의 번호다** — slidePart(n)과 같은 n. 노트 없는
// 슬라이드에서 번호가 비어 뛰지만(OPC 파트 이름은 임의 문자열이고 소비자는 관계만 따라간다),
// "노트가 있는 것만 1부터" 세면 별도 카운터가 생기고, 그 카운터와 rels의 슬라이드 번호가
// 어긋나는 순간 노트가 엉뚱한 슬라이드에 붙는다 — 파일은 열리므로 조용한 실패다. 파트 이름을
// 슬라이드 번호에서 직접 유도하면 그 어긋남을 만들 방법 자체가 없다.
func notesSlidePart(n int) string { return fmt.Sprintf("ppt/notesSlides/notesSlide%d.xml", n) }

// notesSlideRelsPart는 n번째 슬라이드의 노트 슬라이드 관계 파일 경로다.
func notesSlideRelsPart(n int) string {
	return fmt.Sprintf("ppt/notesSlides/_rels/notesSlide%d.xml.rels", n)
}

// part는 패키지에 들어갈 파트 하나다. 순서를 지키기 위해 맵이 아니라 슬라이스로 다룬다.
type part struct {
	name string
	data []byte
}

// buildParts는 덱 하나를 OOXML 파트 목록으로 만든다: 프레젠테이션 + 마스터 + 빈 레이아웃 +
// 테마(토큰 표에서 컴파일) + **미디어 파트**(에셋 표에서 — 유일하게 XML이 아닌 파트다) +
// 슬라이드당 spTree + 노트가 있으면 노트 마스터와 노트 슬라이드. 레이아웃 카탈로그·
// 플레이스홀더는 Iteration 10이다.
//
// g는 지오메트리 패스가 잰 좌표다. 좌표가 필요한 요소가 없는 덱(빈 슬라이드만)은 nil로 온다 —
// 요소가 있는데 nil이면 Renderer.checkGeometry가 이미 멈췄다.
func buildParts(deck *soml.Deck, g *geom.Geometry) ([]part, error) {
	cx, cy := SlideSize(deck.Stage)
	if cx <= 0 || cy <= 0 {
		return nil, fmt.Errorf("논리 스테이지 크기가 유효하지 않다: %dx%d", deck.Stage.Width, deck.Stage.Height)
	}
	if len(deck.Theme.Colors) == 0 {
		// 테마 파트는 토큰 표에서만 나온다 — 표 없이 조립하면 clrScheme을 채울 수 없고,
		// 하드코딩 기본값으로 채우면 웹과 다른 색을 내는 파일이 조용히 만들어진다.
		return nil, fmt.Errorf("덱에 테마 토큰 표가 없다 — core.Resolve를 거치지 않은 덱이다")
	}
	clrScheme, err := clrSchemeXML(deck.Theme)
	if err != nil {
		return nil, err
	}
	fontScheme, err := fontSchemeXML(deck.Theme)
	if err != nil {
		return nil, err
	}

	data := struct {
		Deck             *soml.Deck
		SlideCX          int64
		SlideCY          int64
		NotesCX          int64
		NotesCY          int64
		SlideCount       int
		SlideRelIDs      []string
		ThemeRelID       string
		NotesMasterRelID string
		ClrScheme        string
		FontScheme       string
		NotesLayout      notesLayout
		MediaDefaults    []mediaDefault
		SlotLayouts      []slotLayoutPart
		MasterPh         masterPlaceholders
	}{
		Deck:       deck,
		SlideCX:    cx,
		SlideCY:    cy,
		SlideCount: len(deck.Slides),
		// 노트 페이지는 세로 letter 크기 관례를 따른다.
		NotesCX:    EMU(720),
		NotesCY:    EMU(960),
		ClrScheme:  clrScheme,
		FontScheme: fontScheme,
	}
	// presentation.xml.rels: rId1 = 마스터, rId2.. = 슬라이드, 그다음 = 테마.
	for i := range deck.Slides {
		data.SlideRelIDs = append(data.SlideRelIDs, fmt.Sprintf("rId%d", i+2))
	}
	data.ThemeRelID = fmt.Sprintf("rId%d", len(deck.Slides)+2)
	// 노트 마스터는 **테마 뒤에 덧붙인다** (rId = 슬라이드 수 + 3). 기존 규칙(rId1 = 마스터,
	// rId2.. = 슬라이드, 그다음 = 테마)의 어느 번호도 밀지 않는 유일한 자리다 — 중간에 끼우면
	// 슬라이드 관계가 한 칸씩 밀려 엉뚱한 파트를 가리키고, 그 파일은 열리기는 한다.
	// 노트가 하나도 없는 덱은 노트 마스터 자체를 내지 않는다 (빈 문자열 = 템플릿 조건 꺼짐).
	if deckHasNotes(deck) {
		data.NotesMasterRelID = fmt.Sprintf("rId%d", len(deck.Slides)+3)
		data.NotesLayout = notesMasterLayout(data.NotesCX, data.NotesCY, cx, cy)
	}

	// 미디어 파트의 콘텐츠 타입은 **덱에 실제로 든 포맷만** 선언한다 (mediaDefaults) —
	// Content_Types 템플릿이 이 목록을 돈다. 파트 자체는 아래에서 붙인다.
	mediaDefs, err := mediaDefaults(deck)
	if err != nil {
		return nil, err
	}
	data.MediaDefaults = mediaDefs
	// 슬롯을 쓴 슬라이드가 있으면 그 레이아웃마다 플레이스홀더 파트를 낸다 (ADR 0035) —
	// 없으면 nil이라 마스터·Content_Types·슬라이드 관계가 예전과 바이트 동일하다 (골든 불변).
	data.SlotLayouts = deckSlotLayouts(deck)
	if len(data.SlotLayouts) > 0 {
		// 슬롯 레이아웃 플레이스홀더가 상속받을 마스터 title/body 프로토타입 좌표.
		data.MasterPh = masterPlaceholderLayout(cx, cy)
	}
	templates := []struct {
		name string
		text string
	}{
		{contentTypesPart, contentTypesTemplate},
		{rootRelsPart, rootRelsTemplate},
		{corePropsPart, corePropsTemplate},
		{appPropsPart, appPropsTemplate},
		{presentationPart, presentationTemplate},
		{presRelsPart, presRelsTemplate},
		{masterPart, masterTemplate},
		{masterRelsPart, masterRelsTemplate},
		{layoutPart, layoutTemplate},
		{layoutRelsPart, layoutRelsTemplate},
		{themePart, themeTemplate},
	}
	if data.NotesMasterRelID != "" {
		templates = append(templates,
			struct{ name, text string }{notesMasterPart, notesMasterTemplate},
			struct{ name, text string }{notesMasterRelsPart, notesMasterRelsXML},
		)
	}
	parts := make([]part, 0, len(templates)+len(deck.Assets)+3*len(deck.Slides))
	for _, tpl := range templates {
		rendered, err := renderTemplate(tpl.name, tpl.text, data)
		if err != nil {
			return nil, err
		}
		parts = append(parts, part{name: tpl.name, data: rendered})
	}
	// 슬롯 레이아웃 파트와 그 관계 파일을 덧붙인다 (blank 레이아웃 뒤). 관계 파일은 blank와
	// 같은 내용이다 (마스터를 가리킬 뿐) — layoutRelsTemplate을 그대로 재사용한다. 슬롯을 쓴
	// 슬라이드가 없으면 SlotLayouts가 nil이라 이 루프가 돌지 않는다 (골든 불변).
	for _, sl := range data.SlotLayouts {
		layoutXML, err := renderTemplate(sl.File, slotLayoutTemplate, sl)
		if err != nil {
			return nil, err
		}
		relsXML, err := renderTemplate(sl.File+".rels", layoutRelsTemplate, nil)
		if err != nil {
			return nil, err
		}
		parts = append(parts,
			part{name: "ppt/slideLayouts/" + sl.File, data: layoutXML},
			part{name: "ppt/slideLayouts/_rels/" + sl.File + ".rels", data: relsXML},
		)
	}
	// 미디어는 **덱의 것**이다 (테마와 같은 층) — 슬라이드마다 다시 세지 않는다. 파트를
	// 슬라이드보다 앞에 두는 이유는 패키지를 풀어 본 사람이 그림을 슬라이드보다 먼저 만나는
	// 것이 자연스럽기 때문이고, 순서는 결정적이면 어디든 유효하다 (zipParts는 이 순서를
	// 지킨다 — [Content_Types].xml만 첫 엔트리여야 한다).
	media, err := mediaParts(deck)
	if err != nil {
		return nil, err
	}
	parts = append(parts, media...)

	for i, slide := range deck.Slides {
		n := i + 1
		shapes, rels, err := shapesXML(deck, slide, g)
		if err != nil {
			return nil, err
		}
		rendered, err := renderTemplate(slidePart(n), slideTemplate, struct {
			*soml.Slide
			Shapes string
		}{slide, shapes})
		if err != nil {
			return nil, err
		}
		// 관계의 id·순서는 shapesXML의 순회가 배정한 그대로다 — `hlinkClick@r:id`·
		// `blip@r:embed`와 이 목록이 다른 곳에서 나오면 어긋난 순간이 파일을 열 때까지
		// 조용하다 (링크가 죽거나 그림 자리가 빈다).
		relsPart, err := renderTemplate(slideRelsPart(n), slideRelsTemplate, struct {
			*soml.Slide
			N          int
			LayoutFile string
			Rels       []slideRel
		}{slide, n, slideLayoutFile(slide, data.SlotLayouts), rels.items})
		if err != nil {
			return nil, err
		}
		parts = append(parts,
			part{name: slidePart(n), data: rendered},
			part{name: slideRelsPart(n), data: relsPart},
		)
		// 빈 `<note></note>`는 Notes == ""이므로 노트 파트를 내지 않는다 — 웹 렌더러가
		// balpyo-notes div를 내지 않는 판정과 같다 (render/web.writeSlide).
		if slide.Notes == "" {
			continue
		}
		notes, err := renderTemplate(notesSlidePart(n), notesSlideTemplate, struct {
			*soml.Slide
			N int
		}{slide, n})
		if err != nil {
			return nil, err
		}
		notesRels, err := renderTemplate(notesSlideRelsPart(n), notesSlideRelsTemplate, struct{ N int }{n})
		if err != nil {
			return nil, err
		}
		parts = append(parts,
			part{name: notesSlidePart(n), data: notes},
			part{name: notesSlideRelsPart(n), data: notesRels},
		)
	}
	return parts, nil
}

// deckHasNotes는 노트 슬라이드가 하나라도 필요한지 본다 — 노트 마스터는 이때만 낸다.
func deckHasNotes(deck *soml.Deck) bool {
	for _, slide := range deck.Slides {
		if slide.Notes != "" {
			return true
		}
	}
	return false
}

// notesLayout은 노트 마스터 플레이스홀더의 배치다 (EMU).
//
// 노트는 화면에 렌더되지 않으므로 지오메트리 패스의 대상이 아니다 — 좌표는 여기서 정해
// 마스터에 넣고, 노트 슬라이드는 상속만 받는다 (플레이스홀더에 xfrm을 내지 않는다).
type notesLayout struct {
	ImgX, ImgY, ImgCX, ImgCY     int64
	BodyX, BodyY, BodyCX, BodyCY int64
}

// notesMasterLayout은 노트 페이지 안에 슬라이드 이미지와 본문 자리를 잡는다.
//
// PowerPoint 기본 노트 마스터의 관례를 따른다: 좌우 0.75in 여백, 이미지는 위(y=1.25in),
// 본문은 그 아래. 이미지 높이만 계산이다 — 슬라이드 종횡비를 따라야 4:3 덱에서 이미지와
// 본문이 겹치지 않는다 (PowerPoint 기본값은 종횡비별 파일에 박혀 있어 그대로 못 쓴다).
func notesMasterLayout(pageCX, pageCY, slideCX, slideCY int64) notesLayout {
	const (
		margin = EMUPerInch * 3 / 4 // 좌우·아래 여백 0.75in
		imgY   = EMUPerInch * 5 / 4 // 이미지 상단 1.25in
		gap    = EMUPerInch / 2     // 이미지와 본문 사이 0.5in
	)
	imgCX := pageCX - 2*margin
	imgCY := imgCX * slideCY / slideCX
	bodyY := imgY + imgCY + gap
	return notesLayout{
		ImgX: margin, ImgY: imgY, ImgCX: imgCX, ImgCY: imgCY,
		BodyX: margin, BodyY: bodyY, BodyCX: imgCX, BodyCY: pageCY - bodyY - margin,
	}
}

// firstSlideID는 OOXML의 슬라이드 ID 시작값이다. 256보다 작은 값은 예약되어 있다.
const firstSlideID = 256

// partFuncs는 파트 템플릿이 쓰는 함수다.
//
// XML 이스케이프는 text/template의 기본 동작이 아니므로 `xml`을 명시적으로 쓴다.
var partFuncs = template.FuncMap{
	"xml":     escapeXML,
	"inc":     func(i int) int { return i + 1 },
	"slideID": func(i int) int { return firstSlideID + i },
}

// renderTemplate는 파트 템플릿 하나를 렌더한다.
func renderTemplate(name, text string, data any) ([]byte, error) {
	tpl, err := template.New(name).Funcs(partFuncs).Parse(text)
	if err != nil {
		return nil, fmt.Errorf("%s 템플릿 파싱 실패: %w", name, err)
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("%s 렌더 실패: %w", name, err)
	}
	return buf.Bytes(), nil
}

// escapeXML은 텍스트를 XML 문자 데이터로 안전하게 만든다.
func escapeXML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")
	return r.Replace(s)
}

const xmlDecl = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n"

const contentTypesTemplate = xmlDecl + `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
{{- range .MediaDefaults}}
  <Default Extension="{{.Ext}}" ContentType="{{.ContentType}}"/>
{{- end}}
  <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>
  <Override PartName="/ppt/slideMasters/slideMaster1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideMaster+xml"/>
  <Override PartName="/ppt/slideLayouts/slideLayout1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideLayout+xml"/>
{{- range .SlotLayouts}}
  <Override PartName="/ppt/slideLayouts/{{.File}}" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideLayout+xml"/>
{{- end}}
  <Override PartName="/ppt/theme/theme1.xml" ContentType="application/vnd.openxmlformats-officedocument.theme+xml"/>
{{- if .NotesMasterRelID}}
  <Override PartName="/ppt/notesMasters/notesMaster1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.notesMaster+xml"/>
{{- end}}
{{- range $i, $s := .Deck.Slides}}
  <Override PartName="/ppt/slides/slide{{inc $i}}.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>
{{- if $s.Notes}}
  <Override PartName="/ppt/notesSlides/notesSlide{{inc $i}}.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.notesSlide+xml"/>
{{- end}}
{{- end}}
  <Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>
  <Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>
</Types>
`

const rootRelsTemplate = xmlDecl + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/>
</Relationships>
`

// 날짜를 넣지 않는다 — 같은 입력이 같은 바이트를 내야 골든 비교가 성립한다.
const corePropsTemplate = xmlDecl + `<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:dcmitype="http://purl.org/dc/dcmitype/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <dc:title>{{xml .Deck.Title}}</dc:title>
  <dc:creator>{{xml .Deck.Author}}</dc:creator>
  <cp:lastModifiedBy>balpyo</cp:lastModifiedBy>
  <cp:revision>1</cp:revision>
</cp:coreProperties>
`

const appPropsTemplate = xmlDecl + `<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties" xmlns:vt="http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes">
  <Application>balpyo</Application>
  <Slides>{{.SlideCount}}</Slides>
  <ScaleCrop>false</ScaleCrop>
  <LinksUpToDate>false</LinksUpToDate>
  <SharedDoc>false</SharedDoc>
  <HyperlinksChanged>false</HyperlinksChanged>
</Properties>
`

const presentationTemplate = xmlDecl + `<p:presentation xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" saveSubsetFonts="1">
  <p:sldMasterIdLst>
    <p:sldMasterId id="2147483648" r:id="rId1"/>
  </p:sldMasterIdLst>
{{- if .NotesMasterRelID}}
  <p:notesMasterIdLst>
    <p:notesMasterId r:id="{{.NotesMasterRelID}}"/>
  </p:notesMasterIdLst>
{{- end}}
  <p:sldIdLst>
{{- range $i, $rid := .SlideRelIDs}}
    <p:sldId id="{{slideID $i}}" r:id="{{$rid}}"/>
{{- end}}
  </p:sldIdLst>
  <p:sldSz cx="{{.SlideCX}}" cy="{{.SlideCY}}"/>
  <p:notesSz cx="{{.NotesCX}}" cy="{{.NotesCY}}"/>
</p:presentation>
`

// 관계 ID 배정: rId1 = 마스터, rId2.. = 슬라이드, 그다음 = 테마, **노트 마스터는 맨 뒤**.
// 테마 뒤에 덧붙이는 이유는 buildParts의 NotesMasterRelID 주석에 있다 — 앞쪽 번호를 밀면
// 슬라이드 관계가 엉뚱한 파트를 가리키고, 그 파일은 열리기는 한다.
const presRelsTemplate = xmlDecl + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="slideMasters/slideMaster1.xml"/>
{{- range $i, $rid := .SlideRelIDs}}
  <Relationship Id="{{$rid}}" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide{{inc $i}}.xml"/>
{{- end}}
  <Relationship Id="{{.ThemeRelID}}" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="theme/theme1.xml"/>
{{- if .NotesMasterRelID}}
  <Relationship Id="{{.NotesMasterRelID}}" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/notesMaster" Target="notesMasters/notesMaster1.xml"/>
{{- end}}
</Relationships>
`

// 마스터의 clrMap이 있어야 도형의 schemeClr 참조가 테마 색으로 해석된다 — 산출물이
// PowerPoint 안에서 테마 색 편집을 따라가는 근거다.
//
// 슬라이드 배경도 여기서 정해진다: bgRef의 bg1은 clrMap을 거쳐 테마 lt1이 되고, lt1은
// 토큰 표의 `bg` 슬롯에서 컴파일된다 (theme.go). 웹의 `.balpyo-slide` 배경이 같은 `bg`
// 토큰에서 나오므로(render/web.themeCSS) 두 타깃의 배경이 한 표에서 만난다 — 슬라이드
// 파트에 별도 p:bg를 내지 않는 이유다 (배경이 갈라지면 픽셀 비교가 전면 불일치로 뜬다).
const masterTemplate = xmlDecl + `<p:sldMaster xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld>
    <p:bg>
      <p:bgRef idx="1001">
        <a:schemeClr val="bg1"/>
      </p:bgRef>
    </p:bg>
    <p:spTree>
      <p:nvGrpSpPr>
        <p:cNvPr id="1" name=""/>
        <p:cNvGrpSpPr/>
        <p:nvPr/>
      </p:nvGrpSpPr>
      <p:grpSpPr>
        <a:xfrm>
          <a:off x="0" y="0"/>
          <a:ext cx="0" cy="0"/>
          <a:chOff x="0" y="0"/>
          <a:chExt cx="0" cy="0"/>
        </a:xfrm>
      </p:grpSpPr>
{{- if .SlotLayouts}}
      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="2" name="Title Placeholder 1"/>
          <p:cNvSpPr>
            <a:spLocks noGrp="1"/>
          </p:cNvSpPr>
          <p:nvPr>
            <p:ph type="title"/>
          </p:nvPr>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm>
            <a:off x="{{.MasterPh.TitleX}}" y="{{.MasterPh.TitleY}}"/>
            <a:ext cx="{{.MasterPh.TitleCX}}" cy="{{.MasterPh.TitleCY}}"/>
          </a:xfrm>
          <a:prstGeom prst="rect">
            <a:avLst/>
          </a:prstGeom>
        </p:spPr>
        <p:txBody>
          <a:bodyPr/>
          <a:lstStyle/>
          <a:p>
            <a:endParaRPr/>
          </a:p>
        </p:txBody>
      </p:sp>
      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="3" name="Body Placeholder 2"/>
          <p:cNvSpPr>
            <a:spLocks noGrp="1"/>
          </p:cNvSpPr>
          <p:nvPr>
            <p:ph type="body" idx="1"/>
          </p:nvPr>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm>
            <a:off x="{{.MasterPh.BodyX}}" y="{{.MasterPh.BodyY}}"/>
            <a:ext cx="{{.MasterPh.BodyCX}}" cy="{{.MasterPh.BodyCY}}"/>
          </a:xfrm>
          <a:prstGeom prst="rect">
            <a:avLst/>
          </a:prstGeom>
        </p:spPr>
        <p:txBody>
          <a:bodyPr/>
          <a:lstStyle/>
          <a:p>
            <a:endParaRPr/>
          </a:p>
        </p:txBody>
      </p:sp>
{{- end}}
    </p:spTree>
  </p:cSld>
  <p:clrMap bg1="lt1" tx1="dk1" bg2="lt2" tx2="dk2" accent1="accent1" accent2="accent2" accent3="accent3" accent4="accent4" accent5="accent5" accent6="accent6" hlink="hlink" folHlink="folHlink"/>
  <p:sldLayoutIdLst>
    <p:sldLayoutId id="2147483649" r:id="rId1"/>
{{- range .SlotLayouts}}
    <p:sldLayoutId id="{{.SldLayoutID}}" r:id="{{.MasterRelID}}"/>
{{- end}}
  </p:sldLayoutIdLst>
{{- if .SlotLayouts}}
  <p:txStyles>
    <p:titleStyle>
      <a:lvl1pPr algn="l"><a:defRPr sz="4400"/></a:lvl1pPr>
    </p:titleStyle>
    <p:bodyStyle>
      <a:lvl1pPr><a:defRPr sz="2400"/></a:lvl1pPr>
    </p:bodyStyle>
    <p:otherStyle/>
  </p:txStyles>
{{- end}}
</p:sldMaster>
`

// rId1 = blank 레이아웃, rId2 = theme (예전 그대로), 그다음부터 슬롯 레이아웃 파트들이다
// (MasterRelID = rId3..). 슬롯을 쓴 슬라이드가 없으면 range가 비어 예전과 바이트 동일하다.
const masterRelsTemplate = xmlDecl + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="../theme/theme1.xml"/>
{{- range .SlotLayouts}}
  <Relationship Id="{{.MasterRelID}}" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/{{.File}}"/>
{{- end}}
</Relationships>
`

// 레이아웃은 아직 빈 레이아웃 하나뿐이다. 표준 레이아웃 카탈로그 11종과 슬롯 →
// 플레이스홀더 매핑은 Iteration 10이다.
const layoutTemplate = xmlDecl + `<p:sldLayout xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" type="blank" preserve="1">
  <p:cSld name="Blank">
    <p:spTree>
      <p:nvGrpSpPr>
        <p:cNvPr id="1" name=""/>
        <p:cNvGrpSpPr/>
        <p:nvPr/>
      </p:nvGrpSpPr>
      <p:grpSpPr>
        <a:xfrm>
          <a:off x="0" y="0"/>
          <a:ext cx="0" cy="0"/>
          <a:chOff x="0" y="0"/>
          <a:chExt cx="0" cy="0"/>
        </a:xfrm>
      </p:grpSpPr>
    </p:spTree>
  </p:cSld>
  <p:clrMapOvr>
    <a:masterClrMapping/>
  </p:clrMapOvr>
</p:sldLayout>
`

const layoutRelsTemplate = xmlDecl + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="../slideMasters/slideMaster1.xml"/>
</Relationships>
`

// slotLayoutTemplate은 슬롯 레이아웃 파트 하나다 (ADR 0035). 이 레이아웃이 선언하는 슬롯마다
// 플레이스홀더 `<p:sp>`를 낸다 — 슬라이드 도형의 `<p:ph type/idx>`가 이것에 연결되어야
// PowerPoint가 진짜 플레이스홀더로 인식한다 (buildParts의 슬롯 파트 주석).
//
// **플레이스홀더에 xfrm을 내지 않는다**: 좌표는 마스터 title/body 프로토타입에서 상속받고
// 슬라이드가 잰 rect로 덮어쓴다 (layoutPlaceholder의 근거 — 레이아웃 위치는 산출물 픽셀에
// 닿지 않는다). type="cust"는 사용자 정의 레이아웃의 표준 종류값이다 (ST_SlideLayoutType).
const slotLayoutTemplate = xmlDecl + `<p:sldLayout xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" type="cust" preserve="1">
  <p:cSld name="{{.Layout}}">
    <p:spTree>
      <p:nvGrpSpPr>
        <p:cNvPr id="1" name=""/>
        <p:cNvGrpSpPr/>
        <p:nvPr/>
      </p:nvGrpSpPr>
      <p:grpSpPr>
        <a:xfrm>
          <a:off x="0" y="0"/>
          <a:ext cx="0" cy="0"/>
          <a:chOff x="0" y="0"/>
          <a:chExt cx="0" cy="0"/>
        </a:xfrm>
      </p:grpSpPr>
{{- range .Placeholders}}
      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="{{.ID}}" name="{{xml .Name}}"/>
          <p:cNvSpPr>
            <a:spLocks noGrp="1"/>
          </p:cNvSpPr>
          <p:nvPr>
            <p:ph type="{{.Type}}"{{if .HasIdx}} idx="{{.Idx}}"{{end}}/>
          </p:nvPr>
        </p:nvSpPr>
        <p:spPr/>
        <p:txBody>
          <a:bodyPr/>
          <a:lstStyle/>
          <a:p>
            <a:endParaRPr/>
          </a:p>
        </p:txBody>
      </p:sp>
{{- end}}
    </p:spTree>
  </p:cSld>
  <p:clrMapOvr>
    <a:masterClrMapping/>
  </p:clrMapOvr>
</p:sldLayout>
`

// 슬라이드의 spTree는 shapesXML이 컴파일한 도형 목록을 담는다 (문서 순서 = z-순서).
// name 속성에 슬라이드 ID를 넣어 PowerPoint의 선택 창에서 어느 슬라이드인지 보이게 한다.
const slideTemplate = xmlDecl + `<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld name="{{xml .ID}}">
    <p:spTree>
      <p:nvGrpSpPr>
        <p:cNvPr id="1" name=""/>
        <p:cNvGrpSpPr/>
        <p:nvPr/>
      </p:nvGrpSpPr>
      <p:grpSpPr>
        <a:xfrm>
          <a:off x="0" y="0"/>
          <a:ext cx="0" cy="0"/>
          <a:chOff x="0" y="0"/>
          <a:chExt cx="0" cy="0"/>
        </a:xfrm>
      </p:grpSpPr>{{.Shapes}}
    </p:spTree>
  </p:cSld>
  <p:clrMapOvr>
    <a:masterClrMapping/>
  </p:clrMapOvr>
</p:sld>
`

// 관계 id 배정: rId1 = 레이아웃, rId2 = 노트 슬라이드(없어도 예약 — slideRelBase의 주석),
// rId3부터 **외부 하이퍼링크와 미디어(그림)가 문서 순서로** 붙는다 (shape.go의 slideRels가
// 한 카운터로 배정한다 — 종류마다 구간을 나누지 않은 근거는 그 타입의 주석에 있다).
//
// Type과 TargetMode가 데이터인 것이 이 템플릿의 전부다: 한 목록에 두 종류가 섞이므로 템플릿이
// 종류를 알면 배정과 출력이 두 곳에서 종류를 판단하게 된다. 하이퍼링크의
// `TargetMode="External"`이 빠지면 소비자가 URL을 패키지 내부 파트 경로로 읽어 링크가 조용히
// 죽고, 반대로 미디어에 그것이 붙으면 그림 자리가 빈다.
const slideRelsTemplate = xmlDecl + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/{{.LayoutFile}}"/>
{{- if .Notes}}
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/notesSlide" Target="../notesSlides/notesSlide{{.N}}.xml"/>
{{- end}}
{{- range .Rels}}
  <Relationship Id="{{.ID}}" Type="{{.Type}}" Target="{{xml .Target}}"{{if .TargetMode}} TargetMode="{{.TargetMode}}"{{end}}/>
{{- end}}
</Relationships>
`

// 노트 마스터는 하나다. 플레이스홀더 좌표는 notesMasterLayout이 정하고, 노트 슬라이드는
// xfrm 없이 이 마스터에서 상속받는다 — 노트는 화면에 렌더되지 않으므로 잰 박스가 없다.
//
// notesStyle의 타이포그래피(12pt)는 자리 표시자다 — 테마 토큰 표에서 컴파일하는 일은
// Iteration 10의 테마 패키지 포맷이 맡는다 (노트는 발표자 화면 전용이라 웹·PPTX 픽셀
// 일치 계약 밖이고, 지금 하드코딩해도 두 타깃이 갈라지지 않는다).
const notesMasterTemplate = xmlDecl + `<p:notesMaster xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld>
    <p:bg>
      <p:bgRef idx="1001">
        <a:schemeClr val="bg1"/>
      </p:bgRef>
    </p:bg>
    <p:spTree>
      <p:nvGrpSpPr>
        <p:cNvPr id="1" name=""/>
        <p:cNvGrpSpPr/>
        <p:nvPr/>
      </p:nvGrpSpPr>
      <p:grpSpPr>
        <a:xfrm>
          <a:off x="0" y="0"/>
          <a:ext cx="0" cy="0"/>
          <a:chOff x="0" y="0"/>
          <a:chExt cx="0" cy="0"/>
        </a:xfrm>
      </p:grpSpPr>
      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="2" name="Slide Image Placeholder 1"/>
          <p:cNvSpPr>
            <a:spLocks noGrp="1" noRot="1" noChangeAspect="1"/>
          </p:cNvSpPr>
          <p:nvPr>
            <p:ph type="sldImg"/>
          </p:nvPr>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm>
            <a:off x="{{.NotesLayout.ImgX}}" y="{{.NotesLayout.ImgY}}"/>
            <a:ext cx="{{.NotesLayout.ImgCX}}" cy="{{.NotesLayout.ImgCY}}"/>
          </a:xfrm>
          <a:prstGeom prst="rect">
            <a:avLst/>
          </a:prstGeom>
        </p:spPr>
      </p:sp>
      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="3" name="Notes Placeholder 2"/>
          <p:cNvSpPr>
            <a:spLocks noGrp="1"/>
          </p:cNvSpPr>
          <p:nvPr>
            <p:ph type="body" idx="1"/>
          </p:nvPr>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm>
            <a:off x="{{.NotesLayout.BodyX}}" y="{{.NotesLayout.BodyY}}"/>
            <a:ext cx="{{.NotesLayout.BodyCX}}" cy="{{.NotesLayout.BodyCY}}"/>
          </a:xfrm>
          <a:prstGeom prst="rect">
            <a:avLst/>
          </a:prstGeom>
        </p:spPr>
        <p:txBody>
          <a:bodyPr/>
          <a:lstStyle/>
          <a:p>
            <a:endParaRPr/>
          </a:p>
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
  <p:clrMap bg1="lt1" tx1="dk1" bg2="lt2" tx2="dk2" accent1="accent1" accent2="accent2" accent3="accent3" accent4="accent4" accent5="accent5" accent6="accent6" hlink="hlink" folHlink="folHlink"/>
  <p:notesStyle>
    <a:lvl1pPr>
      <a:defRPr sz="1200"/>
    </a:lvl1pPr>
  </p:notesStyle>
</p:notesMaster>
`

// 노트 마스터의 테마는 슬라이드 마스터와 같은 theme1.xml을 가리킨다 — 노트 마스터 전용
// 테마 파트를 따로 내지 않는다 (한 파트가 여러 관계의 대상이 되는 것은 OPC에서 유효하다).
const notesMasterRelsXML = xmlDecl + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="../theme/theme1.xml"/>
</Relationships>
`

// 노트 슬라이드는 플레이스홀더 두 개다: 슬라이드 이미지(sldImg)와 본문(body idx=1).
// 둘 다 xfrm을 내지 않는다 — 좌표는 노트 마스터에서 상속받는다 (notesMasterTemplate).
// 본문 텍스트는 slide.Notes 그대로다: 파서가 개행을 접었으므로 웹의 balpyo-notes와
// 같은 문자열이고, 문단은 하나다 (render/web.writeSlide와 같은 결정 — 원천은 soml.Parse).
const notesSlideTemplate = xmlDecl + `<p:notes xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld name="{{xml .ID}}">
    <p:spTree>
      <p:nvGrpSpPr>
        <p:cNvPr id="1" name=""/>
        <p:cNvGrpSpPr/>
        <p:nvPr/>
      </p:nvGrpSpPr>
      <p:grpSpPr>
        <a:xfrm>
          <a:off x="0" y="0"/>
          <a:ext cx="0" cy="0"/>
          <a:chOff x="0" y="0"/>
          <a:chExt cx="0" cy="0"/>
        </a:xfrm>
      </p:grpSpPr>
      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="2" name="Slide Image Placeholder 1"/>
          <p:cNvSpPr>
            <a:spLocks noGrp="1" noRot="1" noChangeAspect="1"/>
          </p:cNvSpPr>
          <p:nvPr>
            <p:ph type="sldImg"/>
          </p:nvPr>
        </p:nvSpPr>
        <p:spPr/>
      </p:sp>
      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="3" name="Notes Placeholder 2"/>
          <p:cNvSpPr>
            <a:spLocks noGrp="1"/>
          </p:cNvSpPr>
          <p:nvPr>
            <p:ph type="body" idx="1"/>
          </p:nvPr>
        </p:nvSpPr>
        <p:spPr/>
        <p:txBody>
          <a:bodyPr/>
          <a:lstStyle/>
          <a:p>
            <a:r>
              <a:t>{{xml .Notes}}</a:t>
            </a:r>
          </a:p>
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
  <p:clrMapOvr>
    <a:masterClrMapping/>
  </p:clrMapOvr>
</p:notes>
`

const notesSlideRelsTemplate = xmlDecl + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/notesMaster" Target="../notesMasters/notesMaster1.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="../slides/slide{{.N}}.xml"/>
</Relationships>
`

// 테마 파트의 clrScheme·fontScheme은 덱의 토큰 표에서 컴파일된다 (theme.go) — 웹의
// CSS 변수와 같은 한 표다. 값이 두 곳에 있으면 `color="accent1"` 텍스트가 웹과 PPTX에서
// 다른 색으로 나오고, 토큰 보존 계약이 겉으로만 지켜진다. fmtScheme만 하드코딩으로 남는다
// (도형 기본 스타일 — 테마 패키지 포맷이 Iteration 10에서 해체한다).
// 폰트는 테마 표가 Noto를 가리킨다 — v1은 폰트를 임베드하지 않는다 (PRD §13-10).
const themeTemplate = xmlDecl + `<a:theme xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" name="{{xml .Deck.Theme.Name}}">
  <a:themeElements>
{{.ClrScheme}}
{{.FontScheme}}
    <a:fmtScheme name="{{xml .Deck.Theme.Name}}">
      <a:fillStyleLst>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
      </a:fillStyleLst>
      <a:lnStyleLst>
        <a:ln w="9525" cap="flat" cmpd="sng" algn="ctr"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill><a:prstDash val="solid"/></a:ln>
        <a:ln w="19050" cap="flat" cmpd="sng" algn="ctr"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill><a:prstDash val="solid"/></a:ln>
        <a:ln w="28575" cap="flat" cmpd="sng" algn="ctr"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill><a:prstDash val="solid"/></a:ln>
      </a:lnStyleLst>
      <a:effectStyleLst>
        <a:effectStyle><a:effectLst/></a:effectStyle>
        <a:effectStyle><a:effectLst/></a:effectStyle>
        <a:effectStyle><a:effectLst/></a:effectStyle>
      </a:effectStyleLst>
      <a:bgFillStyleLst>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
      </a:bgFillStyleLst>
    </a:fmtScheme>
  </a:themeElements>
</a:theme>
`
