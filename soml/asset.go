package soml

import (
	"path"
	"strings"
)

// 이 파일은 **에셋 표**다 — `<image src>`가 가리킨 파일의 바이트·포맷·고유 크기.
//
// 덱 모델에서 처음으로 값이 **디스크를 가리키는** 자리다. 지금까지 덱은 `.slide` 텍스트만으로
// 만들어졌고 렌더러는 순수 함수였다 (같은 덱 → 같은 바이트, 디스크를 건드리지 않는다 —
// render 패키지 주석). 그림은 바이트를 필요로 한다: PPTX는 미디어 파트에 그 바이트를 넣고
// 웹은 산출 디렉토리에 복사한다. 그 바이트가 렌더러에
// 도달하는 길이 이 표다.
//
// **표가 덱에 있는 이유** (Deck.Assets): 에셋은 파생 데이터가 아니라 **입력**이고, 렌더 전에
// 결정적으로 한 번 채워진다 — `--target html,pptx`와 `pptx,html`이 같은 덱을 본다. 그래서
// 지오메트리를 곁 표로 뺀 근거(측정값은 파생이고 채워지는 시점이 타깃 순서에
// 의존한다)가 여기서는 성립하지 않는다. 이미 같은 성질의 표가 덱에 있다 — Deck.Theme이다
// (core가 채우고 두 렌더러가 읽는다). 에셋은 테마의 형제다.
//
// `fs.FS` 핸들을 덱에 실어 렌더러가 그때 읽게 하는 선택지는 버렸다: 렌더러가 순수 함수라는
// 성질과 테스트가 디스크를 건드리지 않는다는 성질이 함께 깨진다.

// ImageFormat은 v1이 그리는 이미지 포맷이다 — **PNG·JPEG 2종**.
//
// 어휘를 둘로 닫은 근거는 `<shape type>`을 OOXML 프리셋으로 닫은 것과 같다: **양 타깃이
// 그리는 것만 연다.** SVG는 PPTX에서 확장 요소(`svgBlip`)와 래스터 폴백이 함께 필요해
// 소비자마다 다르게 열리고, GIF/WebP는
// 애니메이션과 소비자 지원이 갈린다.
//
// 값이 소문자 이름인 이유: image.DecodeConfig가 돌려주는 포맷 이름과 **같은 문자열**이어야
// 판정이 한 번으로 끝난다 (core의 확장자·내용 대조).
type ImageFormat string

const (
	// FormatPNG는 PNG다 (`.png`).
	FormatPNG ImageFormat = "png"
	// FormatJPEG는 JPEG다 (`.jpg`·`.jpeg`).
	FormatJPEG ImageFormat = "jpeg"
)

// ImageFormats는 포맷 어휘 전체다 (순서 = 진단·문서 순서). ImageFits와 같은 태도로 복사본을
// 돌려준다 — 목록은 soml에 있고 **검사는 core**다.
func ImageFormats() []ImageFormat {
	return []ImageFormat{FormatPNG, FormatJPEG}
}

// imageExts는 확장자 → 포맷 표다. 순서가 곧 진단이 나열하는 순서다 (맵이 아닌 이유).
//
// **확장자가 포맷의 선언이고 내용이 판정이다.** 둘을 대조하는 검사(core)가 필요한 이유는
// PPTX의 `[Content_Types].xml`이 **확장자로** 콘텐츠 타입을 정하기 때문이다 —
// 확장자만 바꾼 파일은 "빌드는 성공했는데 PowerPoint가 열지 못하는 산출물"이 되고, 그것이
// 조용한 실패의 교과서다.
var imageExts = []struct {
	ext    string
	format ImageFormat
}{
	{".png", FormatPNG},
	{".jpg", FormatJPEG},
	{".jpeg", FormatJPEG},
}

// ImageExts는 쓸 수 있는 확장자 목록이다 (진단·문서 순서).
func ImageExts() []string {
	exts := make([]string, len(imageExts))
	for i, entry := range imageExts {
		exts[i] = entry.ext
	}
	return exts
}

// Ext는 이 포맷을 적는 **첫** 확장자다 (`.png`·`.jpg`) — 표의 순서가 그 우선순위다.
//
// 포맷 → 확장자 방향이 필요한 자리가 둘이다: 확장자·내용 불일치 진단이 "무엇으로 바꾸라"를
// 말할 때(core의 에셋 패스)와 **PPTX 미디어 파트 이름**을 지을 때(render/pptx). 파트 이름을
// 저자가 쓴 확장자가 아니라 이 값으로 짓는 이유는 `[Content_Types].xml`의 `Default`가
// **확장자로** 콘텐츠 타입을 정하기 때문이다: `.JPG`·`.jpg`·`.jpeg`가 한 포맷이므로 이름을
// 포맷에서 만들면 Default 항목이 포맷당 하나로 모이고, 확장자 대소문자 매칭을 소비자의
// 관용에 맡기지 않는다 (Asset.Key가 대소문자를 눕히지 않는 결정의 짝이다 — 키는 디스크를
// 가리키고 파트 이름은 패키지 안의 이름이다).
func (f ImageFormat) Ext() string {
	for _, entry := range imageExts {
		if entry.format == f {
			return entry.ext
		}
	}
	return ""
}

// ImageFormatForExt는 확장자가 선언하는 포맷이다. 비교는 **대소문자를 무시한다** — `.PNG`와
// `.png`가 다른 포맷이 되면 저자가 파일 이름의 대문자 때문에 그리지 못하는 그림을 얻는다.
func ImageFormatForExt(ext string) (ImageFormat, bool) {
	lower := strings.ToLower(ext)
	for _, entry := range imageExts {
		if entry.ext == lower {
			return entry.format, true
		}
	}
	return "", false
}

// Asset은 에셋 하나다 — 표의 항목.
type Asset struct {
	// Key는 **프로젝트 루트 기준 슬래시 경로**다 (path.Clean 정규화). 표의 키이자
	// ImageStyle.Src가 가리키는 이름이다.
	//
	// 대소문자를 **바꾸지 않는다**: 이 경로로 파일을 다시 읽고(웹 렌더러가 산출 디렉토리로
	// 복사한다) 대소문자를 구별하는 파일 시스템에서 `A.PNG`를 `a.png`로 눕히면 그 파일이
	// 사라진다. 확장자 대소문자에 걸린 판단(콘텐츠 타입·미디어 파트 이름)은 Key가 아니라
	// **Format**이 소유한다 — 그래서 `.JPG`와 `.jpeg`가 한 콘텐츠 타입으로 모인다.
	Key string
	// Format은 **내용이 판정한** 포맷이다 (확장자의 선언이 아니다 — imageExts의 주석).
	Format ImageFormat
	// Width·Height는 고유 크기다. 단위는 **논리 스테이지 px 그대로**다 (좌표계의
	// 단위 하나): 2배 해상도 에셋은 두 배 크게 나오고, 그것을 조절하는 표기는 `w`/`h`/`ratio`다.
	// 고유 크기를 임의의 계수로 나누는 규칙(DPR 추측)은 만들지 않는다 — 저자가 확인할 수
	// 없는 배율이 산출물에 섞이면 좌표계가 하나라는 계약이 무너진다.
	Width  int
	Height int
	// Bytes는 파일 내용 그대로다. 엔진은 바이트를 고치지 않는다 (리사이즈·재압축·EXIF
	// 회전 없음) — 저자가 커밋한 파일과 산출물의 그림이 다른 것이 되면 그 차이를 저자가
	// 확인할 방법이 없다.
	Bytes []byte
}

// Ext는 Key의 확장자다 (소문자). PPTX의 `Default Extension` 항목과 미디어 파트 이름이
// 여기서 나온다 — 그 자리에서 `path.Ext`를 다시 부르면 소문자 정규화가 두 곳에 생긴다.
func (a Asset) Ext() string {
	return strings.ToLower(path.Ext(a.Key))
}

// Assets는 덱이 참조하는 에셋 표다 — **첫 사용의 문서 순서**.
//
// 맵이 아닌 이유는 Theme.Colors와 같다: 산출물이 결정적이어야 한다. 맵 순회로 순서가
// 흔들리면 PPTX 미디어 파트 번호가 실행마다 달라지고 골든이 매번 깨진다.
type Assets []Asset

// Get은 정규화 키로 에셋을 찾는다 (Theme.Color와 같은 태도의 선형 조회 — 표가 작고 순서가
// 계약이다).
func (a Assets) Get(key string) (Asset, bool) {
	for _, asset := range a {
		if asset.Key == key {
			return asset, true
		}
	}
	return Asset{}, false
}

// Keys는 에셋 키 목록이다 (표 순서) — 진단과 산출물 목록이 같은 순서를 쓰게 한다.
func (a Assets) Keys() []string {
	keys := make([]string, len(a))
	for i, asset := range a {
		keys[i] = asset.Key
	}
	return keys
}
