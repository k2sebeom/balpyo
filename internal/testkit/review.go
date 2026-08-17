package testkit

import (
	"fmt"
	"html"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ReviewDir는 리뷰 산출물이 쌓이는 디렉토리다 (저장소 기준). 커밋하지 않는다.
const ReviewDir = "review"

// Review는 사람이 눈으로 확인하는 산출물이다 — 픽스처별 웹 스크린샷·PPTX 렌더·픽셀 diff와
// 이들을 나란히 놓는 갤러리 (`make review`).
//
// 테스트가 통과하는 것과 결과가 좋아 보이는 것은 다른 문제다. 허용 오차 안에서도 글자가
// 어색하게 잘릴 수 있고, 그건 사람만 알아본다. 그래서 리뷰는 반복 절차의 단계이고(TASKS ③),
// 테스트와 **같은 코드**가 만든다.
//
// Reviewing()이 아니면 모든 메서드가 아무 일도 하지 않는다 — 호출하는 쪽에 분기가 없다.
type Review struct {
	// dir가 비면 비활성이다.
	dir      string
	fixtures []*reviewFixture
}

// reviewFixture는 갤러리에서 픽스처 하나의 묶음이다.
type reviewFixture struct {
	name      string
	slides    []reviewSlide
	artifacts []string
}

// reviewSlide는 갤러리 한 행이다 (웹 · PPTX · diff).
type reviewSlide struct {
	index   int
	slideID string
	web     string
	pptx    string
	diff    string
	summary string
	ok      bool
}

// NewReview는 리뷰 디렉토리를 준비한다. 비활성이면 빈 Review를 돌려준다.
func NewReview(t testing.TB) *Review {
	t.Helper()
	if !Reviewing() {
		return &Review{}
	}
	dir := filepath.Join(RepoRoot(t), ReviewDir)
	// 앞선 실행이 남긴 픽스처가 갤러리에 섞이면 사람이 지금 없는 결과를 보고 판단한다.
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("리뷰 디렉토리를 비울 수 없다: %v", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("리뷰 디렉토리를 만들 수 없다: %v", err)
	}
	return &Review{dir: dir}
}

// Enabled는 리뷰 산출물을 남기는지 알려준다.
func (r *Review) Enabled() bool { return r != nil && r.dir != "" }

// AddSlide는 슬라이드 하나의 세 이미지를 남긴다.
func (r *Review) AddSlide(t testing.TB, fixture, slideID string, index int,
	webImg, pptxImg, diffImg image.Image, summary string, ok bool) {
	t.Helper()
	if !r.Enabled() {
		return
	}
	prefix := fmt.Sprintf("%02d", index+1)
	slide := reviewSlide{
		index:   index,
		slideID: slideID,
		web:     filepath.Join(fixture, prefix+"-web.png"),
		pptx:    filepath.Join(fixture, prefix+"-pptx.png"),
		diff:    filepath.Join(fixture, prefix+"-diff.png"),
		summary: summary,
		ok:      ok,
	}
	SavePNG(t, filepath.Join(r.dir, slide.web), webImg)
	SavePNG(t, filepath.Join(r.dir, slide.pptx), pptxImg)
	SavePNG(t, filepath.Join(r.dir, slide.diff), diffImg)
	f := r.fixture(fixture)
	f.slides = append(f.slides, slide)
}

// AddArtifact는 산출 파일을 리뷰 디렉토리로 복사한다.
//
// 갤러리만으로는 부족한 확인이 있다 — PPTX를 Keynote/PowerPoint로 **직접 열어** 텍스트 박스가
// 진짜 텍스트 박스인지, 테마 색 편집이 먹는지 보는 일 (TASKS ③). 그러려면 파일이 임시
// 디렉토리가 아니라 손에 잡히는 곳에 있어야 한다.
func (r *Review) AddArtifact(t testing.TB, fixture, srcPath string) {
	t.Helper()
	if !r.Enabled() {
		return
	}
	data, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("리뷰 산출물을 읽을 수 없다: %v", err)
	}
	rel := filepath.Join(fixture, filepath.Base(srcPath))
	dest := filepath.Join(r.dir, rel)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatalf("리뷰 디렉토리를 만들 수 없다: %v", err)
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		t.Fatalf("리뷰 산출물을 쓸 수 없다: %v", err)
	}
	f := r.fixture(fixture)
	f.artifacts = append(f.artifacts, rel)
}

// fixture는 픽스처 묶음을 찾거나 만든다 (등록 순서를 유지한다).
func (r *Review) fixture(name string) *reviewFixture {
	for _, f := range r.fixtures {
		if f.name == name {
			return f
		}
	}
	f := &reviewFixture{name: name}
	r.fixtures = append(r.fixtures, f)
	return f
}

// Write는 갤러리 index.html을 쓴다.
func (r *Review) Write(t testing.TB) {
	t.Helper()
	if !r.Enabled() {
		return
	}
	indexPath := filepath.Join(r.dir, "index.html")
	if err := os.WriteFile(indexPath, []byte(r.indexHTML()), 0o644); err != nil {
		t.Fatalf("갤러리를 쓸 수 없다: %v", err)
	}
	t.Logf("리뷰 갤러리: %s", indexPath)
}

// indexHTML은 갤러리 문서다. 의존성 없이 파일을 그대로 열 수 있어야 한다.
func (r *Review) indexHTML() string {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html lang="ko">
<head>
<meta charset="utf-8">
<title>balpyo 리뷰</title>
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
  .cols { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; }
  figure { margin: 0; }
  figcaption { color: #888; font-size: 12px; margin-bottom: 4px; }
  img { width: 100%; display: block; background: #fff; border: 1px solid #333; }
  .artifacts a { color: #7fb3ff; margin-right: 12px; }
</style>
</head>
<body>
<h1>balpyo 리뷰 — 웹 vs PPTX</h1>
<p class="summary">왼쪽이 웹 스크린샷, 가운데가 LibreOffice 렌더, 오른쪽이 픽셀 diff(마젠타가 다른 픽셀)다.</p>
`)
	for _, f := range r.fixtures {
		fmt.Fprintf(&b, "<h2>%s</h2>\n", html.EscapeString(f.name))
		if len(f.artifacts) > 0 {
			b.WriteString(`<p class="artifacts">`)
			for _, artifact := range f.artifacts {
				href := filepath.ToSlash(artifact)
				fmt.Fprintf(&b, "<a href=%q>%s</a>", href, html.EscapeString(filepath.Base(artifact)))
			}
			b.WriteString("</p>\n")
		}
		for _, slide := range f.slides {
			badge, label := "ok", "허용 오차 안"
			if !slide.ok {
				badge, label = "fail", "허용 오차 초과"
			}
			b.WriteString(`<div class="slide">` + "\n")
			fmt.Fprintf(&b, `<div class="meta"><span class="id">#%d %s</span>`+
				`<span class="badge %s">%s</span><span class="summary">%s</span></div>`+"\n",
				slide.index+1, html.EscapeString(slide.slideID), badge, label,
				html.EscapeString(slide.summary))
			b.WriteString(`<div class="cols">` + "\n")
			for _, col := range []struct{ caption, src string }{
				{"web", slide.web}, {"pptx", slide.pptx}, {"diff", slide.diff},
			} {
				fmt.Fprintf(&b, "<figure><figcaption>%s</figcaption><img src=%q alt=%q></figure>\n",
					col.caption, filepath.ToSlash(col.src), col.caption)
			}
			b.WriteString("</div>\n</div>\n")
		}
	}
	b.WriteString("</body>\n</html>\n")
	return b.String()
}
