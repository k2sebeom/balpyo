package testkit

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const (
	// PinsFile은 이미지가 담고 있어야 하는 도구 버전이다 (저장소 기준).
	PinsFile = "docker/versions.env"
	// ImageVersionsFile은 테스트 이미지가 실제 설치 버전을 기록해 둔 파일이다.
	ImageVersionsFile = "/etc/balpyo-versions.txt"
)

// PinnedVersions는 docker/versions.env를 읽는다.
func PinnedVersions(t testing.TB) map[string]string {
	t.Helper()
	path := filepath.Join(RepoRoot(t), filepath.FromSlash(PinsFile))
	pins, err := readEnvFile(path)
	if err != nil {
		t.Fatalf("%s를 읽을 수 없다: %v", PinsFile, err)
	}
	if len(pins) == 0 {
		t.Fatalf("%s에 핀이 없다", PinsFile)
	}
	return pins
}

// ImageVersions는 이미지가 기록한 실제 버전을 읽는다. 이미지 밖(로컬)에서는 파일이 없으므로
// ok가 false다.
func ImageVersions(t testing.TB) (map[string]string, bool) {
	t.Helper()
	versions, err := readEnvFile(ImageVersionsFile)
	if os.IsNotExist(err) {
		return nil, false
	}
	if err != nil {
		t.Fatalf("%s를 읽을 수 없다: %v", ImageVersionsFile, err)
	}
	return versions, true
}

// CompareVersions는 실제 버전이 핀을 만족하는지 본다. 핀은 **접두 일치**로 비교하므로
// `LibreOffice 7.4.7.2`가 `LibreOffice 7.4.7.2 40(Build:2)`를 통과시킨다 — 뒤쪽 빌드
// 문자열은 렌더 결과와 무관한 잡음이다.
//
// 돌려주는 값은 사람이 읽는 불일치 목록이다 (핀 이름 순).
func CompareVersions(pins, actual map[string]string) []string {
	names := make([]string, 0, len(pins))
	for name := range pins {
		names = append(names, name)
	}
	sort.Strings(names)

	var mismatches []string
	for _, name := range names {
		want := pins[name]
		got, ok := actual[name]
		switch {
		case !ok:
			mismatches = append(mismatches, fmt.Sprintf("%s: 이미지에 없다 (핀 %q)", name, want))
		case !strings.HasPrefix(got, want):
			mismatches = append(mismatches, fmt.Sprintf("%s: 핀 %q, 이미지 %q", name, want, got))
		}
	}
	return mismatches
}

// readEnvFile은 `KEY=VALUE` 파일을 읽는다 (`#` 주석과 빈 줄은 건너뛴다).
func readEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	values := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			return nil, fmt.Errorf("`KEY=VALUE` 형식이 아닌 줄: %q", line)
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return values, nil
}
