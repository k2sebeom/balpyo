package testkit

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// 동일성 판정과 구조 골든은 환경과 무관하다 — 폰트가 다른 호스트에서도 초록이어야
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

// 윈도우 편집기는 줄 끝을 CRLF로 저장한다. 같은 덱을 CRLF로 쓴 저자도 LF로 쓴 저자와 같은 덱을
// 받아야 한다: 구조 스냅샷(코드 블록의 줄 · 텍스트 런)과 진단(`line:col` 포함)이 한 바이트도 다르지
// 않고, 두 타깃의 동일성도 그대로다. 저장소는 체크아웃을 LF로 고정하므로(.gitattributes) CRLF
// 입력은 이 테스트가 만들어서 판정한다.
func TestCRLFSlidesBuildTheSameDeck(t *testing.T) {
	for _, name := range []string{"02_text_subset", "03_layout", "13_code"} {
		t.Run(name, func(t *testing.T) {
			original := fixtureByName(t, name)
			dir := filepath.Join(t.TempDir(), original.Name)
			if err := os.CopyFS(dir, os.DirFS(original.Dir)); err != nil {
				t.Fatal(err)
			}
			converted := 0
			err := filepath.WalkDir(filepath.Join(dir, "slides"), func(path string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() || filepath.Ext(path) != ".slide" {
					return err
				}
				src, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				lf := bytes.ReplaceAll(src, []byte("\r\n"), []byte("\n"))
				converted++
				return os.WriteFile(path, bytes.ReplaceAll(lf, []byte("\n"), []byte("\r\n")), 0o644)
			})
			if err != nil {
				t.Fatal(err)
			}
			if converted == 0 {
				t.Fatal("CRLF로 바꾼 .slide가 없다 — 이 테스트는 아무것도 증명하지 않는다")
			}

			other := Fixture{Name: original.Name + "(crlf)", Dir: dir}
			result := other.Build(t)
			other.AssertParity(t, result)
			got := other.goldenFiles(t, result)
			for _, golden := range []string{layoutGolden, diagnosticsGolden} {
				want, err := os.ReadFile(filepath.Join(original.GoldenDir(), golden))
				if os.IsNotExist(err) {
					if _, ok := got[golden]; ok {
						t.Errorf("CRLF 입력에서만 %s가 생겼다:\n%s", golden, got[golden])
					}
					continue
				}
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(got[golden], want) {
					t.Errorf("줄 끝만 CRLF로 바꿨는데 %s가 움직였다\n%s", golden, diffReport(want, got[golden]))
				}
			}
		})
	}
}
