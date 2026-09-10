package core

import (
	"path"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **이미지 속성의 값 해석**이다 (PRD §7.5·§7.6 표의 image 행) — 파서가 이름만 본
// 속성을 확정된 이미지 외관(soml.ImageStyle)으로 눕힌다.
//
// 텍스트(resolveText)·배치(resolveLayout)·도형(resolveShape)에 이어 **네 번째 갈래**이고,
// 구조는 도형의 것을 그대로 쓴다: 인라인 패치를 만들어 테마의 두 단(컴포넌트 기본값·variant)
// 위에 병합한다 (resolveImage). 외관 값 문법은 도형과 **같은 함수**를 부른다 (border·radius·
// shadow·opacity·rotate·flip) — 같은 일을 하는 코드가 두 벌이 되는 것이 이 저장소에서 가장
// 비싼 실패이고, 그 갈라짐은 두 산출물을 나란히 볼 때까지 드러나지 않는다.
//
// 새로운 것은 **`src`의 값이 파일을 가리킨다**는 사실 하나다. 그래서 값 층이 둘로 나뉘었다:
//
//   - 여기(Resolve): 경로 **문법**과 어휘 — 절대 경로·`..` 탈출·URL·확장자. 디스크를 보지
//     않으므로 렌더 전 해석이 여전히 순수하고, 파서와 같은 정밀도의 line:col을 갖는다.
//   - asset.go(ResolveAssets): 파일이 **실제로** 무엇인지 — 존재·디렉토리 여부·헤더·고유 크기.
//     파일 시스템을 인자로 받는 별도 진입점이다 (그 근거는 그 파일의 주석에 있다).
//
// 나누는 선을 "디스크를 보는가"에 둔 이유: 문법 오류를 파일 접근 없이 전부 알 수 있어야
// 에디터(lsp)가 프로젝트를 열지 않고도 같은 진단을 낸다.

// resolveImage는 `<image>`의 외관을 4단 캐스케이드로 확정한다 (PRD §8.3):
//
//	인라인 속성 > variant > 컴포넌트 기본값 > (역할 없음)
//
// 도형과 같은 모양이고 갈라지는 자리는 **바닥의 값**이다: 이미지의 컴포넌트 기본값은 비어
// 있다 (imageComponentDefault — 그림은 자기 픽셀을 갖고 있어 장식이 없어도 완전하다).
//
// `src`를 먼저 해석하는 이유는 순서가 아니라 입력이다 — 실패한 `src`가 빈 문자열로 남아야
// 에셋 패스가 같은 그림에 두 번째 진단을 얹지 않는다. `fit`·`alt`와 박스는 캐스케이드 밖이라
// style에 직접 채워진다 (imagePatch의 근거).
func (r *resolver) resolveImage(elem *soml.Element) {
	style := soml.ImageStyle{Src: r.imageSrc(elem), Fit: soml.DefaultImageFit}
	inline := r.imageAttrs(elem, &style)
	mergeImage(&style, r.comps.imageDefault)
	mergeImage(&style, r.imageVariant(elem))
	mergeImage(&style, inline)
	elemAppearance(elem).Image = &style
}

// imageVariant는 `variant`를 테마의 **이미지** variant 표에서 찾는다 — 모르는 이름은 에러가
// 아니라 경고 + 빈 패치다 (PRD §13-6, 텍스트·도형과 같은 정책이고 문구도 같은 곳에서 나온다 —
// warnUnknownVariant). 나열하는 어휘가 이미지의 것이어야 저자가 이 테마에 있는 이름으로 간다.
func (r *resolver) imageVariant(elem *soml.Element) imagePatch {
	attr, ok := r.variantAttr(elem)
	if !ok {
		return imagePatch{}
	}
	patch, known := r.imageVariantPatch(attr.Value)
	if !known {
		r.warnUnknownVariant(attr, r.imageVariantNames())
		return imagePatch{}
	}
	return patch
}

// imageVariantPatch·imageVariantNames는 이미지 variant 단을 로드된 표(r.comps)에서 읽는다 —
// 표가 셋(텍스트·도형·이미지)이라는 계약은 그대로다.
func (r *resolver) imageVariantPatch(name string) (imagePatch, bool) {
	for _, v := range r.comps.imageVariants {
		if v.name == name {
			return v.patch, true
		}
	}
	return imagePatch{}, false
}

func (r *resolver) imageVariantNames() []string {
	names := make([]string, len(r.comps.imageVariants))
	for i, v := range r.comps.imageVariants {
		names[i] = v.name
	}
	return names
}

// imageAttrs는 속성 한 벌을 훑어 **인라인 외관 패치**를 만든다 (shapeAttrs의 이미지 판).
//
// 값 어휘 검사와 진단은 속성별 해석 함수가 그대로 소유한다 — 틀린 값은 패치에 실리지 않으므로
// 아래 단이 남는다 (틀린 fit이 언어의 기본값 contain을 남기는 것).
func (r *resolver) imageAttrs(elem *soml.Element, style *soml.ImageStyle) imagePatch {
	var patch imagePatch
	for _, attr := range elem.Attrs {
		switch attr.Name {
		case "src":
			// 위에서 해석했다 — 여기서 다시 보면 진단이 두 번 난다.
		case "fit":
			if fit, ok := r.imageFit(attr); ok {
				style.Fit = fit
			}
		case "alt":
			if alt, ok := r.imageAlt(attr); ok {
				style.Alt = alt
			}
		case "variant":
			// 캐스케이드 둘째 단이라 imageVariant가 소유한다 — 여기서 또 보면 경고가 두 번 난다.
		case "border":
			if border, ok := r.border(attr); ok {
				patch.border = &border
			}
		case "radius":
			// **프리셋 제약이 없다**: ADR 0019(`radius`는 `rounded-rect` 전용)의 근거는 프리셋
			// 기하에 반경 조정값 자리가 없다는 것이었고, 그림은 프리셋이 아니라 자기 클립을
			// 갖는다 (웹 `border-radius`, PPTX `pic/spPr/prstGeom prst="roundRect"`). 값 문법은
			// 도형과 같은 함수다 (radiusValue) — 갈라지면 같은 표기가 자리에 따라 다르게 읽힌다.
			if r.hasValue(attr) {
				if radius, ok := r.radiusValue(attr); ok {
					patch.radius = &radius
				}
			}
		case "shadow":
			if shadow, ok := r.shadow(attr); ok {
				patch.shadow = &shadow
			}
		case "opacity":
			if opacity, ok := r.opacity(attr); ok {
				patch.opacity = &opacity
			}
		case "rotate":
			if rotate, ok := r.rotate(attr); ok {
				patch.rotate = &rotate
			}
		case "flip":
			if flip, ok := r.flip(attr); ok {
				patch.flip = &flip
			}
		default:
			// 박스 속성(`pad`·크기·`ratio`)은 컨테이너·도형과 공유하는 헬퍼가 해석하고, 자식
			// 배치 속성은 resolveLayout이 이미 지났다. 그 둘도 아니면 카탈로그와 값 층이
			// 어긋난 것이다 — 조용히 지나가면 저자가 쓴 것이 사라진다.
			if r.boxAttr(&style.Box, attr) ||
				containsName(soml.ChildPlacementAttrs(), attr.Name) {
				continue
			}
			r.catalogGap(attr)
		}
	}
	return patch
}

// imageSrc는 `src`를 해석한다 — 값은 **프로젝트 루트 기준 상대 경로**이고 결과는 에셋 표의
// 정규화 키다 (실패하면 빈 문자열).
//
// **기준이 프로젝트 루트인 근거**: 하위 디렉토리가 챕터가 되므로(PRD §6.1) 슬라이드 파일 기준
// 경로를 쓰면 슬라이드를 챕터로 옮길 때 같은 그림의 경로가 파일마다 달라진다 — 그리고 슬라이드
// 이동은 이 제품이 값싸다고 약속한 조작이다. 대가는 하나뿐이다: 챕터 안 슬라이드가 `assets/`를
// 가리킬 때 경로가 길어지지 않는다 (오히려 짧아진다).
//
// `assets/` 아래일 것은 **강제하지 않는다** — PRD §6.1의 디렉토리 구조는 관례이고 프로젝트 루트
// 안이면 충분하다. 강제하면 테마 에셋·챕터별 그림의 자리를 지금 정해야 한다.
//
// **속성이 없으면 에러다** (`type`과 같은 결): 그림의 경로는 추측할 수 없고, 빈 그림을 그리는
// 것은 저자의 오타를 산출물에 통과시키는 일이다.
func (r *resolver) imageSrc(elem *soml.Element) string {
	attr, ok := elem.Attr("src")
	if !ok {
		r.errorf(elem.NameSpan, "<%s>에 src가 필요하다 (src=%q) — 그림의 경로는 추측할 수 없다",
			soml.ImageElement, attrExample("src"))
		return ""
	}
	// hasValue를 그대로 쓰지 않는 이유는 빈 값의 진단 문구다: "지우면 기본값을 쓴다"가 여기서는
	// 틀린 안내다 (src에는 기본값이 없고, 지우면 위의 에러가 난다).
	if !attr.HasValue {
		r.errorf(attr.NameSpan, "src 속성에는 값이 필요하다 (src=%q)", attrExample("src"))
		return ""
	}
	if attr.Value == "" {
		r.errorf(attr.ValueSpan, "src가 비어 있다 — 프로젝트 루트 기준 경로를 적는다 (예: %q)",
			attrExample("src"))
		return ""
	}
	raw := attr.Value
	if strings.ContainsRune(raw, '\\') {
		// 역슬래시를 조용히 `/`로 바꾸지 않는다: 파일 이름에 역슬래시를 쓸 수 있는 파일
		// 시스템이 있어 변환이 다른 파일을 가리킬 수 있고, 그 차이는 CI에서만 드러난다.
		r.errorf(attr.ValueSpan, "src의 경로 구분자는 %q다 (%q에 역슬래시가 있다) — 같은 덱이 기계마다 다르게 빌드되지 않게 한다",
			"/", raw)
		return ""
	}
	if scheme := srcScheme(raw); scheme != "" {
		if scheme == "data" {
			r.errorf(attr.ValueSpan, "src에 data: URL은 쓸 수 없다 — 그림은 파일로 커밋한다 (원문에 바이트를 실으면 슬라이드를 사람이 읽을 수도, 진단이 자리를 가리킬 수도 없다)")
			return ""
		}
		r.errorf(attr.ValueSpan, "src에 URL은 쓸 수 없다 (%q) — 빌드가 네트워크에 의존하면 같은 덱이 CI에서 다르게 빌드되고 오프라인 발표 약속이 깨진다 (PRD §9.1). 파일을 프로젝트 안에 두고 상대 경로로 가리킨다",
			raw)
		return ""
	}
	if strings.HasPrefix(raw, "/") {
		r.errorf(attr.ValueSpan, "src는 프로젝트 루트 기준 상대 경로다 (%q는 절대 경로) — 절대 경로는 빌드를 이 기계에 묶는다 (CI에서 같은 덱이 다르게 빌드된다)",
			raw)
		return ""
	}
	// 정규화가 표의 키를 만든다 — `./assets/a.png`와 `assets/a.png`가 두 항목이 되면 산출물에
	// 같은 바이트가 두 번 들어간다 (PPTX 미디어 파트가 둘이 된다).
	key := path.Clean(raw)
	if key == ".." || strings.HasPrefix(key, "../") {
		r.errorf(attr.ValueSpan, "src가 프로젝트 밖을 가리킨다 (%q) — 에셋은 프로젝트 안에 둔다 (밖을 가리키면 빌드가 이 기계에 의존한다)",
			raw)
		return ""
	}
	if key == "." || key == "/" {
		r.errorf(attr.ValueSpan, "src가 파일을 가리키지 않는다 (%q)", raw)
		return ""
	}
	ext := path.Ext(key)
	if ext == "" {
		r.errorf(attr.ValueSpan, "src에 확장자가 없다 (%q) — 확장자가 PPTX의 콘텐츠 타입을 정한다 (쓸 수 있는 것: %s)",
			raw, imageExtList())
		return ""
	}
	if _, known := soml.ImageFormatForExt(ext); !known {
		// 진단은 **이유와 대안**을 말한다: 어휘를 2종으로 닫은 것은 양 타깃이 그리는 것만
		// 연다는 규칙의 결과이고(soml.ImageFormat), 저자에게는 늘 고칠 방법이 있다.
		r.errorf(attr.ValueSpan, "src의 %q는 v1이 그리지 못한다 (쓸 수 있는 것: %s) — %s",
			strings.ToLower(ext), imageExtList(), imageExtReason(ext))
		return ""
	}
	return key
}

// srcScheme은 값 앞의 URL 스킴이다 (`http`, `https`, `data`). 없으면 빈 문자열이다.
//
// URL 판정을 `strings.Contains(raw, "://")`로 하지 않는 이유는 `data:`다 — 슬래시가 없는
// 스킴을 놓치면 그 값이 파일 이름으로 취급되어 "파일이 없다"로 진단되고, 저자는 왜 자기가 적은
// 데이터가 파일로 읽히는지 알 수 없다.
func srcScheme(raw string) string {
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
			continue
		case c >= '0' && c <= '9', c == '+', c == '-', c == '.':
			if i > 0 {
				continue
			}
			return ""
		case c == ':':
			if i == 0 {
				return ""
			}
			return strings.ToLower(raw[:i])
		}
		return ""
	}
	return ""
}

// imageExtReason은 지원하지 않는 확장자를 **왜** 열지 않았는지다. 문구가 진단 안에 있는 이유는
// 저자가 "왜 안 되는가"를 알면 대안을 스스로 고르기 때문이다 (svg를 png로 내보낼지, 아이콘을
// 기다릴지).
func imageExtReason(ext string) string {
	switch strings.ToLower(ext) {
	case ".svg":
		return "PPTX는 벡터를 확장 요소(svgBlip)와 래스터 폴백으로 실어 소비자마다 다르게 열린다 — png나 jpeg로 내보낸다 (벡터 그래픽은 <icon>의 자리다)"
	case ".gif", ".apng":
		return "애니메이션에는 양 타깃의 대응이 없다 — png나 jpeg로 내보낸다"
	case ".webp", ".avif", ".heic":
		return "PowerPoint 버전마다 지원이 갈려 열리지 않는 기계가 생긴다 — png나 jpeg로 내보낸다"
	case ".bmp", ".tif", ".tiff":
		return "웹 브라우저의 지원이 갈린다 — png나 jpeg로 내보낸다"
	}
	return "png나 jpeg로 내보낸다"
}

// imageFit은 `fit`을 해석한다 — 어휘 2종이다 (soml.ImageFits).
//
// CSS `object-fit`의 나머지 값(`fill`·`none`·`scale-down`)에 **대안을 말한다**: 어휘를 둘로
// 닫은 근거가 "양 타깃에 대응이 있는 것만"이므로(soml.ImageFit), 저자가 CSS 이름을 쓴 것은
// 오타가 아니라 다른 언어의 지식이다 — 그 지식에서 이 언어의 표기로 건너오는 길을 진단이 준다.
func (r *resolver) imageFit(attr soml.Attr) (soml.ImageFit, bool) {
	if !r.hasValue(attr) {
		return "", false
	}
	for _, fit := range soml.ImageFits() {
		if string(fit) == attr.Value {
			return fit, true
		}
	}
	if cssOnlyFits[attr.Value] {
		r.errorf(attr.ValueSpan, "CSS object-fit의 %q는 PPTX에 대응이 없다 (쓸 수 있는 값: %s) — 비율을 무시하고 늘리는 대신 ratio로 박스를 그림 비율에 맞춘다",
			attr.Value, imageFitList())
		return "", false
	}
	r.errorf(attr.ValueSpan, "알 수 없는 fit %q (쓸 수 있는 값: %s)", attr.Value, imageFitList())
	return "", false
}

// cssOnlyFits는 CSS에만 있는 `object-fit` 값이다 — 저자가 아는 이름이지만 PPTX에 자리가 없다
// (`none`은 `pic`의 박스 밖으로 그림이 넘치는 상태라 대응이 아예 없다).
var cssOnlyFits = map[string]bool{"fill": true, "none": true, "scale-down": true}

// imageAlt은 `alt`을 해석한다 — 값 검사가 아니라 **빈 값의 뜻**이 결정이다.
//
// `alt=""`를 막지 않는다: `<img>`에서 속성 없음과 빈 문자열은 스크린 리더에 다른 뜻이고, 빈
// 문자열이 "장식 이미지"의 표기다. 생략과 `alt=""`가 같은 확정 값이 되는 것까지가 이 결정의
// 일부다 — 생략한 그림도 웹에서 `alt=""`로 나가므로(항목 01의 결정) 두 표기가 같은 산출물을
// 낸다. 값 **없는** `alt`은 여전히 에러다: 뜻을 정할 수 없는 형태다.
func (r *resolver) imageAlt(attr soml.Attr) (string, bool) {
	if !attr.HasValue {
		r.errorf(attr.NameSpan, "alt 속성에는 값이 필요하다 (alt=%q) — 장식 그림은 alt=\"\"로 적는다",
			attrExample("alt"))
		return "", false
	}
	return attr.Value, true
}

// 진단에 실을 어휘 목록이다 — 어휘 정의(soml)와 메시지가 갈라지지 않도록 손으로 쓰지 않는다
// (shapeTypeList와 같은 태도).
func imageFitList() string {
	names := make([]string, 0, len(soml.ImageFits()))
	for _, fit := range soml.ImageFits() {
		names = append(names, string(fit))
	}
	return strings.Join(names, ", ")
}

func imageExtList() string { return strings.Join(soml.ImageExts(), ", ") }
