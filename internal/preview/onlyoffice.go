package preview

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// EnvOnlyOffice는 OnlyOffice Document Builder 설치 디렉토리(`x2t`와 `docbuilder`가 있는 곳)나
// 그 안의 `x2t` 경로다.
const EnvOnlyOffice = "BALPYO_ONLYOFFICE"

// OnlyOfficeCandidates는 Document Builder 설치 디렉토리 후보다 — 배포 패키지(.deb/.rpm)와
// 압축본을 /opt에 푼 설치가 같은 자리를 쓴다. 윈도우 설치 프로그램은 Program Files 아래에 둔다.
var OnlyOfficeCandidates = onlyOfficeCandidates(runtime.GOOS, os.Getenv)

func onlyOfficeCandidates(goos string, getenv func(string) string) []string {
	if goos == "windows" {
		return windowsInstalls(getenv, "ProgramFiles", `ONLYOFFICE\DocumentBuilder`)
	}
	return []string{"/opt/onlyoffice/documentbuilder"}
}

// LookOnlyOffice는 Document Builder 설치 디렉토리를 찾는다 (환경 변수 → 후보 디렉토리).
//
// 실행 파일 하나가 아니라 디렉토리를 찾는 이유: 변환기(`x2t`)는 옆에 있는 공유 라이브러리와
// 글꼴 색인(`sdkjs/common/AllFonts.js`)을 읽고, 그 색인은 같은 디렉토리의 `docbuilder`가 만든다.
// 셋 중 하나라도 다른 설치에서 오면 글꼴 없는 그림이 조용히 나온다.
func LookOnlyOffice() (string, error) {
	if custom := os.Getenv(EnvOnlyOffice); custom != "" {
		dir := custom
		if base := filepath.Base(custom); base == "x2t" || strings.EqualFold(base, "x2t.exe") {
			dir = filepath.Dir(custom)
		}
		if err := onlyOfficeDir(dir); err != nil {
			return "", fmt.Errorf("%s=%s: %w", EnvOnlyOffice, custom, err)
		}
		return dir, nil
	}
	for _, dir := range OnlyOfficeCandidates {
		if onlyOfficeDir(dir) == nil {
			return dir, nil
		}
	}
	return "", fmt.Errorf("cannot find Document Builder (candidates: %s)", strings.Join(OnlyOfficeCandidates, ", "))
}

// onlyOfficeDir는 dir이 변환기와 색인 생성기를 함께 가진 설치인지 본다.
func onlyOfficeDir(dir string) error {
	for _, name := range []string{"x2t", "docbuilder"} {
		if err := executable(filepath.Join(dir, exeName(name))); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

// OnlyOffice는 OnlyOffice의 변환기(`x2t`)로 PPTX를 슬라이드별 PNG로 되돌린다.
//
// LibreOffice보다 앞에 두는 이유: OnlyOffice는 OOXML을 1차 형식으로 읽는 엔진이라 프리셋 도형의
// 텍스트 영역 공식 · 플레이스홀더 상속 · 테마 색 변형을 명세대로 계산하고, 커닝을 켜지 않은 런을
// PowerPoint처럼 커닝 없이 놓는다. 그래서 우리 리더(동일성 판정)가 같은 사람이 쓴 탓에 보지
// 못하는 갈라짐 — 같은 XML을 다른 엔진이 어떻게 읽는가 — 이 이 렌더에서 드러난다.
// LibreOffice는 ODF로 옮긴 뒤 그리므로 그 번역이 오히려 차이를 덮는다 (한글↔라틴 간격 같은
// 자기 규칙을 더한다).
type OnlyOffice struct {
	// Dir는 Document Builder 설치 디렉토리다 (LookOnlyOffice).
	Dir string
}

// Name은 갤러리와 로그에 쓰는 엔진 이름이다.
func (o *OnlyOffice) Name() string { return "OnlyOffice" }

// Slides는 pptx를 슬라이드별 PNG로 렌더한다 — size는 논리 스테이지 크기이고 PNG가 그 크기로 나온다.
//
// 변환 전에 `docbuilder`를 빈 스크립트로 한 번 돌린다. Document Builder는 시작할 때 설치된 글꼴
// 목록을 색인(`AllFonts.js`)과 대조해 바뀌었으면 다시 만들고, `x2t`는 그 색인으로만 글꼴을 찾는다
// — 색인이 없거나 낡으면 변환은 성공하면서 **한글 글리프만 빠진 그림**을 낸다. 대조는 0.3초
// 남짓이라 매번 돈다. 설치 디렉토리에 쓸 수 없어 갱신이 실패해도 색인이 남아 있으면 그것으로 간다.
func (o *OnlyOffice) Slides(pptxPath string, size image.Point, workDir string) (pages []image.Image, noise string, err error) {
	if size.X <= 0 || size.Y <= 0 {
		return nil, "", fmt.Errorf("invalid render size: %dx%d", size.X, size.Y)
	}
	absPPTX, err := filepath.Abs(pptxPath)
	if err != nil {
		return nil, "", fmt.Errorf("cannot resolve PPTX path: %w", err)
	}
	work := filepath.Join(workDir, "onlyoffice")
	if err := os.MkdirAll(work, 0o755); err != nil {
		return nil, "", fmt.Errorf("cannot create conversion directory: %w", err)
	}
	work, err = filepath.Abs(work)
	if err != nil {
		return nil, "", fmt.Errorf("cannot resolve conversion directory path: %w", err)
	}

	fontIndex := filepath.Join(o.Dir, "sdkjs", "common", "AllFonts.js")
	empty := filepath.Join(work, "fonts.docbuilder")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		return nil, "", fmt.Errorf("cannot write font index script: %w", err)
	}
	_, refreshErr, refreshFail := o.run("docbuilder", empty)
	if _, statErr := os.Stat(fontIndex); statErr != nil {
		return nil, "", fmt.Errorf("OnlyOffice font index missing (%s) — docbuilder failed to create it: %v\n%s",
			fontIndex, refreshFail, refreshErr)
	}

	out := filepath.Join(work, "slides.zip")
	params := filepath.Join(work, "params.xml")
	if err := os.WriteFile(params, []byte(x2tParams(absPPTX, out, o.Dir, work, size)), 0o644); err != nil {
		return nil, "", fmt.Errorf("cannot write conversion config: %w", err)
	}
	stdout, stderr, err := o.run("x2t", params)
	if err != nil {
		return nil, "", fmt.Errorf("OnlyOffice conversion failed: %w\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	pages, err = readSlideZip(out)
	if err != nil {
		return nil, "", fmt.Errorf("%w\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	return pages, strings.TrimSpace(stderr), nil
}

// run은 설치 디렉토리의 도구를 그 디렉토리에서 돌린다 — 도구가 공유 라이브러리와 `sdkjs`를
// 상대 경로로 찾기 때문이다.
func (o *OnlyOffice) run(tool string, args ...string) (stdout, stderr string, err error) {
	return runToolIn(o.Dir, []string{"LD_LIBRARY_PATH=" + o.Dir}, filepath.Join(o.Dir, exeName(tool)), args...)
}

// x2tParams는 변환 하나의 설정 문서다 — PPTX → 슬라이드별 PNG 묶음(zip).
//
// 형식 번호 1029는 "그림 묶음"이고, thumbnail의 `first=false`가 첫 장만이 아니라 전부를 뜻한다.
// 글꼴 디렉토리를 색인이 있는 자리로 주는 이유: `x2t`는 그 디렉토리의 `font_selection.bin`으로
// 대체 글꼴을 고른다 — 시스템 글꼴 디렉토리를 주면 색인 없이 라틴 면만 찾는다.
func x2tParams(from, to, dir, temp string, size image.Point) string {
	esc := func(s string) string {
		var b strings.Builder
		_ = xml.EscapeText(&b, []byte(s))
		return b.String()
	}
	common := filepath.Join(dir, "sdkjs", "common")
	return `<?xml version="1.0" encoding="utf-8"?>
<TaskQueueDataConvert>
  <m_sFileFrom>` + esc(from) + `</m_sFileFrom>
  <m_sFileTo>` + esc(to) + `</m_sFileTo>
  <m_nFormatTo>1029</m_nFormatTo>
  <m_sFontDir>` + esc(common) + `</m_sFontDir>
  <m_sAllFontsPath>` + esc(filepath.Join(common, "AllFonts.js")) + `</m_sAllFontsPath>
  <m_sThemeDir>` + esc(filepath.Join(dir, "sdkjs", "slide", "themes")) + `</m_sThemeDir>
  <m_bDontSaveAdditional>true</m_bDontSaveAdditional>
  <m_sTempDir>` + esc(temp) + `</m_sTempDir>
  <m_oThumbnail>
    <format>4</format>
    <aspect>1</aspect>
    <first>false</first>
    <width>` + strconv.Itoa(size.X) + `</width>
    <height>` + strconv.Itoa(size.Y) + `</height>
  </m_oThumbnail>
</TaskQueueDataConvert>
`
}

// readSlideZip은 변환기가 낸 묶음에서 PNG를 슬라이드 순서로 읽는다.
//
// 이름이 `image<N>.png`(N은 1부터, 자리수 채움 없음)라 사전순이 아니라 번호순으로 정렬한다 —
// 사전순이면 열 장째부터 `image10`이 `image2` 앞에 온다.
func readSlideZip(path string) ([]image.Image, error) {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("OnlyOffice produced no image archive: %w", err)
	}
	defer archive.Close()
	type page struct {
		n    int
		file *zip.File
	}
	var found []page
	for _, f := range archive.File {
		name := strings.TrimSuffix(strings.TrimPrefix(f.Name, "image"), ".png")
		n, err := strconv.Atoi(name)
		if err != nil || !strings.HasSuffix(f.Name, ".png") {
			continue
		}
		found = append(found, page{n, f})
	}
	if len(found) == 0 {
		return nil, fmt.Errorf("OnlyOffice image archive has no slides (%s)", path)
	}
	sort.Slice(found, func(i, j int) bool { return found[i].n < found[j].n })
	images := make([]image.Image, 0, len(found))
	for i, p := range found {
		if p.n != i+1 {
			return nil, fmt.Errorf("OnlyOffice image archive is missing slide %d", i+1)
		}
		rc, err := p.file.Open()
		if err != nil {
			return nil, err
		}
		img, err := png.Decode(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p.file.Name, err)
		}
		images = append(images, img)
	}
	return images, nil
}
