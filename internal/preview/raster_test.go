package preview

import (
	"strings"
	"testing"
)

func TestSofficeNoiseKeepsOnlyRealWarnings(t *testing.T) {
	stderr := strings.Join([]string{
		"javaldx: Could not find a Java Runtime Environment!",
		"Warning: failed to launch javaldx - java may not function correctly",
		"",
		"Fontconfig warning: ignoring UTF-8: not a valid region tag",
	}, "\n")
	if noise := SofficeNoise(stderr); noise != "" {
		t.Errorf("무해한 잡음을 경고로 봤다: %q", noise)
	}

	broken := "Error: source file could not be loaded\nrepairing document"
	noise := SofficeNoise(broken)
	if !strings.Contains(noise, "Error: source file could not be loaded") ||
		!strings.Contains(noise, "repairing document") {
		t.Errorf("실제 경고를 놓쳤다: %q", noise)
	}
}

func TestReplaceExt(t *testing.T) {
	if got := replaceExt("my-talk.pptx", ".pdf"); got != "my-talk.pdf" {
		t.Errorf("got %q", got)
	}
	if got := replaceExt("2026.리뷰.pptx", ".pdf"); got != "2026.리뷰.pdf" {
		t.Errorf("점이 여러 개인 이름에서 마지막 확장자만 바뀌어야 한다: %q", got)
	}
}
