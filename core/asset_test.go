package core

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/k2sebeom/balpyo/soml"
)

// 에셋 테스트는 디스크 대신 fstest.MapFS를 쓰고 바이트를 그때 만든다 — 기대 크기가 원본에서
// 나와야 무엇이 틀렸는지 알 수 있다.

// pngBytes는 w×h PNG 바이트다.
func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// jpegBytes는 w×h JPEG 바이트다.
func jpegBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{B: 255, A: 255})
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// resolveImageDeck은 슬라이드 여러 장을 파싱해 외관까지 해석한다 — 에셋 표의 순서가 **덱
// 전체의 문서 순서**라는 계약을 시험하려면 슬라이드가 둘 이상이어야 한다.
func resolveImageDeck(t *testing.T, srcs ...string) (*soml.Deck, soml.Diagnostics) {
	t.Helper()
	deck := &soml.Deck{}
	for i, src := range srcs {
		file := fmt.Sprintf("slides/%03d_image%s", i+1, soml.Ext)
		doc, parseDiags := soml.Parse(file, src)
		if len(parseDiags) != 0 {
			t.Fatalf("%s 파서 진단: %v", file, parseDiags)
		}
		deck.Slides = append(deck.Slides, &soml.Slide{
			Index: i, ID: doc.ID, Source: doc.File, Content: doc.Content, Tokens: doc.Tokens,
		})
	}
	return deck, Resolve(deck, nil)
}

// 정상 경로: 포맷 2종, 같은 파일 두 번(표 항목 1개), 경로 정규화, 고유 크기가 헤더의 값과
// 같은지, 그리고 **순서가 첫 사용의 문서 순서**인지.
func TestResolveAssetsTable(t *testing.T) {
	wide, photo := pngBytes(t, 16, 8), jpegBytes(t, 8, 16)
	fsys := fstest.MapFS{
		"assets/wide.png":  {Data: wide},
		"assets/photo.jpg": {Data: photo},
	}
	deck, diags := resolveImageDeck(t,
		"<slide id=\"a\">\n  <image src=\"assets/photo.jpg\"/>\n  <image src=\"./assets/wide.png\"/>\n</slide>\n",
		// 다음 슬라이드가 같은 두 파일을 다시 쓴다 — 표는 덱 전체에서 한 벌이다.
		"<slide id=\"b\">\n  <image src=\"assets/wide.png\"/>\n  <image src=\"assets//photo.jpg\"/>\n</slide>\n")
	if len(diags) != 0 {
		t.Fatalf("외관 해석 진단이 없어야 한다: %v", diags)
	}
	if diags := ResolveAssets(deck, fsys); len(diags) != 0 {
		t.Fatalf("에셋 진단이 없어야 한다: %v", diags)
	}

	want := soml.Assets{
		{Key: "assets/photo.jpg", Format: soml.FormatJPEG, Width: 8, Height: 16, Bytes: photo},
		{Key: "assets/wide.png", Format: soml.FormatPNG, Width: 16, Height: 8, Bytes: wide},
	}
	if !reflect.DeepEqual(deck.Assets, want) {
		t.Errorf("에셋 표 = %v, want %v", showAssets(deck.Assets), showAssets(want))
	}

	// 확정 외관의 Src가 표의 키다 — 정규화된 두 표기가 같은 항목을 가리킨다.
	for _, style := range imageStyles(t, deck) {
		if _, ok := deck.Assets.Get(style.Src); !ok {
			t.Errorf("확정 src %q가 표에 없다 (키 = %v)", style.Src, deck.Assets.Keys())
		}
	}

	// 두 번 해석하면 같은 표다 — 맵 순회로 순서가 흔들리면 PPTX 미디어 파트 번호가 실행마다
	// 달라지고 골든이 매번 깨진다.
	first := deck.Assets.Keys()
	if diags := ResolveAssets(deck, fsys); len(diags) != 0 {
		t.Fatalf("두 번째 해석의 진단: %v", diags)
	}
	if !reflect.DeepEqual(deck.Assets.Keys(), first) {
		t.Errorf("두 번 해석한 표가 다르다: %v ≠ %v", deck.Assets.Keys(), first)
	}
	if len(deck.Assets) != 2 {
		t.Errorf("표 항목 %d개, want 2개 (두 번째 해석이 표를 부풀렸다)", len(deck.Assets))
	}
}

// 그림이 없는 덱은 파일 시스템을 보지 않는다 — 표가 빈 채로 남고 진단도 없다.
func TestResolveAssetsWithoutImages(t *testing.T) {
	deck, _ := resolveImageDeck(t, "<slide id=\"a\">\n  <p>글자만</p>\n</slide>\n")
	if diags := ResolveAssets(deck, fstest.MapFS{}); len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	if len(deck.Assets) != 0 {
		t.Errorf("에셋 표 = %v, want 빈 표", deck.Assets.Keys())
	}
}

// 파일을 실제로 열어야 알 수 있는 실패 전수. 진단은 **값 스팬**을 가리키고 고칠 방법을 말한다.
func TestResolveAssetsErrors(t *testing.T) {
	valid := pngBytes(t, 4, 4)
	jpegData := jpegBytes(t, 4, 4)
	fsys := fstest.MapFS{
		"assets/ok.png":         {Data: valid},
		"assets/dir.png/in.txt": {Data: []byte("x")},
		"assets/empty.png":      {Data: nil},
		"assets/cut.png":        {Data: valid[:20]},
		"assets/liar.png":       {Data: jpegData},
		"assets/text.png":       {Data: []byte("이건 그림이 아니다")},
	}
	cases := []struct {
		name     string
		src      string
		contains []string
	}{
		{"없는 파일", "assets/missing.png",
			[]string{"does not exist", "relative to the project root"}},
		{"디렉토리", "assets/dir.png",
			[]string{"points to a directory"}},
		{"0바이트", "assets/empty.png",
			[]string{"is empty", "0 bytes"}},
		{"잘린 헤더", "assets/cut.png",
			[]string{"cannot read the image header", "truncated"}},
		{"확장자와 실제 포맷 불일치", "assets/liar.png",
			[]string{"the extension of", "png", "the content is", "jpeg", "PowerPoint"}},
		{"이미지가 아닌 내용", "assets/text.png",
			[]string{"is not png", "PowerPoint"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			attrs := `src="` + tc.src + `"`
			deck, resolveDiags := resolveImageDeck(t, imageSlide(attrs))
			if len(resolveDiags) != 0 {
				t.Fatalf("외관 해석은 통과해야 한다 (경로 문법은 맞다): %v", resolveDiags)
			}
			diags := ResolveAssets(deck, fsys)
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
			}
			assertDiag(t, diags[0], 2, imageCol(attrs, tc.src), tc.contains...)
			// 실패한 에셋은 표에 실리지 않는다 — 렌더러가 반쪽 항목을 보지 않게.
			if len(deck.Assets) != 0 {
				t.Errorf("실패한 에셋이 표에 실렸다: %v", deck.Assets.Keys())
			}
		})
	}

	// 같은 파일을 두 번 참조하면 진단도 **한 번**이다 — 표를 채우는 시도가 항목당 한 번이라는
	// 뜻이고, 저자는 같은 오타를 슬라이드 수만큼 읽지 않는다.
	deck, _ := resolveImageDeck(t,
		"<slide id=\"a\">\n  <image src=\"assets/missing.png\"/>\n  <image src=\"assets/missing.png\"/>\n</slide>\n")
	if diags := ResolveAssets(deck, fsys); len(diags) != 1 {
		t.Errorf("진단 %d개, want 1개: %v", len(diags), diags)
	}
}

// 경로 문법에서 이미 실패한 `src`는 파일을 찾지 않는다 — 같은 그림에 두 번째 진단을 얹으면
// 저자가 고칠 곳이 둘로 보인다 (값 층 전체의 정책).
func TestResolveAssetsSkipsUnresolvedSrc(t *testing.T) {
	deck, resolveDiags := resolveImageDeck(t, imageSlide(`src="assets/anim.gif"`))
	if len(resolveDiags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(resolveDiags), resolveDiags)
	}
	if diags := ResolveAssets(deck, fstest.MapFS{}); len(diags) != 0 {
		t.Errorf("두 번째 진단이 났다: %v", diags)
	}
	if len(deck.Assets) != 0 {
		t.Errorf("에셋 표 = %v, want 빈 표", deck.Assets.Keys())
	}
}

// 파일 시스템 없이 그림을 해석하려는 것은 **엔진 버그**다 (호출자가 조립을 빠뜨렸다) —
// 조용히 빈 표를 남기면 산출물에서 그림만 사라진다.
func TestResolveAssetsWithoutFS(t *testing.T) {
	deck, _ := resolveImageDeck(t, imageSlide(`src="assets/wide.png"`))
	diags := ResolveAssets(deck, nil)
	if len(diags) != 1 || !diags.HasErrors() {
		t.Fatalf("진단 %d개, want 에러 1개: %v", len(diags), diags)
	}
	if !strings.Contains(diags[0].Message, "engine bug") {
		t.Errorf("진단 = %q, want 엔진 버그", diags[0].String())
	}
}

// 프로젝트 밖을 가리키는 심볼릭 링크는 `..`로 나간 src와 같은 에러다 — 링크를 따라가 읽으면 빌드가
// 이 기계의 디렉토리에 묶이고, 커밋된 것은 링크뿐이라 다른 기계에서는 그림이 사라진다. 프로젝트 안을
// 가리키는 링크는 그대로 읽는다 (같은 파일을 두 이름으로 부를 뿐이다).
//
// 디스크에 진짜 링크를 만든다: 밖을 가리키는 링크의 대상은 fstest.MapFS로는 만들 수 없다 (맵 밖은 없다).
func TestResolveAssetsSymlinks(t *testing.T) {
	base := t.TempDir()
	project := filepath.Join(base, "project")
	outside := filepath.Join(base, "outside")
	for _, dir := range []string{filepath.Join(project, "assets"), outside} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	data := pngBytes(t, 4, 4)
	for _, file := range []string{filepath.Join(project, "assets", "ok.png"), filepath.Join(outside, "a.png")} {
		if err := os.WriteFile(file, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	links := map[string]string{
		"assets/out.png":   "../../outside/a.png",
		"assets/abs.png":   filepath.Join(outside, "a.png"),
		"shared":           "../outside",
		"assets/alias.png": "ok.png",
		"assets/hop.png":   "alias.png",
		"assets/up.png":    "../assets/ok.png",
	}
	for name, target := range links {
		if err := os.Symlink(target, filepath.Join(project, filepath.FromSlash(name))); err != nil {
			t.Skipf("이 파일 시스템은 심볼릭 링크를 만들 수 없다: %v", err)
		}
	}
	// 파이프라인처럼 설치한 테마의 홈을 붙인 FS로도 같은 판정이어야 한다 — 감싼 FS가 링크를 숨기면
	// 빌드만 검사를 건너뛴다.
	systems := map[string]fs.FS{
		"DirFS":               os.DirFS(project),
		"WithInstalledThemes": WithInstalledThemes(os.DirFS(project), fstest.MapFS{}),
	}
	for fsName, fsys := range systems {
		for _, src := range []string{"assets/out.png", "assets/abs.png", "shared/a.png"} {
			t.Run(fsName+"/밖/"+src, func(t *testing.T) {
				attrs := `src="` + src + `"`
				deck, resolveDiags := resolveImageDeck(t, imageSlide(attrs))
				if len(resolveDiags) != 0 {
					t.Fatalf("외관 해석은 통과해야 한다: %v", resolveDiags)
				}
				diags := ResolveAssets(deck, fsys)
				if len(diags) != 1 {
					t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
				}
				assertDiag(t, diags[0], 2, imageCol(attrs, src), "symbolic link", "outside the project")
				if len(deck.Assets) != 0 {
					t.Errorf("밖을 가리키는 에셋이 표에 실렸다: %v", deck.Assets.Keys())
				}
			})
		}
		for _, src := range []string{"assets/alias.png", "assets/hop.png", "assets/up.png"} {
			t.Run(fsName+"/안/"+src, func(t *testing.T) {
				deck, _ := resolveImageDeck(t, imageSlide(`src="`+src+`"`))
				if diags := ResolveAssets(deck, fsys); len(diags) != 0 {
					t.Fatalf("프로젝트 안의 링크는 통과해야 한다: %v", diags)
				}
				if _, ok := deck.Assets.Get(src); !ok || len(deck.Assets) != 1 {
					t.Errorf("에셋 표 = %v, want [%s]", deck.Assets.Keys(), src)
				}
			})
		}
	}
}

// imageStyles는 덱의 모든 확정 이미지 외관을 문서 순서로 모은다.
func imageStyles(t *testing.T, deck *soml.Deck) []*soml.ImageStyle {
	t.Helper()
	var styles []*soml.ImageStyle
	var walk func(node soml.Node)
	walk = func(node soml.Node) {
		elem, ok := node.(*soml.Element)
		if !ok {
			return
		}
		if style := elem.ImageStyle(); style != nil {
			styles = append(styles, style)
		}
		for _, child := range elem.Children {
			walk(child)
		}
	}
	for _, slide := range deck.Slides {
		for _, node := range slide.Content {
			walk(node)
		}
	}
	return styles
}

// showAssets는 실패 메시지용 문자열이다 — 바이트를 그대로 찍으면 실패 출력이 읽을 수 없게 된다.
func showAssets(assets soml.Assets) string {
	var parts []string
	for _, asset := range assets {
		parts = append(parts, fmt.Sprintf("%s(%s %dx%d %dB)",
			asset.Key, asset.Format, asset.Width, asset.Height, len(asset.Bytes)))
	}
	return "[" + strings.Join(parts, " ") + "]"
}
