package geom

import (
	"os"
	"slices"
	"testing"
)

// 윈도우의 Chrome · Edge는 PATH에 없다. 후보는 설치 프로그램이 쓰는 환경 변수에서 만들어야
// 하고, 값이 빈 변수로 만든 경로는 후보가 아니다.
func TestCandidatesOnWindowsComeFromInstallDirectories(t *testing.T) {
	env := map[string]string{
		"ProgramFiles":      `C:\Program Files`,
		"ProgramFiles(x86)": `C:\Program Files (x86)\`,
		"LocalAppData":      "",
	}
	got := candidatesFor("windows", func(key string) string { return env[key] })
	want := []string{
		"chrome", "chromium", "msedge",
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("windows candidates:\n got  %q\n want %q", got, want)
	}
}

func TestCandidatesElsewhereIgnoreWindowsDirectories(t *testing.T) {
	got := candidatesFor("linux", func(string) string { return `C:\nope` })
	if got[0] != "chromium" || slices.ContainsFunc(got, func(c string) bool { return c == "chrome" || c == "msedge" }) {
		t.Errorf("non-windows candidates changed: %q", got)
	}
}

// 윈도우에서 Go는 파일에 실행 비트를 주지 않는다 — 비트로 판정하면 chrome.exe를 거절한다.
// 그 자리에서는 확장자가 실행 여부를 정한다.
func TestExecutableOnWindowsReadsTheExtension(t *testing.T) {
	cases := []struct {
		path string
		mode os.FileMode
		ok   bool
	}{
		{`C:\Chrome\chrome.exe`, 0o666, true},
		{`C:\Chrome\CHROME.EXE`, 0o444, true},
		{`C:\tools\run.cmd`, 0o666, true},
		{`C:\tools\readme.txt`, 0o666, false},
		{`C:\tools\soffice`, 0o777, false},
		{`C:\tools\dir.exe`, os.ModeDir | 0o777, false},
	}
	for _, c := range cases {
		if err := executableInfo(c.path, c.mode, "windows"); (err == nil) != c.ok {
			t.Errorf("windows %s (%v): err = %v, want ok=%v", c.path, c.mode, err, c.ok)
		}
	}
	if err := executableInfo("/usr/bin/chromium", 0o755, "linux"); err != nil {
		t.Errorf("linux 0755: %v", err)
	}
	if err := executableInfo("/tmp/chrome.exe", 0o644, "linux"); err == nil {
		t.Error("linux 0644 passed as executable")
	}
}
