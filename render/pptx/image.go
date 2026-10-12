package pptx

import (
	"fmt"
	"math"
	"strings"

	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **`<image>` → `p:pic` + 미디어 파트 + 관계**다.
//
// # 세 자리가 맞아야 그림이 나온다
//
//	ppt/media/image<N>.<ext>            바이트 (파트가 처음으로 XML이 아니다)
//	[Content_Types].xml의 Default        확장자 → 콘텐츠 타입 (파트가 무엇인지의 선언)
//	ppt/slides/_rels/slideN.xml.rels     관계 (blip@r:embed가 가리키는 목적지)
//
// 셋 중 하나가 어긋나면 파일이 열리지 않거나 **그림 자리가 빈 채로 열린다** — 조용한 실패가
// 가장 많은 자리다. 그래서 셋을 각각 따로 단정한다 (image_test.go의 삼각 대조): 한 곳만 보는
// 테스트는 나머지 두 곳의 어긋남을 통과시킨다.
//
// 관계 id 배정은 하이퍼링크와 **한 카운터**를 나눠 쓴다 (slideRels) — 그 결정의 근거는 그
// 타입의 주석에 있다.
//
// # `fit`은 잰 rect가 이미 답을 담고 있다
//
//	fit        xfrm            blipFill
//	contain    잰 rect 그대로   크롭 없음 — 잰 rect가 **맞춰진 그림**이다 (웹이 max-*로 박스를 그림에 맞췄다)
//	cover      잰 rect 그대로   a:srcRect로 넘치는 쪽을 **양쪽 절반씩** 잘라낸다
//
// 계산이 한 경우뿐인 것은 웹의 결정 덕분이다 (render/web/image.go의 fit 절): 잰 rect가 언제나
// 그려진 그림이므로 letterbox 여백을 이쪽에서 다시 계산할 일이 없다. `contain`에서 `xfrm`이
// 곧 그림이고, `cover`에서 `xfrm`은 박스이며 넘치는 부분만 잘린다 — 그 크롭의 정의가 CSS
// `object-fit: cover`(짧은 축 기준 확대 + 중앙 정렬)와 같아야 두 타깃이 같은 영역을 보여준다
// (coverCrop).
//
// 예외는 EXIF 방향이 가로·세로를 바꾸는 JPEG 하나다: 잰 rect는 브라우저가 돌린 그림이고
// PowerPoint는 저장된 픽셀을 그대로 펴므로, `xfrm`은 같은 중심에서 가로·세로를 바꾼 틀에 회전을
// 얹는다 (orientedFrame · orientedTransform).
//
// # 도형의 함수를 그대로 쓴다
//
// 좌표(Offset·Extent)·회전과 대칭(transformOf)·프리셋 지오메트리(writeGeom)·선(writeLine)·
// 그림자(writeEffect)·색(writeColor)은 **도형과 같은 함수**다. 그림 때문에 두 벌이 되면
// 테두리 색 토큰 하나가 두 타깃·두 요소에서 갈라질 길이 생긴다 (그리고 그 갈라짐은 산출물을
// 나란히 볼 때까지 보이지 않는다). 그림에만 있는 것은 셋뿐이다: `blipFill`(바이트 참조),
// `srcRect`(cover 크롭), `alphaModFix`(불투명도).
//
// # `opacity`는 `a:alphaModFix`다
//
// 도형의 `opacity`는 합성 불투명도가 아니라 색 알파로 컴파일된다. 도형에서 `alphaModFix`를
// 쓰지 않은 이유는 그것이 **도형 안 글자까지** 흐리게 하는데 웹의 `opacity`는 svg에만 걸려
// 글자를 건드리지 않는다는 것이었다 (spSpec.alpha의 주석). 그림에는 담을 글자가 없으므로 그
// 근거가 걸리지 않고, 여기서는 CSS `opacity`와 `alphaModFix`가 **정확한 대응**이다.
// 테두리·그림자 색에도 같은 알파를 싣는다 — CSS `opacity`는 요소 전체(테두리 포함)를 흐리게
// 하고 `filter: drop-shadow`의 결과도 함께 흐려지므로, 그림만 흐려지면 두 타깃이 갈라진다.

// mediaPartDir은 미디어 파트가 사는 디렉토리다 (`ppt/media/`) — PowerPoint의 관례이고, 소비자가
// 패키지를 풀어 보았을 때 그림이 어디 있는지 아는 자리다.
const mediaPartDir = "ppt/media/"

// fallbackExt는 SVG 대체 그림 파트의 확장자다.
const fallbackExt = ".png"

// svgBlipExtURI · svgBlipNS는 Office 2016의 SVG 그림 확장이다 — `a:blip/a:extLst/a:ext@uri`와 `asvg:svgBlip`의 이름공간.
const (
	svgBlipExtURI = "{96DAC541-7B7A-43D3-8B79-37D633B846F1}"
	svgBlipNS     = "http://schemas.microsoft.com/office/drawing/2016/SVG/main"
)

// mediaPart는 N번째(1-based) 미디어 파트 경로다.
//
// **N은 덱 전체의 에셋 표 순서다** (soml.Assets — 첫 사용의 문서 순서로 결정적으로 정해져
// 있다). 슬라이드마다 다시 세면 같은 그림이 여러 파트가 되어 파일이 커지고, 무엇보다
// PowerPoint에서 "이 그림 바꾸기"가 한 슬라이드만 고친다 — 저자는 나머지 슬라이드가 왜
// 따라오지 않는지 알 수 없다. 표가 하나이므로 파트도 하나다.
//
// 확장자는 **포맷**에서 나온다 (soml.ImageFormat.Ext — 저자가 쓴 확장자가 아니다):
// `[Content_Types]`의 Default가 확장자로 콘텐츠 타입을 정하므로, `.JPG`·`.jpeg`가 섞인 덱에서
// 파트 이름을 저자의 표기대로 지으면 Default 항목이 여러 개 필요해지고 대소문자 매칭을
// 소비자의 관용에 맡기게 된다.
func mediaPart(n int, ext string) string {
	return fmt.Sprintf("%simage%d%s", mediaPartDir, n, ext)
}

// mediaTarget은 슬라이드 rels에 적는 미디어 파트의 상대 경로다 (`ppt/slides/`에서 본 자리).
//
// 관계 Target은 관계 파일을 소유한 파트 기준의 상대 경로다 — 절대 경로(`/ppt/media/…`)도
// 유효하지만 PowerPoint가 내는 표기를 따른다 (재저장 diff를 만들지 않는다).
func mediaTarget(n int, ext string) string {
	return fmt.Sprintf("../media/image%d%s", n, ext)
}

// mediaOf는 에셋 표에서 이 키의 자리와 항목을 함께 찾는다 — 번호와 바이트가 **한 조회**에서
// 나와야 파트 이름과 관계 목표가 어긋날 수 없다.
func mediaOf(deck *soml.Deck, key string) (int, soml.Asset, bool) {
	for i, asset := range deck.Assets {
		if asset.Key == key {
			return i + 1, asset, true
		}
	}
	return 0, soml.Asset{}, false
}

// imageRelType·externalTargetMode는 관계 파일의 값이다 (hyperlinkRelType의 형제).
const (
	hyperlinkRelType   = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink"
	imageRelType       = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/image"
	externalTargetMode = "External"
)

// roundRectPreset은 반경이 있는 그림의 프리셋이다 — 도형 어휘의 `rounded-rect`가 가는 곳과
// **같은 프리셋**이지만 그림은 도형 어휘를 거치지 않는다 (rectPreset과 같은 처지다).
// 두 이름이 같은 값인 것을 테스트가 web.ShapePreset과 맞대어 못 박는다.
const roundRectPreset = "roundRect"

// writePicShape는 `<image/>` 하나를 p:pic으로 쓴다.
//
// 좌표는 잰 rect **그대로**다 (모든 객체와 같다). `pad`도 다시 빼지 않는다: 웹의 여백은
// 저자 박스가 흡수하고 잰 rect는 그 안쪽의 **그림**이다 (05/03의 실측: `pad="20"`인 240×120
// 칸의 잰 rect가 200×80이다).
func writePicShape(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, elem *soml.Element, id string, shapeID int, g *geom.Geometry, rels *slideRels) error {
	box, ok := g.Box(id)
	if !ok {
		return fmt.Errorf("%s: %s <%s> has no measured coordinates", slide.Source, id, elem.Name)
	}
	style := elem.ImageStyle()
	if style == nil {
		// 해석되지 않은 외관을 기본값으로 그리면 PPTX만 그럴듯하게 나오고 웹과 갈라진다
		// (render/web.writeImage와 같은 관문).
		return fmt.Errorf("%s:%s <%s> image appearance is unresolved — the deck did not go through core.Resolve",
			slide.Source, elem.Span, elem.Name)
	}
	if style.Src == "" {
		// 값 층이 이미 진단했다 (없는 src·경로 문법·지원하지 않는 포맷 — core/image.go).
		// 그래도 멈추는 이유: 빈 `r:embed`는 **파일이 열리는** 실패다 (뷰어가 관계를 찾지
		// 못해 그림 자리를 비운다).
		return fmt.Errorf("%s:%s <%s> src is unresolved — nothing to draw",
			slide.Source, elem.Span, elem.Name)
	}
	n, asset, ok := mediaOf(deck, style.Src)
	if !ok {
		// 에셋 패스를 거치지 않은 덱이다 (core.ResolveAssets) — 표에 없으면 미디어 파트에
		// 실을 바이트가 없다.
		return fmt.Errorf("%s:%s <%s>: %q is not in the asset table — the deck did not go through core.ResolveAssets",
			slide.Source, elem.Span, elem.Name, style.Src)
	}
	ext := asset.Format.Ext()
	if ext == "" {
		return fmt.Errorf("%s:%s <%s>: unknown file extension for format %q — a format was added without a media part name",
			slide.Source, elem.Span, elem.Name, string(asset.Format))
	}
	// SVG 그림의 blip은 대체 PNG를 가리키고, 원본은 확장 요소가 가리킨다 — 벡터를 모르는 소비자도 그림을 그린다.
	embedExt := ext
	if asset.Format == soml.FormatSVG {
		if _, ok := g.Fallback(asset.Key); !ok {
			return fmt.Errorf("%s:%s <%s>: SVG %q has no raster fallback — the deck was not measured", slide.Source, elem.Span, elem.Name, asset.Key)
		}
		embedExt = fallbackExt
	}
	spec, err := picSpec(shapeID, elem.Name+" "+id, box.Rect, *style, asset, rels.mediaID(mediaTarget(n, embedExt)))
	if err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	if asset.Format == soml.FormatSVG {
		spec.svgEmbed = rels.mediaID(mediaTarget(n, ext))
	}
	spec.ph = slotPlaceholder(slide, elem)
	if err := writePic(b, spec, deck.Theme); err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	return nil
}

// picInfo는 p:pic 하나의 조립 명세다 (spSpec의 형제 — 골격을 함수 하나에 두어야 스키마
// 순서가 한 곳에서만 지켜진다는 근거도 같다).
type picInfo struct {
	id   int
	name string
	// descr은 `cNvPr@descr` = 저자의 `alt`다. 빈 값이면 속성을 내지 않는다 (picSpec의 근거).
	descr string
	// rect는 잰 rect 그대로다 (writePicShape의 주석).
	rect geom.Rect
	// embed는 미디어 관계 id다 (`blip@r:embed`). SVG 그림이면 대체 PNG를 가리킨다.
	embed string
	// svgEmbed는 SVG 원본의 관계 id다 (`asvg:svgBlip@r:embed`) — SVG가 아니면 빈 문자열이다.
	svgEmbed string
	// crop은 cover의 srcRect다 (0이면 요소를 내지 않는다 — contain과 비율이 같은 cover).
	crop srcRect
	// alpha는 불투명도다 (nil이면 alphaModFix 없음). 그림·선·그림자에 함께 실린다.
	alpha *float64
	xfrm
	preset string
	adjust []spAdjust
	border soml.Border
	shadow *soml.ShadowToken
	// ph는 이 그림이 채우는 슬롯 플레이스홀더다 (nil이면 `<p:nvPr/>` — spSpec.ph와 같은 뜻).
	ph *placeholder
}

// srcRect는 `a:srcRect`의 크롭이다 — 원본에서 **잘라내는** 비율이다 (100000 = 100%,
// PercentScale과 같은 눈금). 보이는 영역이 아니라 버리는 몫인 것이 이 요소의 뜻이다.
type srcRect struct {
	l, t, r, b int64
}

// attrs는 0이 아닌 크롭만 속성으로 낸다 (순서는 CT_RelativeRect의 l·t·r·b) — 빈 문자열이면
// 크롭이 없다는 뜻이고 호출자가 요소 자체를 내지 않는다. 0인 변을 적지 않는 이유는 기본값이
// 0이라서다: 골든에 뜻 없는 `l="0"`이 남으면 어느 축이 잘렸는지 눈으로 읽을 수 없다.
func (s srcRect) attrs() string {
	var attrs string
	for _, side := range []struct {
		name string
		val  int64
	}{{"l", s.l}, {"t", s.t}, {"r", s.r}, {"b", s.b}} {
		if side.val != 0 {
			attrs += fmt.Sprintf(` %s="%d"`, side.name, side.val)
		}
	}
	return attrs
}

// picSpec은 확정 이미지 외관을 p:pic 명세로 컴파일한다 (외관 매핑 표의 PPTX 열).
//
// **순서는 매핑 표의 행 순서다** (fit → border → radius → shadow → opacity → rotate → flip) —
// 웹의 imageInkCSS와 나란히 읽을 수 있어야 "표에 있고 한쪽에만 있는 행"이 눈에 띈다.
// `radius`는 여기서 **기하**가 된다 (`roundRect` + 조정값): OOXML에서 반경이 사는 자리가
// 프리셋 지오메트리뿐이고, 웹에서 그것이 `border-radius`인 것과 짝이다. 도형과 달리
// **프리셋 제약이 없다** — 그림의 기하는 사각형과 반경으로 끝나므로 반경은 언제나 실을 자리가
// 있다 (반경을 rounded-rect 프리셋에만 허용하는 제약은 도형의 것이다 — soml.ImageStyle.Radius).
func picSpec(id int, name string, rect geom.Rect, style soml.ImageStyle, asset soml.Asset, embed string) (picInfo, error) {
	if embed == "" {
		return picInfo{}, fmt.Errorf("empty media relationship id")
	}
	// 틀과 크롭은 **저장된 픽셀**의 축으로 센다 — PowerPoint는 EXIF 방향을 보지 않고 저장된 픽셀을
	// 틀에 펴므로, 방향은 아래의 회전·대칭이 옮긴다 (orientedFrame).
	frame, stored := orientedFrame(rect, asset)
	spec := picInfo{id: id, name: name, descr: style.Alt, rect: frame, embed: embed}
	switch style.Fit {
	case soml.FitContain:
		// **계산 없음.** 잰 rect가 이미 맞춰진 그림이다 (파일 머리 주석의 표).
	case soml.FitCover:
		crop, err := coverCrop(stored, frame)
		if err != nil {
			return picInfo{}, err
		}
		spec.crop = crop
	default:
		// 어휘 밖 값은 core가 이미 막았다 (닫힌 어휘). 여기서 멈추는 이유는 빈 문자열이다:
		// 해석되지 않은 `Fit`을 통과시키면 크롭 없는 cover가 나가 그림이 **늘어난다**
		// (stretch/fillRect는 원본을 도형에 맞춰 늘린다 — 웹의 `object-fit: fill`과 같은 상태).
		return picInfo{}, fmt.Errorf("fit %q is not a known fit value", string(style.Fit))
	}
	// 테두리를 정하지 않은 그림(nil)은 **선 없음**이다 — 도형과 갈라지는 자리다: 도형의
	// 확정 외관에는 테마 컴포넌트 기본값이 언제나 채워져 있지만(테마의 shape.default)
	// 이미지 컴포넌트 기본값은 비어 있다 (그림은 자기 픽셀을 갖고, 기본으로 테두리를 두르면
	// 스크린샷마다 이중 경계가 생긴다). 그래도 `a:ln`을 **명시한다**:
	// 비워 두면 테마 fmtScheme의 기본 선이 상속되어 웹에 없는 외곽선이 그려질 수 있고,
	// `border="none"`과 "정하지 않음"이 같은 그림(선 없음)이므로 산출도 같아야 한다.
	spec.border = soml.NoBorder()
	if style.Border != nil {
		spec.border = *style.Border
	}
	spec.preset = rectPreset
	if style.Radius != nil {
		spec.preset = roundRectPreset
		adjust := web.RoundRectAdjust()
		val, err := AdjustValue(adjust, frame, style.Radius.Px)
		if err != nil {
			return picInfo{}, err
		}
		spec.adjust = []spAdjust{{name: adjust.Name, val: val}}
	}
	spec.shadow = style.Shadow
	if style.Opacity != nil {
		if *style.Opacity < 0 || *style.Opacity > 1 {
			// 웹과 같은 관문이다 (imageInkCSS) — 범위 밖 알파는 뷰어가 조용히 자른다.
			return picInfo{}, fmt.Errorf("opacity %v is outside 0–1", *style.Opacity)
		}
		spec.alpha = style.Opacity
	}
	transform, err := orientedTransform(style.Rotate, style.Flip, asset.Orientation)
	if err != nil {
		return picInfo{}, err
	}
	spec.xfrm = transform
	return spec, nil
}

// orientedFrame은 EXIF 방향을 PPTX로 옮길 때의 틀이다 — 잰 rect(보이는 그림)와 저장된 픽셀
// 크기의 에셋을 돌려준다.
//
// 가로·세로를 바꾸는 방향(90°·270°)이면 틀은 잰 rect와 **중심이 같고** 가로·세로가 바뀐 사각형이다:
// `xfrm@rot`은 틀의 중심을 돌리므로, 저장된 픽셀을 그 틀에 편 뒤 90° 돌리면 잰 rect를 정확히 덮는다.
// 바꾸지 않는 방향이면 틀이 곧 잰 rect다.
func orientedFrame(rect geom.Rect, asset soml.Asset) (geom.Rect, soml.Asset) {
	if !asset.Orientation.SwapsAxes() {
		return rect, asset
	}
	stored := asset
	stored.Width, stored.Height = asset.Height, asset.Width
	cx, cy := rect.X+rect.W/2, rect.Y+rect.H/2
	return geom.Rect{X: cx - rect.H/2, Y: cy - rect.W/2, W: rect.H, H: rect.W}, stored
}

// orientedTransform은 저자의 회전·대칭에 EXIF 방향을 합성한다.
//
// 보이는 그림은 "저장된 픽셀 → EXIF 방향(대칭 → 회전 φ) → 저자의 대칭 → 저자의 회전 θ"다 (웹에서
// 저자의 변형은 브라우저가 이미 돌린 그림에 걸린다). `xfrm`이 적을 수 있는 것은 "대칭 → 회전" 한
// 쌍뿐이므로, 저자의 대칭을 EXIF 회전 앞으로 옮긴다: 축 하나의 대칭은 회전의 방향을 뒤집고
// (F·R(φ) = R(−φ)·F), 두 축 대칭은 180° 회전이라 그대로 지나간다. 대칭끼리는 순서와 무관하게
// 축마다 겹친다 (같은 축의 두 번은 대칭 없음).
func orientedTransform(rotate *float64, flip soml.Flip, o soml.Orientation) (xfrm, error) {
	t, err := transformOf(rotate, flip)
	if err != nil {
		return xfrm{}, err
	}
	phi := o.Rotation()
	if phi == 0 && !o.FlipH() {
		return t, nil
	}
	if t.flipH != t.flipV {
		phi = -phi
	}
	deg := float64(phi)
	if rotate != nil {
		deg += *rotate
	}
	if rot := Rotation(deg); rot != 0 || rotate != nil {
		t.rot = &rot
	} else {
		t.rot = nil
	}
	t.flipH = t.flipH != o.FlipH()
	return t, nil
}

// coverCrop은 `fit="cover"`의 `a:srcRect` 크롭이다 — CSS `object-fit: cover`와 같은 영역을
// 보여주기 위한 계산이고, 이 파일에서 **유일한 기하 계산**이다.
//
// 입력은 둘뿐이다: 그림의 고유 종횡비와 잰 박스의 종횡비. cover는 짧은 축을 채우도록 확대해
// 긴 축이 넘치므로, 넘치는 축의 **양쪽을 절반씩** 잘라내면 중앙 정렬이 된다 (CSS의 기본
// `object-position: 50% 50%`).
//
//	그림이 박스보다 넓다 (imgRatio > boxRatio) → 좌우를 자른다: l = r = (1 − boxRatio/imgRatio) / 2
//	그림이 박스보다 좁다 (imgRatio < boxRatio) → 위아래를 자른다: t = b = (1 − imgRatio/boxRatio) / 2
//	비율이 같다                                 → 크롭 0 (srcRect 요소를 내지 않는다)
//
// **반올림과 잔차**: 1000분의 %는 픽셀보다 거칠다. 양변을 각각 반올림하므로 보이는 몫은
// `100000 − 2·crop`이고 정확한 값과 최대 1단위(0.001%) 어긋난다 — 1280px 폭에서 0.013px이라
// 픽셀 비교에 보이지 않는다. **양변을 같은 값으로 두는 것**을 잔차 배분보다 앞세웠다: 한 변에
// 잔차를 몰면 크롭이 비대칭이 되어 그림이 반 픽셀 밀리고, 그것은 "cover는 중앙을 보여준다"는
// 계약이 눈에 보이지 않게 깨지는 쪽이다.
//
// 크기 0 박스는 크롭 0이다 (그릴 것이 없다 — AdjustValue가 `ss = 0`을 다루는 것과 같은 태도).
func coverCrop(asset soml.Asset, rect geom.Rect) (srcRect, error) {
	if asset.Width <= 0 || asset.Height <= 0 {
		// 에셋 패스가 이미 막았다 (고유 크기 0은 진단이다 — core/asset.go). 여기서 멈추는
		// 이유는 0으로 나눈 NaN이 `srcRect`에 실리면 파일이 열리지 않기 때문이다.
		return srcRect{}, fmt.Errorf("intrinsic size is %d×%d — cannot compute the crop", asset.Width, asset.Height)
	}
	if rect.W <= 0 || rect.H <= 0 {
		return srcRect{}, nil
	}
	imgRatio := float64(asset.Width) / float64(asset.Height)
	boxRatio := rect.W / rect.H
	half := func(visible float64) int64 {
		return int64(math.Round((1 - visible) / 2 * PercentScale))
	}
	switch {
	case imgRatio > boxRatio:
		crop := half(boxRatio / imgRatio)
		return srcRect{l: crop, r: crop}, nil
	case imgRatio < boxRatio:
		crop := half(imgRatio / boxRatio)
		return srcRect{t: crop, b: crop}, nil
	}
	return srcRect{}, nil
}

// writePic은 그림 하나의 골격을 쓴다 — nvPicPr(id·이름·대체 텍스트) → blipFill(바이트 참조·
// 크롭) → spPr(좌표·변형·기하·선·효과).
//
// 자식 순서는 CT_Picture와 CT_ShapeProperties가 정한 그대로다. 어기면 LibreOffice가 복구를
// 묻는다 (writeSp와 같은 종류의 제약).
//
// **spPr에 채움을 내지 않는다** — 도형과 갈라지는 자리다. 도형은 `noFill`을 명시해야 테마
// fmtScheme의 기본 채움이 상속되지 않지만(writeFill), 그림의 채움은 `p:blipFill`이 이미
// 소유하고 있다: spPr에 채움을 또 적으면 같은 요소의 채움이 두 곳에서 선언되고, 어느 쪽이
// 이기는지는 소비자마다 다르다 (PowerPoint가 내는 p:pic에도 그 자리가 비어 있다).
func writePic(b *strings.Builder, spec picInfo, theme soml.Theme) error {
	x, y := Offset(spec.rect)
	cx, cy := Extent(spec.rect)

	w := func(depth int, line string) {
		b.WriteString("\n" + shapeIndent + strings.Repeat("  ", depth) + line)
	}
	w(0, "<p:pic>")
	w(1, "<p:nvPicPr>")
	// 이름은 도형과 같은 규칙이다 (지오메트리 식별자 — writeSp): PowerPoint 선택 창과 진단
	// 메시지가 같은 이름으로 같은 요소를 가리킨다. `descr`은 **저자의 `alt`**이고 그것이
	// PowerPoint의 "대체 텍스트" 칸이다 — 빈 값이면 속성을 내지 않는다 (cNvPrXML).
	w(2, cNvPrXML(spec.id, spec.name, spec.descr))
	// `a:picLocks noChangeAspect`를 넣지 않는다: 산출물은 편집 가능해야 하고, 저자가 아니라
	// 엔진이 잠근 제약은 파일을 열어 본 사람이 이유를 알 수 없다 (종횡비를 지키며 늘리는 것은
	// PowerPoint에서 코너 핸들 + Shift로 이미 되고, 잠그면 그것 말고는 아예 되지 않는다).
	w(2, "<p:cNvPicPr/>")
	writeNvPr(w, 2, spec.ph)
	w(1, "</p:nvPicPr>")
	if spec.embed == "" {
		return fmt.Errorf("empty media relationship id")
	}
	w(1, "<p:blipFill>")
	if spec.alpha != nil || spec.svgEmbed != "" {
		w(2, fmt.Sprintf(`<a:blip r:embed="%s">`, spec.embed))
		if spec.alpha != nil {
			w(3, fmt.Sprintf(`<a:alphaModFix amt="%d"/>`, Alpha(*spec.alpha)))
		}
		if spec.svgEmbed != "" {
			// 스키마 순서(CT_Blip): 효과들 → extLst. svgBlip은 Office 2016의 확장이다 — 아는 소비자는 벡터를,
			// 모르는 소비자는 blip의 PNG를 그린다.
			w(3, "<a:extLst>")
			w(4, `<a:ext uri="`+svgBlipExtURI+`">`)
			w(5, fmt.Sprintf(`<asvg:svgBlip xmlns:asvg="%s" r:embed="%s"/>`, svgBlipNS, spec.svgEmbed))
			w(4, "</a:ext>")
			w(3, "</a:extLst>")
		}
		w(2, "</a:blip>")
	} else {
		w(2, fmt.Sprintf(`<a:blip r:embed="%s"/>`, spec.embed))
	}
	// 스키마 순서(CT_BlipFillProperties): blip → srcRect → tile|stretch.
	if attrs := spec.crop.attrs(); attrs != "" {
		w(2, "<a:srcRect"+attrs+"/>")
	}
	// stretch/fillRect는 "잘라낸 원본을 도형 전체에 맞춘다"다 — tile(반복)이 아닌 것이
	// 그림의 계약이고, 잰 rect가 그려진 그림이므로 이 늘림은 크기를 바꾸지 않는다
	// (contain은 박스가 곧 맞춰진 그림, cover는 크롭 후 남은 몫이 박스와 같은 비율이다).
	w(2, "<a:stretch>")
	w(3, "<a:fillRect/>")
	w(2, "</a:stretch>")
	w(1, "</p:blipFill>")
	w(1, "<p:spPr>")
	w(2, "<a:xfrm"+spec.attrs()+">")
	w(3, fmt.Sprintf(`<a:off x="%d" y="%d"/>`, x, y))
	w(3, fmt.Sprintf(`<a:ext cx="%d" cy="%d"/>`, cx, cy))
	w(2, "</a:xfrm>")
	writeGeom(w, 2, spec.preset, spec.adjust)
	if err := writeLine(w, 2, spec.border, spec.alpha, theme); err != nil {
		return err
	}
	if err := writeEffect(w, 2, spec.shadow, spec.alpha, theme); err != nil {
		return err
	}
	w(1, "</p:spPr>")
	w(0, "</p:pic>")
	return nil
}

// mediaParts는 덱의 에셋을 미디어 파트 목록으로 만든다 (표 순서 = 파트 번호 순서).
//
// **표 전체를 낸다** — 어느 슬라이드가 무엇을 쓰는지 보지 않는다: 표는 참조에서 만들어지므로
// (core.ResolveAssets) 항목마다 쓰는 슬라이드가 있고, 파트 번호가 표의 자리에서 나오므로
// 슬라이드를 지웠을 때 번호가 밀리지 않아야 관계가 살아 있는 슬라이드들과 어긋나지 않는다.
//
// 바이트는 표의 것을 **그대로** 넘긴다 (복사·재압축 없음 — 엔진은 그림을 고치지 않는다.
// soml.Asset.Bytes의 계약이고 웹의 assetFiles와 같은 태도다).
func mediaParts(deck *soml.Deck, g *geom.Geometry) ([]part, error) {
	if len(deck.Assets) == 0 {
		return nil, nil
	}
	parts := make([]part, 0, len(deck.Assets))
	for i, asset := range deck.Assets {
		ext := asset.Format.Ext()
		if ext == "" {
			return nil, fmt.Errorf("asset %q: format %q has no file extension", asset.Key, string(asset.Format))
		}
		if len(asset.Bytes) == 0 {
			// 빈 파트는 파일을 열 수는 있게 두고 그림만 비운다 — 조용한 실패다.
			return nil, fmt.Errorf("asset %q has no bytes — the deck did not go through the asset pass", asset.Key)
		}
		parts = append(parts, part{name: mediaPart(i+1, ext), data: asset.Bytes})
		if asset.Format == soml.FormatSVG {
			// 대체 PNG는 원본과 같은 번호의 형제 파트다 — 그림 하나가 파트 둘이다.
			png, ok := g.Fallback(asset.Key)
			if !ok {
				return nil, fmt.Errorf("asset %q is an SVG without a raster fallback — the deck was not measured", asset.Key)
			}
			parts = append(parts, part{name: mediaPart(i+1, fallbackExt), data: png})
		}
	}
	return parts, nil
}

// mediaDefault는 `[Content_Types].xml`의 `Default` 항목 하나다 (확장자 → 콘텐츠 타입).
type mediaDefault struct {
	// Ext는 점 없는 확장자다 (`Default@Extension`의 표기).
	Ext         string
	ContentType string
}

// mediaDefaults는 이 덱에 **실제로 든** 확장자의 Default 목록이다 (첫 사용 순서).
//
// 쓰지 않는 선언을 미리 넣지 않는 이유: 골든이 "이 덱에 무엇이 들었는가"를 말해야 한다.
// `rels`·`xml`의 Default가 언제나 있는 것과 갈라지는 점은 그 둘이 모든 패키지에 있다는
// 것이다 — 그림은 없는 덱이 대부분이다.
//
// 포맷당 하나로 모이는 것은 파트 이름을 포맷에서 짓기 때문이다 (mediaPart): 저자가
// `.jpg`와 `.jpeg`를 섞어 써도 패키지 안의 확장자는 한 가지다.
func mediaDefaults(deck *soml.Deck) ([]mediaDefault, error) {
	var defaults []mediaDefault
	seen := map[string]bool{}
	add := func(format soml.ImageFormat, key string) error {
		ext := strings.TrimPrefix(format.Ext(), ".")
		if ext == "" {
			return fmt.Errorf("asset %q: format %q has no file extension", key, string(format))
		}
		if seen[ext] {
			return nil
		}
		contentType, err := mediaContentType(format)
		if err != nil {
			return fmt.Errorf("asset %q: %w", key, err)
		}
		seen[ext] = true
		defaults = append(defaults, mediaDefault{Ext: ext, ContentType: contentType})
		return nil
	}
	for _, asset := range deck.Assets {
		if err := add(asset.Format, asset.Key); err != nil {
			return nil, err
		}
		if asset.Format == soml.FormatSVG {
			// SVG에는 대체 PNG 파트가 따라온다 (mediaParts).
			if err := add(soml.FormatPNG, asset.Key); err != nil {
				return nil, err
			}
		}
	}
	return defaults, nil
}

// mediaContentType은 포맷의 OPC 콘텐츠 타입이다.
//
// 표를 soml에 두지 않은 이유: 콘텐츠 타입은 OOXML 패키지의 개념이고 웹은 쓰지 않는다
// (브라우저는 확장자와 매직 바이트로 판단한다). 포맷 어휘를 늘리고 이 표를 잊으면 **에러**로
// 멈춘다 — 빠진 Default는 "PowerPoint가 파일을 열지 못한다"로만 드러나므로 조용히 넘길 수
// 없는 자리다.
func mediaContentType(format soml.ImageFormat) (string, error) {
	switch format {
	case soml.FormatPNG:
		return "image/png", nil
	case soml.FormatJPEG:
		return "image/jpeg", nil
	case soml.FormatSVG:
		return "image/svg+xml", nil
	}
	return "", fmt.Errorf("unknown content type for format %q", string(format))
}
