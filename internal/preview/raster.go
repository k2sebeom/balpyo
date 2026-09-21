package preview

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
	"time"
)

// toolTimeout은 변환 하나에 주는 시간이다. LibreOffice는 실패할 때 대화창을 띄우려 시도하며
// 멈추는 쪽이라 상한이 필요하다.
const toolTimeout = 120 * time.Second

// Rasterizer는 PPTX를 슬라이드별 그림으로 되돌린다 (soffice → PDF → pdftoppm → PNG).
//
// 도구 경로를 값으로 들고 있는 이유: 탐색은 한 번이면 충분하고, 무엇보다 **없을 때 멈추는
// 자리**가 명령 시작 지점이어야 한다 — 6장을 빌드한 뒤에 "soffice가 없다"를 아는 것은 낭비다.
type Rasterizer struct {
	Soffice  string
	Pdftoppm string
}

// NewRasterizer는 도구를 찾아 래스터라이저를 만든다. 없으면 어디서 구하는지를 말한다.
//
// 조용히 웹만 내는 길을 두지 않는 이유는 ADR 0047이다: PPTX를 보려고 명령을 쳤는데 PPTX 열이
// 없는 갤러리가 나오면 사람은 **두 타깃이 같다고 읽는다**.
func NewRasterizer() (*Rasterizer, error) {
	soffice, err := LookSoffice()
	if err != nil {
		return nil, toolError("LibreOffice(soffice)", EnvSoffice, err)
	}
	pdftoppm, err := LookPdftoppm()
	if err != nil {
		return nil, toolError("pdftoppm(poppler-utils)", EnvPdftoppm, err)
	}
	return &Rasterizer{Soffice: soffice, Pdftoppm: pdftoppm}, nil
}

// toolError는 도구를 못 찾았을 때의 안내다 — 사용자가 다음에 할 일이 문장 안에 있어야 한다.
func toolError(name, env string, err error) error {
	return fmt.Errorf("%s가 필요하다: %w\n"+
		"PPTX를 그림으로 되돌리는 데만 쓰는 도구다 — 설치하거나 %s로 경로를 주거나, "+
		"`make shell`(테스트 이미지)에서 돌린다. PowerPoint·Keynote가 있으면 산출물을 직접 여는 쪽이 더 정확하다",
		name, err, env)
}

// PDF는 pptx를 PDF로 바꾸고 경로를 돌려준다. noise는 LibreOffice가 낸 무해하지 않은 출력이다.
//
// "열린다"로는 부족하다. PowerPoint와 LibreOffice는 깨진 패키지를 조용히 복구해서 열어 주기
// 때문에, 복구가 일어나면 사용자는 파일을 열 때마다 경고를 보게 된다. 그래서 그 흔적을 따로
// 돌려주고 판정은 호출자가 한다 (하네스는 테스트 실패로, CLI는 경고 줄로).
func (r *Rasterizer) PDF(pptxPath, workDir string) (pdfPath, noise string, err error) {
	outDir := filepath.Join(workDir, "pdf")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", "", fmt.Errorf("변환 디렉토리를 만들 수 없다: %w", err)
	}
	// 사용자 프로필을 실행마다 새로 만든다. 공유 프로필은 잠금 파일 때문에 병렬 실행에서
	// 서로를 막고, 앞선 실행이 남긴 상태가 결과를 바꾼다.
	profile := filepath.Join(workDir, "profile")
	stdout, stderr, err := runTool(r.Soffice,
		"--headless", "--norestore", "--nolockcheck", "--nodefault", "--nologo",
		"-env:UserInstallation=file://"+profile,
		"--convert-to", "pdf:impress_pdf_Export",
		"--outdir", outDir,
		pptxPath,
	)
	if err != nil {
		return "", "", fmt.Errorf("LibreOffice 변환 실패: %w\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	pdfPath = filepath.Join(outDir, replaceExt(filepath.Base(pptxPath), ".pdf"))
	if _, err := os.Stat(pdfPath); err != nil {
		return "", "", fmt.Errorf("PDF가 나오지 않았다: %w\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	return pdfPath, SofficeNoise(stderr), nil
}

// Pages는 PDF를 페이지별 PNG로 렌더한다.
//
// size는 논리 스테이지 크기다. 웹 스크린샷과 픽셀 비교하려면 같은 크기여야 하므로 pdftoppm에
// 목표 크기를 직접 준다 — DPI로 맞추면 페이지 크기 반올림 때문에 한두 px씩 어긋난다
// (ADR 0047 결정 7).
func (r *Rasterizer) Pages(pdfPath string, size image.Point, workDir string) ([]image.Image, error) {
	if size.X <= 0 || size.Y <= 0 {
		return nil, fmt.Errorf("렌더 크기가 유효하지 않다: %dx%d", size.X, size.Y)
	}
	return r.render(pdfPath, filepath.Join(workDir, "png"),
		"-scale-to-x", fmt.Sprint(size.X), "-scale-to-y", fmt.Sprint(size.Y))
}

// PagesAtPointScale은 PDF를 **종이 크기 그대로** 페이지별 PNG로 렌더한다 — 1pt이 1px이다.
//
// 목표 크기를 아는 Pages와 다른 이유는 판정이 다르기 때문이다: 인쇄 게이트는 그림을 대조하는
// 것이 아니라 **종이 크기**를 본다 (한 장이 한 페이지인지, ADR 0062). 크기를 주고 렌더하면
// 페이지가 letter로 나왔어도 PNG는 원하는 크기가 되어 그 실패가 사라진다. 72dpi를 주면
// PNG의 픽셀 수가 곧 페이지의 pt 크기다 (실측 — pdftoppm -r 72는 1pt를 1px로 낸다).
func (r *Rasterizer) PagesAtPointScale(pdfPath, workDir string) ([]image.Image, error) {
	return r.render(pdfPath, filepath.Join(workDir, "png-pt"), "-r", "72")
}

// render는 pdftoppm을 돌리고 나온 PNG를 페이지 순서로 읽는다 (크기 인자는 호출자가 준다).
func (r *Rasterizer) render(pdfPath, outDir string, scale ...string) ([]image.Image, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("렌더 디렉토리를 만들 수 없다: %w", err)
	}
	prefix := filepath.Join(outDir, "page")
	args := append([]string{"-png"}, scale...)
	stdout, stderr, err := runTool(r.Pdftoppm, append(args, pdfPath, prefix)...)
	if err != nil {
		return nil, fmt.Errorf("pdftoppm 실패: %w\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	// pdftoppm은 페이지 수에 따라 자리수를 채운다 (page-1.png 또는 page-01.png). 한 번의 실행
	// 안에서는 자리수가 같으므로 사전순 정렬이 곧 페이지 순서다.
	matches, err := filepath.Glob(prefix + "-*.png")
	if err != nil {
		return nil, fmt.Errorf("렌더 결과를 찾을 수 없다: %w", err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("pdftoppm이 PNG를 내지 않았다\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	sort.Strings(matches)

	images := make([]image.Image, 0, len(matches))
	for _, match := range matches {
		img, err := LoadPNG(match)
		if err != nil {
			return nil, err
		}
		images = append(images, img)
	}
	return images, nil
}

// Slides는 pptx를 슬라이드별 PNG로 렌더한다 (PDF를 거친다).
func (r *Rasterizer) Slides(pptxPath string, size image.Point, workDir string) (pages []image.Image, noise string, err error) {
	pdfPath, noise, err := r.PDF(pptxPath, workDir)
	if err != nil {
		return nil, noise, err
	}
	pages, err = r.Pages(pdfPath, size, workDir)
	return pages, noise, err
}

// SofficeNoise는 stderr에서 무해한 줄을 걷어내고 남은 것을 돌려준다.
//
// 무해 목록을 두는 쪽을 택한 이유: "경고 없음"을 stderr가 비었다는 것으로 정의하면 환경이
// 내는 잡음(자바 런타임 없음 같은)에 매번 걸리고, 반대로 키워드 목록으로 정의하면 새로운
// 문구의 경고를 놓친다. 무해한 줄만 명시하고 나머지는 전부 남긴다.
func SofficeNoise(stderr string) string {
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
	ctx, cancel := context.WithTimeout(context.Background(), toolTimeout)
	defer cancel()

	var outBuf, errBuf bytes.Buffer
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	if ctxErr := ctx.Err(); ctxErr != nil {
		err = fmt.Errorf("%s 시간 초과(%s): %w", filepath.Base(bin), toolTimeout, ctxErr)
	}
	return outBuf.String(), errBuf.String(), err
}

// replaceExt는 파일 이름의 확장자를 바꾼다.
func replaceExt(name, ext string) string {
	return strings.TrimSuffix(name, filepath.Ext(name)) + ext
}
