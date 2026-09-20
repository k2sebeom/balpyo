package preview

import (
	"fmt"
	"html"
	"image"
	"os"
	"path/filepath"
	"strings"
)

// IndexFile은 갤러리 문서의 이름이다.
const IndexFile = "index.html"

// Verdict는 슬라이드 한 장의 판정이다.
type Verdict int

const (
	// VerdictNone은 판정할 것이 없는 경우다 (비교할 렌더가 하나뿐이다).
	VerdictNone Verdict = iota
	// VerdictOK는 허용 오차 안이다.
	VerdictOK
	// VerdictOver는 허용 오차를 넘었다 — 사람이 볼 자리라는 표시이고, 실패가 아니다
	// (ADR 0047 결정 4).
	VerdictOver
)

// Shot은 갤러리의 한 칸이다.
type Shot struct {
	// Caption은 칸 위에 붙는 이름이다 (web · pptx · diff).
	Caption string
	Image   image.Image
}

// Slide는 갤러리의 한 행이다.
type Slide struct {
	ID    string
	Index int
	Shots []Shot
	// Summary는 행 머리에 붙는 한 줄이다 (보통 DiffResult.String()).
	Summary string
	Verdict Verdict
}

// Gallery는 사람이 눈으로 확인하는 산출물이다 — 슬라이드별 렌더 이미지와 그것들을 나란히
// 놓는 문서 하나.
//
// 테스트가 통과하는 것과 결과가 좋아 보이는 것은 다른 문제다. 허용 오차 안에서도 글자가
// 어색하게 잘릴 수 있고, 그건 사람만 알아본다. 그래서 `make test`와 `make review`,
// 그리고 `balpyo preview`가 **같은 코드**로 이 갤러리를 만든다 (ADR 0047).
type Gallery struct {
	// Title은 문서 제목이다.
	Title string
	// Lede는 제목 아래 한 줄 안내다 — 열이 무엇인지 말한다.
	Lede string

	dir    string
	groups []*galleryGroup
}

// galleryGroup은 묶음 하나다 (하네스는 픽스처, CLI는 덱).
type galleryGroup struct {
	name      string
	slides    []Slide
	paths     [][]string // slides와 같은 순서, 칸별 상대 경로
	artifacts []string
}

// NewGallery는 갤러리 디렉토리를 비우고 만든다.
//
// 비우는 이유: 앞선 실행이 남긴 묶음이 문서에 섞이면 사람이 **지금 없는 결과**를 보고
// 판단한다.
func NewGallery(dir string) (*Gallery, error) {
	if dir == "" {
		return nil, fmt.Errorf("갤러리 디렉토리가 비었다")
	}
	if err := os.RemoveAll(dir); err != nil {
		return nil, fmt.Errorf("갤러리 디렉토리를 비울 수 없다: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("갤러리 디렉토리를 만들 수 없다: %w", err)
	}
	return &Gallery{
		Title: "balpyo 리뷰 — 웹 vs PPTX",
		Lede:  "왼쪽이 웹 스크린샷, 가운데가 LibreOffice 렌더, 오른쪽이 픽셀 diff(마젠타가 다른 픽셀)다.",
		dir:   dir,
	}, nil
}

// Dir는 갤러리 디렉토리다.
func (g *Gallery) Dir() string { return g.dir }

// AddSlide는 슬라이드 한 장의 이미지들을 남긴다. Image가 nil인 칸은 건너뛴다.
func (g *Gallery) AddSlide(group string, slide Slide) error {
	target := g.group(group)
	paths := make([]string, 0, len(slide.Shots))
	kept := make([]Shot, 0, len(slide.Shots))
	for _, shot := range slide.Shots {
		if shot.Image == nil {
			continue
		}
		rel := filepath.Join(group, fmt.Sprintf("%02d-%s.png", slide.Index+1, shot.Caption))
		if err := SavePNG(filepath.Join(g.dir, rel), shot.Image); err != nil {
			return err
		}
		kept = append(kept, shot)
		paths = append(paths, rel)
	}
	slide.Shots = kept
	target.slides = append(target.slides, slide)
	target.paths = append(target.paths, paths)
	return nil
}

// AddArtifact는 산출 파일을 갤러리 디렉토리로 복사하고 문서에 링크한다.
//
// 갤러리만으로는 부족한 확인이 있다 — PPTX를 Keynote/PowerPoint로 **직접 열어** 텍스트 박스가
// 진짜 텍스트 박스인지, 테마 색 편집이 먹는지 보는 일. 그러려면 파일이 임시 디렉토리가 아니라
// 손에 잡히는 곳에 있어야 한다.
func (g *Gallery) AddArtifact(group, srcPath string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("검수 산출물을 읽을 수 없다: %w", err)
	}
	rel := filepath.Join(group, filepath.Base(srcPath))
	dest := filepath.Join(g.dir, rel)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("갤러리 디렉토리를 만들 수 없다: %w", err)
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return fmt.Errorf("검수 산출물을 쓸 수 없다: %w", err)
	}
	target := g.group(group)
	target.artifacts = append(target.artifacts, rel)
	return nil
}

// group은 묶음을 찾거나 만든다 (등록 순서를 유지한다).
func (g *Gallery) group(name string) *galleryGroup {
	for _, group := range g.groups {
		if group.name == name {
			return group
		}
	}
	group := &galleryGroup{name: name}
	g.groups = append(g.groups, group)
	return group
}

// Write는 갤러리 문서를 쓰고 그 경로를 돌려준다.
func (g *Gallery) Write() (string, error) {
	indexPath := filepath.Join(g.dir, IndexFile)
	if err := os.WriteFile(indexPath, []byte(g.indexHTML()), 0o644); err != nil {
		return "", fmt.Errorf("갤러리를 쓸 수 없다: %w", err)
	}
	return indexPath, nil
}

// indexHTML은 갤러리 문서다. 의존성 없이 파일을 그대로 열 수 있어야 한다.
func (g *Gallery) indexHTML() string {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html lang="ko">
<head>
<meta charset="utf-8">
<title>` + html.EscapeString(g.Title) + `</title>
<style>
  body { margin: 0; padding: 24px; background: #111; color: #eee;
         font: 14px/1.5 -apple-system, "Noto Sans", sans-serif; }
  h1 { font-size: 20px; }
  h2 { font-size: 16px; margin: 32px 0 8px; border-bottom: 1px solid #333; padding-bottom: 8px; }
  .slide { margin: 16px 0 32px; }
  .meta { display: flex; gap: 12px; align-items: baseline; margin-bottom: 8px; }
  .id { font-weight: 600; }
  .summary { color: #aaa; font-variant-numeric: tabular-nums; }
  .badge { padding: 1px 8px; border-radius: 999px; font-size: 12px; }
  .ok { background: #1d4a2a; color: #b7f0c6; }
  .fail { background: #5a1d1d; color: #f5b7b7; }
  .cols { display: grid; gap: 12px; }
  figure { margin: 0; }
  figcaption { color: #888; font-size: 12px; margin-bottom: 4px; }
  img { width: 100%; display: block; background: #fff; border: 1px solid #333; }
  .artifacts a { color: #7fb3ff; margin-right: 12px; }
</style>
</head>
<body>
`)
	fmt.Fprintf(&b, "<h1>%s</h1>\n", html.EscapeString(g.Title))
	fmt.Fprintf(&b, "<p class=\"summary\">%s</p>\n", html.EscapeString(g.Lede))
	for _, group := range g.groups {
		fmt.Fprintf(&b, "<h2>%s</h2>\n", html.EscapeString(group.name))
		if len(group.artifacts) > 0 {
			b.WriteString(`<p class="artifacts">`)
			for _, artifact := range group.artifacts {
				href := filepath.ToSlash(artifact)
				fmt.Fprintf(&b, "<a href=%q>%s</a>", href, html.EscapeString(filepath.Base(artifact)))
			}
			b.WriteString("</p>\n")
		}
		for i, slide := range group.slides {
			b.WriteString(`<div class="slide">` + "\n")
			fmt.Fprintf(&b, `<div class="meta"><span class="id">#%d %s</span>`,
				slide.Index+1, html.EscapeString(slide.ID))
			switch slide.Verdict {
			case VerdictOK:
				b.WriteString(`<span class="badge ok">허용 오차 안</span>`)
			case VerdictOver:
				b.WriteString(`<span class="badge fail">허용 오차 초과</span>`)
			}
			fmt.Fprintf(&b, `<span class="summary">%s</span></div>`+"\n", html.EscapeString(slide.Summary))
			// 칸 수만큼 열을 나눈다 — 렌더 하나만 본 갤러리는 한 열이다.
			fmt.Fprintf(&b, `<div class="cols" style="grid-template-columns: repeat(%d, 1fr)">`+"\n",
				max(len(slide.Shots), 1))
			for j, shot := range slide.Shots {
				fmt.Fprintf(&b, "<figure><figcaption>%s</figcaption><img src=%q alt=%q></figure>\n",
					html.EscapeString(shot.Caption), filepath.ToSlash(group.paths[i][j]),
					html.EscapeString(shot.Caption))
			}
			b.WriteString("</div>\n</div>\n")
		}
	}
	b.WriteString("</body>\n</html>\n")
	return b.String()
}
