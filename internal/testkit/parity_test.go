package testkit

import "testing"

// 픽스처마다 웹 트리 == PPTX 트리 (ADR 0064의 계약 — 컴포넌트 · 레이아웃 · 외관 · 글자).
func TestFixtureLayoutParity(t *testing.T) {
	for _, f := range Fixtures(t) {
		t.Run(f.Name, func(t *testing.T) {
			f.AssertParity(t, f.Build(t))
		})
	}
}
