package testkit

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// 동일성 판정과 구조 골든은 환경과 무관하다 (TASKS R4 · ADR 0064) — 폰트가 다른 호스트에서도 초록이어야
// 한다. 다른 호스트를 띄우는 대신 폰트를 바꾼다: 테마의 글꼴 토큰을 metric이 전혀 다른 고정폭으로 갈아
// 끼우면 측정 rect는 움직이지만, 두 트리는 여전히 같고 layout.txt는 한 바이트도 움직이지 않아야 한다.
func TestParityHoldsUnderDifferentFonts(t *testing.T) {
	original := fixtureByName(t, "15_local_theme")
	dir := filepath.Join(t.TempDir(), original.Name)
	if err := os.CopyFS(dir, os.DirFS(original.Dir)); err != nil {
		t.Fatal(err)
	}
	tokens := filepath.Join(dir, "themes", "corp", "tokens.json")
	src, err := os.ReadFile(tokens)
	if err != nil {
		t.Fatal(err)
	}
	var theme map[string]any
	if err := json.Unmarshal(src, &theme); err != nil {
		t.Fatal(err)
	}
	// 제목과 본문 둘 다 — 제목만 바꾸면 한 줄짜리 자리라 rect가 움직이지 않는다.
	theme["font"] = []map[string]string{
		{"name": "heading", "latin": "Courier New", "ea": "Courier New", "generic": "monospace", "scheme": "major"},
		{"name": "body", "latin": "Courier New", "ea": "Courier New", "generic": "monospace", "scheme": "minor"},
	}
	swapped, err := json.Marshal(theme)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tokens, swapped, 0o644); err != nil {
		t.Fatal(err)
	}

	before := original.Build(t)
	other := Fixture{Name: original.Name + "(monospace)", Dir: dir}
	after := other.Build(t)

	// 전제: 폰트 교체가 정말 측정을 움직였다 — 안 움직였으면 이 테스트는 아무것도 증명하지 않는다.
	beforeTree, _ := original.Layouts(t, before)
	afterTree, _ := other.Layouts(t, after)
	if FormatLayout(beforeTree, true) == FormatLayout(afterTree, true) {
		t.Fatal("글꼴을 바꿨는데 측정 rect가 그대로다 — 다른 환경을 흉내 내지 못했다")
	}

	other.AssertParity(t, after)
	want, err := os.ReadFile(filepath.Join(original.GoldenDir(), layoutGolden))
	if err != nil {
		t.Fatal(err)
	}
	if got := other.goldenFiles(t, after)[layoutGolden]; !bytes.Equal(got, want) {
		t.Errorf("글꼴만 바꿨는데 구조 스냅샷이 움직였다 — 골든이 환경에 묶여 있다\n%s", diffReport(want, got))
	}
}
