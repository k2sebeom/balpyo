package core

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	// PNG·JPEG 디코더를 **등록만** 한다 (image.DecodeConfig가 포맷을 판정할 수 있게).
	// 여는 어휘가 2종이라는 결정(soml.ImageFormat)이 이 두 줄로도 강제된다: gif를 등록하지
	// 않으므로 `.png` 이름의 gif는 "내용이 png가 아니다"로 걸린다.
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"path"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **에셋 패스**다 — `<image src>`가 가리킨 파일을 읽어 덱의 에셋 표를 채운다
// (soml.Assets).
//
// 값 층의 나머지(Resolve)와 갈라지는 이유는 하나다: **파일 시스템이 필요하다.** 그래서 진입점이
// core.Resolve가 아니라 `fs.FS`를 받는 별도 함수이고, `os`가 아니라 `fs.FS`인 이유는 테스트다 —
// 없는 파일·디렉토리·잘린 헤더·확장자 불일치를 fstest.MapFS로 디스크 없이 전부 만든다.
// pipeline.LoadDeck이 이미 `os.DirFS(projectDir)`로 슬라이드를 수집하므로 **같은 fs.FS**를
// 넘긴다 (경로의 기준이 하나라는 사실이 그 한 값으로 보인다).
//
// 이 패스가 core.Validate의 형제인 것도 같은 이유다: Validate는 파일을 넘어야 알 수 있는 것을
// 보고(슬라이드 ID 충돌), 여기는 **디스크를 봐야 알 수 있는 것**을 본다. 둘 다 Resolve 다음에
// 온다.

// ResolveAssets는 덱이 참조하는 이미지 파일을 읽어 deck.Assets를 채운다 — **core.Resolve
// 다음에** 부른다 (확정 값의 `src`를 읽는다).
//
// 표의 순서는 **첫 사용의 문서 순서**다: 슬라이드 순서 → 슬라이드 안 문서 순서. 맵 순회로
// 순서가 흔들리면 PPTX 미디어 파트 번호가 실행마다 달라지고 골든이 매번 깨진다.
//
// 같은 키를 두 번 만나면 두 번째는 **아무 일도 하지 않는다** — 표 항목이 하나여야 산출물에 같은
// 바이트가 두 번 들어가지 않고, 실패한 키도 같은 규칙을 지나므로 저자가 같은 오타를 슬라이드
// 수만큼 읽지 않는다.
//
// 표를 매번 처음부터 만든다 (append가 아니라 초기화): 두 번 불러도 같은 표가 나와야 결과가
// 호출 횟수에 의존하지 않는다.
func ResolveAssets(deck *soml.Deck, fsys fs.FS) soml.Diagnostics {
	if deck == nil {
		return nil
	}
	deck.Assets = nil
	pass := assetPass{deck: deck, fsys: fsys, seen: map[string]bool{}, checked: map[string]bool{}}
	for _, slide := range deck.Slides {
		pass.file = slide.Source
		// 슬라이드의 칠 순서대로 모은다 — 배경(맨 아래) · 내용 · 크롬. 표의 순서가 PPTX 미디어 파트 번호이므로
		// 순서가 결정적이면 어느 순서든 유효하지만, 칠 순서가 패키지를 풀어 본 사람에게 읽히는 순서다.
		if slide.Background != nil && slide.Background.Image != "" {
			pass.theme(slide.Background.Image)
		}
		for _, node := range slide.Content {
			pass.node(node)
		}
		for _, item := range slide.Chrome {
			if item.Image != "" {
				pass.theme(item.Image)
			}
		}
	}
	return pass.diags
}

// theme은 테마 자산 하나를 덱의 표로 옮긴다 — 바이트는 테마 로딩이 이미 읽고 판정했다
// (soml.Theme.Assets). 디스크를 다시 읽지 않는다: 내장 테마의 자산은 프로젝트 파일 시스템에 없다.
func (p *assetPass) theme(key string) {
	if p.seen[key] {
		return
	}
	asset, ok := p.deck.Theme.Assets.Get(key)
	if !ok {
		p.errorf(soml.Span{}, "theme asset %q is not in the theme asset table — this is an engine bug (theme loading and the asset pass disagree)", key)
		return
	}
	p.seen[key] = true
	p.deck.Assets = append(p.deck.Assets, asset)
}

// assetPass는 에셋 수집 상태다. 진단을 소스 순서로 정렬하지 않는 이유는 순회가 이미 문서
// 순서이기 때문이다 (Resolve는 역할을 먼저 해석해 발견 순서가 어긋나므로 정렬이 필요하다).
type assetPass struct {
	deck *soml.Deck
	fsys fs.FS
	// file은 지금 훑는 슬라이드 파일이다 — 진단이 가리킬 곳.
	file string
	// seen은 표에 실린 키다. checked는 프로젝트 파일 시스템에서 이미 확인한 경로다 — 둘이 갈리는 이유는
	// 테마 자산 키와 모양이 같은 프로젝트 경로(`themes/<이름>/assets/…`)다: 그 키가 테마 자산으로 먼저
	// 실렸어도 프로젝트 경로는 프로젝트에서 찾아야 결과가 슬라이드의 순서에 의존하지 않는다.
	seen    map[string]bool
	checked map[string]bool
	diags   soml.Diagnostics
}

func (p *assetPass) errorf(span soml.Span, format string, args ...any) {
	p.diags = append(p.diags, soml.Errorf(p.file, span, format, args...))
}

// node는 콘텐츠 트리를 훑어 그림을 찾는다. 종류로 갈라지는 이유는 카탈로그가 한 곳이라는
// 계약이다 (soml.KindOf) — 이름 비교를 층마다 들면 태그가 하나 늘 때 어느 층이 빠졌는지 사람이
// 세어야 한다.
func (p *assetPass) node(node soml.Node) {
	elem, ok := node.(*soml.Element)
	if !ok {
		return
	}
	if soml.KindOf(elem.Name) == soml.KindImage {
		p.image(elem)
		// 그림은 void다 — 자식이 없다.
		return
	}
	for _, child := range elem.Children {
		p.node(child)
	}
}

// image는 그림 하나의 에셋을 표에 넣는다.
func (p *assetPass) image(elem *soml.Element) {
	style := elem.ImageStyle()
	if style == nil || style.Src == "" {
		// 해석 단계가 이미 진단했다 (없는 src·경로 문법·지원하지 않는 포맷) — 같은 그림에 두
		// 번째 진단을 얹으면 저자가 고칠 곳이 둘로 보인다.
		return
	}
	attr, ok := elem.Attr("src")
	if !ok {
		// 확정 값에 Src가 있으면 속성이 있다 — 여기 오는 것은 엔진 버그이고, 진단이 가리킬
		// 자리가 없으므로 조용히 넘긴다 (렌더러의 "그릴 것이 없다" 가드가 마지막 그물이다).
		return
	}
	if srcScheme(attr.Value) == themeSrcScheme {
		// 테마의 그림은 값 층이 이미 읽고 판정했다 (themeImageSrc) — 배경 · 로고와 같은 길로 옮긴다.
		p.theme(style.Src)
		return
	}
	if p.checked[style.Src] {
		return
	}
	p.checked[style.Src] = true
	asset, ok := p.load(style.Src, attr.ValueSpan)
	if !ok || p.seen[style.Src] {
		// 표에 이미 있으면 같은 키의 테마 자산이다 (로컬 테마의 파일을 프로젝트 경로로도 가리켰다) — 같은
		// 파일이므로 한 항목이다.
		return
	}
	p.seen[style.Src] = true
	p.deck.Assets = append(p.deck.Assets, asset)
}

// load는 파일 하나를 읽어 에셋으로 만든다 — 실패는 전부 **값 스팬을 가리키는 에러**다.
//
// 헤더만 읽는다 (image.DecodeConfig): 픽셀은 우리가 쓰지 않고, 전체 디코드는 큰 사진에서 빌드
// 시간을 통째로 잡아먹는다. 바이트는 **그대로** 담는다 — 리사이즈·재압축·EXIF 회전은 하지
// 않는다 (엔진이 바이트를 고치면 저자가 커밋한 파일과 산출물의 그림이 다른 것이 되고, 저자는
// 그 차이를 확인할 방법이 없다). EXIF 방향은 읽어서 에셋에 싣기만 한다 (decodeImageAsset).
func (p *assetPass) load(key string, span soml.Span) (soml.Asset, bool) {
	if p.fsys == nil {
		p.errorf(span, "no file system to read assets from (%s) — this is an engine bug (the caller did not pass an fs.FS to ResolveAssets)", key)
		return soml.Asset{}, false
	}
	info, err := fs.Stat(p.fsys, key)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			p.errorf(span, "the file src points to does not exist (%s) — paths are relative to the project root (not to the slide file)", key)
			return soml.Asset{}, false
		}
		p.errorf(span, "cannot read %s: %v", key, err)
		return soml.Asset{}, false
	}
	if info.IsDir() {
		p.errorf(span, "src points to a directory (%s) — point it at a single image file", key)
		return soml.Asset{}, false
	}
	data, err := fs.ReadFile(p.fsys, key)
	if err != nil {
		p.errorf(span, "cannot read %s: %v", key, err)
		return soml.Asset{}, false
	}
	asset, msg := decodeImageAsset(key, data)
	if msg != "" {
		p.errorf(span, "%s", msg)
		return soml.Asset{}, false
	}
	return asset, true
}

// decodeImageAsset은 그림 파일 하나의 바이트를 에셋으로 판정한다 — 실패하면 진단 문구를 돌려준다.
//
// `<image src>`의 파일과 테마 패키지의 그림(배경 · 로고)이 **같은 판정**을 지난다: 확장자는 선언이고
// 내용이 판정이며, 둘이 어긋나면 PowerPoint가 파일을 열지 못한다. 판정이 두 벌이면 한쪽 자산만
// 그 실패를 통과시킨다.
func decodeImageAsset(key string, data []byte) (soml.Asset, string) {
	if len(data) == 0 {
		return soml.Asset{}, fmt.Sprintf("%s is empty (0 bytes) — not an image file (a download or LFS checkout may not have finished)", key)
	}
	// 확장자의 어휘는 해석 단계가 이미 확인했다 (imageSrc) — 여기서 확장자는 **선언**이고
	// 내용이 판정이다.
	declared, known := soml.ImageFormatForExt(path.Ext(key))
	if !known {
		return soml.Asset{}, fmt.Sprintf("the extension of %s is not an image format (available: %s)", key, strings.Join(soml.ImageExts(), ", "))
	}
	if declared == soml.FormatSVG {
		return decodeSVGAsset(key, data)
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		if errors.Is(err, image.ErrFormat) {
			// 확장자만 바꾼 파일이 여기 걸린다. 조용히 통과시키면 산출물은 만들어지고
			// PowerPoint만 열지 못한다 — 조용한 실패의 교과서다.
			return soml.Asset{}, fmt.Sprintf("the content of %s is not %s — a file with only its extension changed builds fine but PowerPoint cannot open it (re-export as png or jpeg)",
				key, declared)
		}
		return soml.Asset{}, fmt.Sprintf("cannot read the image header of %s (the file is truncated) — check the committed bytes", key)
	}
	if soml.ImageFormat(format) != declared {
		return soml.Asset{}, fmt.Sprintf("the extension of %s says %s but the content is %s — [Content_Types] derives the content type from the extension, so PowerPoint cannot open the file as is (change the extension to %q)",
			key, declared, format, soml.ImageFormat(format).Ext())
	}
	if config.Width <= 0 || config.Height <= 0 {
		return soml.Asset{}, fmt.Sprintf("%s has an intrinsic size of %d×%d — nothing to draw", key, config.Width, config.Height)
	}
	width, height := config.Width, config.Height
	var orientation soml.Orientation
	if declared == soml.FormatJPEG {
		// 브라우저는 EXIF 방향을 따라 돌린 그림을 그린다 — 고유 크기도 돌린 그림의 것이어야
		// cover 크롭과 PPTX의 틀이 웹이 잰 박스와 같은 그림을 가리킨다.
		orientation = soml.JPEGOrientation(data)
		if orientation.SwapsAxes() {
			width, height = height, width
		}
	}
	return soml.Asset{
		Key:    key,
		Format: declared,
		// 픽셀이 그대로 논리 스테이지 px다 (수치의 단위는 그것 하나다) — 배율을 정하는 표기는
		// `w`/`h`/`ratio`이고, 엔진이 DPR을 추측하지 않는다 (soml.Asset의 근거).
		Width:       width,
		Height:      height,
		Orientation: orientation,
		Bytes:       data,
	}, ""
}
