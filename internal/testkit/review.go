package testkit

import (
	"image"
	"path/filepath"
	"testing"

	"github.com/k2sebeom/balpyo/internal/preview"
)

// ReviewDir는 리뷰 산출물이 쌓이는 디렉토리다 (저장소 기준). 커밋하지 않는다.
const ReviewDir = "review"

// Review는 사람이 눈으로 확인하는 산출물이다 — 픽스처별 웹 스크린샷·PPTX 렌더·픽셀 diff와
// 이들을 나란히 놓는 갤러리 (`make review`).
//
// 갤러리 문서를 만드는 것은 internal/preview다 (ADR 0047) — `balpyo preview`가 같은 코드로
// 같은 갤러리를 만든다. 이 타입이 소유하는 것은 테스트 정책이다: 리뷰 모드가 아니면 아무
// 일도 하지 않는다는 것(호출하는 쪽에 분기가 없다)과 실패를 t.Fatal로 바꾸는 층.
type Review struct {
	// gallery가 nil이면 비활성이다.
	gallery *preview.Gallery
}

// NewReview는 리뷰 디렉토리를 준비한다. 비활성이면 빈 Review를 돌려준다.
func NewReview(t testing.TB) *Review {
	t.Helper()
	if !Reviewing() {
		return &Review{}
	}
	gallery, err := preview.NewGallery(filepath.Join(RepoRoot(t), ReviewDir))
	if err != nil {
		t.Fatalf("%v", err)
	}
	return &Review{gallery: gallery}
}

// Enabled는 리뷰 산출물을 남기는지 알려준다.
func (r *Review) Enabled() bool { return r != nil && r.gallery != nil }

// AddSlide는 슬라이드 하나의 세 이미지를 남긴다.
func (r *Review) AddSlide(t testing.TB, fixture, slideID string, index int,
	webImg, pptxImg, diffImg image.Image, summary string, ok bool) {
	t.Helper()
	if !r.Enabled() {
		return
	}
	verdict := preview.VerdictOver
	if ok {
		verdict = preview.VerdictOK
	}
	err := r.gallery.AddSlide(fixture, preview.Slide{
		ID:    slideID,
		Index: index,
		Shots: []preview.Shot{
			{Caption: "web", Image: webImg},
			{Caption: "pptx", Image: pptxImg},
			{Caption: "diff", Image: diffImg},
		},
		Summary: summary,
		Verdict: verdict,
	})
	if err != nil {
		t.Fatalf("%v", err)
	}
}

// AddArtifact는 산출 파일을 리뷰 디렉토리로 복사한다 — PPTX를 Keynote/PowerPoint로 **직접
// 열어** 보는 확인(TASKS ③)에 파일이 손에 잡히는 곳에 있어야 한다.
func (r *Review) AddArtifact(t testing.TB, fixture, srcPath string) {
	t.Helper()
	if !r.Enabled() {
		return
	}
	if err := r.gallery.AddArtifact(fixture, srcPath); err != nil {
		t.Fatalf("%v", err)
	}
}

// Write는 갤러리 index.html을 쓴다.
func (r *Review) Write(t testing.TB) {
	t.Helper()
	if !r.Enabled() {
		return
	}
	indexPath, err := r.gallery.Write()
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("리뷰 갤러리: %s", indexPath)
}
