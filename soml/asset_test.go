package soml

import (
	"reflect"
	"testing"
)

// 포맷 어휘 3종과 그 순서 — 목록이 진단·문서의 순서이고, 복사본이어야 소비자가 표를 흔들 수
// 없다 (ImageFits와 같은 계약).
func TestImageFormatVocabulary(t *testing.T) {
	if want := []ImageFormat{FormatPNG, FormatJPEG, FormatSVG}; !reflect.DeepEqual(ImageFormats(), want) {
		t.Errorf("ImageFormats() = %v, want %v", ImageFormats(), want)
	}
	formats := ImageFormats()
	formats[0] = "gif"
	if ImageFormats()[0] != FormatPNG {
		t.Error("ImageFormats()가 내부 표를 그대로 내보낸다 — 복사본이어야 한다")
	}
	if want := []string{".png", ".jpg", ".jpeg", ".svg"}; !reflect.DeepEqual(ImageExts(), want) {
		t.Errorf("ImageExts() = %v, want %v", ImageExts(), want)
	}
}

// 확장자 → 포맷. 대소문자를 무시하는 것이 계약이다 — `.PNG`가 다른 포맷이 되면 저자가 파일
// 이름의 대문자 때문에 그리지 못하는 그림을 얻는다.
func TestImageFormatForExt(t *testing.T) {
	cases := []struct {
		ext   string
		want  ImageFormat
		known bool
	}{
		{".png", FormatPNG, true},
		{".PNG", FormatPNG, true},
		{".jpg", FormatJPEG, true},
		{".jpeg", FormatJPEG, true},
		{".JPEG", FormatJPEG, true},
		{".svg", FormatSVG, true},
		{".SVG", FormatSVG, true},
		{".gif", "", false},
		{".webp", "", false},
		{"", "", false},
		{"png", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.ext, func(t *testing.T) {
			got, ok := ImageFormatForExt(tc.ext)
			if got != tc.want || ok != tc.known {
				t.Errorf("ImageFormatForExt(%q) = %q, %v, want %q, %v", tc.ext, got, ok, tc.want, tc.known)
			}
		})
	}
}

// 표는 순서 있는 슬라이스다 — 맵 순회로 순서가 흔들리면 PPTX 미디어 파트 번호가 실행마다
// 달라진다. 조회는 정규화 키로 하고, 확장자는 소문자로 파생된다 (Key는 그대로 남는다).
func TestAssetsTable(t *testing.T) {
	assets := Assets{
		{Key: "assets/wide.png", Format: FormatPNG, Width: 16, Height: 8, Bytes: []byte{1}},
		{Key: "assets/Photo.JPG", Format: FormatJPEG, Width: 16, Height: 8, Bytes: []byte{2}},
	}
	if want := []string{"assets/wide.png", "assets/Photo.JPG"}; !reflect.DeepEqual(assets.Keys(), want) {
		t.Errorf("Keys() = %v, want %v", assets.Keys(), want)
	}
	asset, ok := assets.Get("assets/Photo.JPG")
	if !ok {
		t.Fatal("Get이 표에 있는 키를 찾지 못했다")
	}
	if asset.Format != FormatJPEG || asset.Width != 16 {
		t.Errorf("Get = %+v, want jpeg 16×8", asset)
	}
	// 확장자는 소문자로 파생되지만 **Key는 원문 그대로**다 — 대소문자를 구별하는 파일
	// 시스템에서 이 경로로 파일을 다시 읽는다 (웹 렌더러의 복사).
	if got := asset.Ext(); got != ".jpg" {
		t.Errorf("Ext() = %q, want %q", got, ".jpg")
	}
	if asset.Key != "assets/Photo.JPG" {
		t.Errorf("Key = %q, want 원문 그대로", asset.Key)
	}
	if _, ok := assets.Get("assets/photo.jpg"); ok {
		t.Error("키 조회가 대소문자를 무시했다 — 다른 파일이 한 항목으로 합쳐진다")
	}
	if _, ok := Assets(nil).Get("assets/wide.png"); ok {
		t.Error("빈 표에서 조회가 성공했다")
	}
}

// 이미지 확정 외관의 자리 — 요소에서 바로 읽힌다 (렌더러가 두 단계 nil 검사를 쓰지 않게).
func TestElementImageStyle(t *testing.T) {
	elem := &Element{Name: ImageElement}
	if elem.ImageStyle() != nil {
		t.Error("해석 전에는 nil이어야 한다")
	}
	style := &ImageStyle{Src: "assets/wide.png", Fit: FitCover}
	elem.Appearance = &Appearance{Image: style}
	if elem.ImageStyle() != style {
		t.Error("ImageStyle()이 Appearance.Image를 돌려주지 않는다")
	}
	// 도형 자리와 섞이지 않는다 — 두 확정 값이 형제인 것이 계약이다.
	if elem.ShapeStyle() != nil {
		t.Error("이미지 외관이 도형 자리에 실렸다")
	}
}
