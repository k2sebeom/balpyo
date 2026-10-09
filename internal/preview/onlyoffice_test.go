package preview

import (
	"archive/zip"
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// writeSlideZip은 변환기가 내는 모양의 그림 묶음을 쓴다 — 이름마다 폭이 다른 1px 높이 그림이라
// 읽은 순서를 폭으로 알 수 있다.
func writeSlideZip(t *testing.T, names map[string]int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "slides.zip")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, width := range names {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(w, image.NewRGBA(image.Rect(0, 0, width, 1))); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// 열 장을 넘는 덱에서 사전순으로 읽으면 image10이 image2 앞에 와 갤러리의 짝이 어긋난다.
func TestReadSlideZipOrdersByNumber(t *testing.T) {
	names := map[string]int{}
	for i := 1; i <= 11; i++ {
		names["image"+strconv.Itoa(i)+".png"] = i
	}
	names["readme.txt"] = 99 // 그림이 아닌 항목은 건너뛴다
	pages, err := readSlideZip(writeSlideZip(t, names))
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 11 {
		t.Fatalf("장수 = %d, want 11", len(pages))
	}
	for i, page := range pages {
		if got := page.Bounds().Dx(); got != i+1 {
			t.Errorf("%d번째 그림의 폭 = %d, want %d (번호순이 아니다)", i+1, got, i+1)
		}
	}
}

// 번호가 비면 어느 슬라이드가 빠졌는지 말한다 — 조용히 당겨 놓으면 뒤의 짝이 전부 어긋난다.
func TestReadSlideZipRejectsGap(t *testing.T) {
	_, err := readSlideZip(writeSlideZip(t, map[string]int{"image1.png": 1, "image3.png": 3}))
	if err == nil || !strings.Contains(err.Error(), "2번") {
		t.Fatalf("빠진 장을 말하지 않는다: %v", err)
	}
}

// 경로는 XML 본문에 들어가므로 이스케이프한다 — 프로젝트 경로의 `&`가 설정 문서를 깨면 변환기는
// 원인을 말하지 않고 실패한다.
func TestX2TParamsEscapesPaths(t *testing.T) {
	doc := x2tParams("/덱 & 발표/in.pptx", "/tmp/out.zip", "/opt/oo", "/tmp/work", image.Pt(960, 720))
	for _, want := range []string{
		"<m_sFileFrom>/덱 &amp; 발표/in.pptx</m_sFileFrom>",
		"<m_nFormatTo>1029</m_nFormatTo>",
		"<m_sFontDir>/opt/oo/sdkjs/common</m_sFontDir>",
		"<width>960</width>",
		"<height>720</height>",
		"<first>false</first>",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("설정 문서에 %q가 없다:\n%s", want, doc)
		}
	}
}

// 환경 변수는 설치 디렉토리도 그 안의 x2t도 받는다. 둘 중 하나가 빠진 설치는 거른다 — 변환기와
// 색인 생성기가 다른 설치에서 오면 글꼴 없는 그림이 조용히 나온다.
func TestLookOnlyOfficeEnv(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"x2t", "docbuilder"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, value := range []string{dir, filepath.Join(dir, "x2t")} {
		t.Setenv(EnvOnlyOffice, value)
		got, err := LookOnlyOffice()
		if err != nil || got != dir {
			t.Errorf("%s=%s → %q, %v; want %q", EnvOnlyOffice, value, got, err, dir)
		}
	}
	if err := os.Remove(filepath.Join(dir, "docbuilder")); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvOnlyOffice, dir)
	if _, err := LookOnlyOffice(); err == nil || !strings.Contains(err.Error(), "docbuilder") {
		t.Errorf("docbuilder 없는 설치를 받았다: %v", err)
	}
}
