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
	contentTypesPart = "[Content_Types].xml"
	rootRelsPart     = "_rels/.rels"
	presentationPart = "ppt/presentation.xml"
	presRelsPart     = "ppt/_rels/presentation.xml.rels"
	masterPart       = "ppt/slideMasters/slideMaster1.xml"
	masterRelsPart   = "ppt/slideMasters/_rels/slideMaster1.xml.rels"
	layoutPart       = "ppt/slideLayouts/slideLayout1.xml"
	layoutRelsPart   = "ppt/slideLayouts/_rels/slideLayout1.xml.rels"
	themePart        = "ppt/theme/theme1.xml"
	corePropsPart    = "docProps/core.xml"
	appPropsPart     = "docProps/app.xml"
)

// slidePart는 n번째(1-based) 슬라이드 파트 경로다.
func slidePart(n int) string { return fmt.Sprintf("ppt/slides/slide%d.xml", n) }

// slideRelsPart는 n번째 슬라이드의 관계 파일 경로다.
func slideRelsPart(n int) string { return fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", n) }

// part는 패키지에 들어갈 파트 하나다. 순서를 지키기 위해 맵이 아니라 슬라이스로 다룬다.
type part struct {
	name string
	data []byte
}

// buildParts는 덱 하나를 OOXML 파트 목록으로 만든다: 프레젠테이션 + 마스터 + 빈 레이아웃 +
// 테마(토큰 표에서 컴파일) + 슬라이드당 spTree(텍스트 박스). 도형 12종은 Iteration 4,
// 레이아웃 카탈로그·플레이스홀더는 Iteration 10이다.
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
		Deck        *soml.Deck
		SlideCX     int64
		SlideCY     int64
		NotesCX     int64
		NotesCY     int64
		SlideCount  int
		SlideRelIDs []string
		ThemeRelID  string
		ClrScheme   string
		FontScheme  string
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
	// presentation.xml.rels: rId1 = 마스터, rId2.. = 슬라이드, 마지막 = 테마.
	for i := range deck.Slides {
		data.SlideRelIDs = append(data.SlideRelIDs, fmt.Sprintf("rId%d", i+2))
	}
	data.ThemeRelID = fmt.Sprintf("rId%d", len(deck.Slides)+2)

	parts := make([]part, 0, 10+2*len(deck.Slides))
	for _, tpl := range []struct {
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
	} {
		rendered, err := renderTemplate(tpl.name, tpl.text, data)
		if err != nil {
			return nil, err
		}
		parts = append(parts, part{name: tpl.name, data: rendered})
	}

	for i, slide := range deck.Slides {
		shapes, err := shapesXML(deck, slide, g)
		if err != nil {
			return nil, err
		}
		rendered, err := renderTemplate(slidePart(i+1), slideTemplate, struct {
			*soml.Slide
			Shapes string
		}{slide, shapes})
		if err != nil {
			return nil, err
		}
		parts = append(parts,
			part{name: slidePart(i + 1), data: rendered},
			part{name: slideRelsPart(i + 1), data: []byte(slideRelsXML)},
		)
	}
	return parts, nil
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
  <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>
  <Override PartName="/ppt/slideMasters/slideMaster1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideMaster+xml"/>
  <Override PartName="/ppt/slideLayouts/slideLayout1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideLayout+xml"/>
  <Override PartName="/ppt/theme/theme1.xml" ContentType="application/vnd.openxmlformats-officedocument.theme+xml"/>
{{- range $i, $s := .Deck.Slides}}
  <Override PartName="/ppt/slides/slide{{inc $i}}.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>
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
  <p:sldIdLst>
{{- range $i, $rid := .SlideRelIDs}}
    <p:sldId id="{{slideID $i}}" r:id="{{$rid}}"/>
{{- end}}
  </p:sldIdLst>
  <p:sldSz cx="{{.SlideCX}}" cy="{{.SlideCY}}"/>
  <p:notesSz cx="{{.NotesCX}}" cy="{{.NotesCY}}"/>
</p:presentation>
`

const presRelsTemplate = xmlDecl + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="slideMasters/slideMaster1.xml"/>
{{- range $i, $rid := .SlideRelIDs}}
  <Relationship Id="{{$rid}}" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide{{inc $i}}.xml"/>
{{- end}}
  <Relationship Id="{{.ThemeRelID}}" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="theme/theme1.xml"/>
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
    </p:spTree>
  </p:cSld>
  <p:clrMap bg1="lt1" tx1="dk1" bg2="lt2" tx2="dk2" accent1="accent1" accent2="accent2" accent3="accent3" accent4="accent4" accent5="accent5" accent6="accent6" hlink="hlink" folHlink="folHlink"/>
  <p:sldLayoutIdLst>
    <p:sldLayoutId id="2147483649" r:id="rId1"/>
  </p:sldLayoutIdLst>
</p:sldMaster>
`

const masterRelsTemplate = xmlDecl + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="../theme/theme1.xml"/>
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

const slideRelsXML = xmlDecl + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/>
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
