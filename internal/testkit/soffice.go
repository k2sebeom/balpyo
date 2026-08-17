package testkit

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// sofficeTimeout은 변환 하나에 주는 시간이다. LibreOffice는 실패할 때 대화창을 띄우려
// 시도하며 멈추는 쪽이라 상한이 필요하다.
const sofficeTimeout = 120 * time.Second

// OpenPPTX는 LibreOffice가 파일을 **깨끗하게** 여는지 검사하고 PDF 경로를 돌려준다.
//
// "열린다"로는 부족하다. PowerPoint와 LibreOffice는 깨진 패키지를 조용히 복구해서 열어
// 주기 때문에, 복구가 일어나면 사용자는 파일을 열 때마다 경고를 보게 된다. 그래서 stderr에
// 복구·수리·경고 흔적이 있으면 실패로 본다.
func OpenPPTX(t testing.TB, pptxPath string) string {
	t.Helper()
	soffice := RequireSoffice(t)

	outDir := t.TempDir()
	// 사용자 프로필을 테스트마다 새로 만든다. 공유 프로필은 잠금 파일 때문에 병렬 실행에서
	// 서로를 막고, 앞선 실행이 남긴 상태가 결과를 바꾼다.
	profile := filepath.Join(t.TempDir(), "profile")
	stdout, stderr, err := runTool(soffice,
		"--headless", "--norestore", "--nolockcheck", "--nodefault", "--nologo",
		"-env:UserInstallation=file://"+profile,
		"--convert-to", "pdf:impress_pdf_Export",
		"--outdir", outDir,
		pptxPath,
	)
	if err != nil {
		t.Fatalf("LibreOffice 변환 실패: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	if noise := sofficeNoise(stderr); noise != "" {
		t.Errorf("LibreOffice가 경고·복구를 냈다 — 사용자도 파일을 열 때마다 본다:\n%s", noise)
	}

	pdfPath := filepath.Join(outDir, replaceExt(filepath.Base(pptxPath), ".pdf"))
	if _, err := os.Stat(pdfPath); err != nil {
		t.Fatalf("PDF가 나오지 않았다: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	return pdfPath
}

// RenderPPTX는 pptx를 슬라이드별 PNG로 렌더한다 (soffice → pdf → pdftoppm → png).
//
// size는 논리 스테이지 크기다. 웹 스크린샷과 픽셀 비교하려면 같은 크기여야 하므로
// pdftoppm에 목표 크기를 직접 준다 — DPI로 맞추면 페이지 크기 반올림 때문에 한두 px씩
// 어긋난다.
func RenderPPTX(t testing.TB, pptxPath string, size image.Point) []image.Image {
	t.Helper()
	return RenderPDF(t, OpenPPTX(t, pptxPath), size)
}

// RenderPDF는 PDF를 페이지별 PNG로 렌더한다.
func RenderPDF(t testing.TB, pdfPath string, size image.Point) []image.Image {
	t.Helper()
	pdftoppm := RequirePdftoppm(t)
	if size.X <= 0 || size.Y <= 0 {
		t.Fatalf("렌더 크기가 유효하지 않다: %dx%d", size.X, size.Y)
	}

	outDir := t.TempDir()
	prefix := filepath.Join(outDir, "page")
	stdout, stderr, err := runTool(pdftoppm,
		"-png",
		"-scale-to-x", fmt.Sprint(size.X),
		"-scale-to-y", fmt.Sprint(size.Y),
		pdfPath, prefix,
	)
	if err != nil {
		t.Fatalf("pdftoppm 실패: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	// pdftoppm은 페이지 수에 따라 자리수를 채운다 (page-1.png 또는 page-01.png). 한 번의
	// 실행 안에서는 자리수가 같으므로 사전순 정렬이 곧 페이지 순서다.
	matches, err := filepath.Glob(prefix + "-*.png")
	if err != nil {
		t.Fatalf("렌더 결과를 찾을 수 없다: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("pdftoppm이 PNG를 내지 않았다\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	sort.Strings(matches)

	images := make([]image.Image, 0, len(matches))
	for _, match := range matches {
		images = append(images, LoadPNG(t, match))
	}
	return images
}

// sofficeNoise는 stderr에서 무해한 줄을 걷어내고 남은 것을 돌려준다.
//
// 무해 목록을 두는 쪽을 택한 이유: "경고 없음"을 stderr가 비었다는 것으로 정의하면
// 환경이 내는 잡음(자바 런타임 없음 같은)에 매번 걸리고, 반대로 키워드 목록으로 정의하면
// 새로운 문구의 경고를 놓친다. 무해한 줄만 명시하고 나머지는 전부 실패로 본다.
func sofficeNoise(stderr string) string {
	benign := []string{
		"javaldx",        // 자바 런타임 없음 — 임프레스 렌더에 필요 없다
		"Fontconfig",     // 폰트 캐시 경고
		"dbus",           // 컨테이너에 세션 버스가 없다
		"Failed to open", // 위 두 가지가 함께 내는 부수 메시지
	}
	var kept []string
	for _, line := range strings.Split(stderr, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		harmless := false
		for _, pattern := range benign {
			if strings.Contains(trimmed, pattern) {
				harmless = true
				break
			}
		}
		if !harmless {
			kept = append(kept, trimmed)
		}
	}
	return strings.Join(kept, "\n")
}

// runTool은 외부 도구를 시간 제한 안에서 돌리고 출력을 나눠 돌려준다.
func runTool(bin string, args ...string) (stdout, stderr string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), sofficeTimeout)
	defer cancel()

	var outBuf, errBuf bytes.Buffer
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	if ctxErr := ctx.Err(); ctxErr != nil {
		err = fmt.Errorf("%s 시간 초과(%s): %w", filepath.Base(bin), sofficeTimeout, ctxErr)
	}
	return outBuf.String(), errBuf.String(), err
}

// replaceExt는 파일 이름의 확장자를 바꾼다.
func replaceExt(name, ext string) string {
	return strings.TrimSuffix(name, filepath.Ext(name)) + ext
}
