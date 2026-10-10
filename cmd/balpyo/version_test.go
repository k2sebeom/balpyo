package main

import (
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
)

// 이 파일은 `balpyo version`의 **세 줄**을 판정한다. 버그 리포트에 그대로 붙는 줄이므로
// 요구는 형식이 아니라 사실이다: 어느 태그인지 · 어느 커밋인지(그리고 그 나무가 깨끗했는지) ·
// 어디서 돈 바이너리인지.
//
// 이 판정이 순수 함수 둘(buildInfoFrom·formatVersion)에 걸려 있는 이유는 실측이다: `go test -c`로
// 만든 바이너리에는 `vcs.*`가 **하나도 없다**(빌드 정보를 찍어 확인했다). 그래서 테스트 안에서는
// "커밋이 박혀 있다"를 요구할 수 없고, 박힌 커밋을 보는 판정은 진짜 바이너리를 빌드하는 쪽
// (internal/release)이 갖는다.

// 주입한 태그가 모듈 버전을 **이긴다** — 릴리스 아카이브의 바이너리는 사람이 아는 이름을 말한다.
func TestVersionTagWinsOverModuleVersion(t *testing.T) {
	info := buildInfoFrom(&debug.BuildInfo{
		Main: debug.Module{Version: "v0.0.0-20260921012736-feda13ce5d78"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "feda13ce5d780470789b86c378272d543d8ac60d"},
			{Key: "vcs.time", Value: "2026-09-21T01:27:36Z"},
			{Key: "vcs.modified", Value: "false"},
		},
	}, "v0.3.0", "linux/amd64", "go1.26")

	want := "balpyo v0.3.0\ncommit feda13c (2026-09-21T01:27:36Z)\nlinux/amd64 · go1.26"
	if got := formatVersion(info); got != want {
		t.Errorf("버전 출력 =\n%s\nwant\n%s", got, want)
	}
}

// 주입이 없으면 **툴체인이 박은 것**을 말한다 — `go install`로 받은 바이너리도 커밋을 안다.
func TestVersionWithoutInjectionUsesModuleAndVCS(t *testing.T) {
	info := buildInfoFrom(&debug.BuildInfo{
		Main: debug.Module{Version: "v0.0.0-20260921012736-feda13ce5d78"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "feda13ce5d780470789b86c378272d543d8ac60d"},
			{Key: "vcs.time", Value: "2026-09-21T01:27:36Z"},
			{Key: "vcs.modified", Value: "false"},
		},
	}, "", "darwin/arm64", "go1.26")

	want := "balpyo v0.0.0-20260921012736-feda13ce5d78\ncommit feda13c (2026-09-21T01:27:36Z)\ndarwin/arm64 · go1.26"
	if got := formatVersion(info); got != want {
		t.Errorf("버전 출력 =\n%s\nwant\n%s", got, want)
	}
}

// 더러운 나무는 **그렇게 말한다** — 저자가 자기 트리에서 빌드해 쓰는 단계이므로, 이 표시가 없으면
// 우리가 재현할 수 없는 보고를 받는다.
func TestVersionMarksADirtyTree(t *testing.T) {
	info := buildInfoFrom(&debug.BuildInfo{
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "feda13ce5d780470789b86c378272d543d8ac60d"},
			{Key: "vcs.time", Value: "2026-09-21T01:27:36Z"},
			{Key: "vcs.modified", Value: "true"},
		},
	}, "", "linux/arm64", "go1.26")

	if got := formatVersion(info); !strings.Contains(got, "commit feda13c+dirty (") {
		t.Errorf("버전 출력 =\n%s\nwant 커밋 줄에 +dirty", got)
	}
	// 태그도 모듈 버전도 없으면 개발 빌드다 — 없는 이름을 만들지 않는다.
	if got, want := strings.SplitN(formatVersion(info), "\n", 2)[0], "balpyo (devel)"; got != want {
		t.Errorf("첫 줄 = %q, want %q", got, want)
	}
}

// 모르는 것은 **줄을 내지 않는다** — "커밋 알 수 없음"은 리포트에 쓸모없는 줄을 하나 늘린다.
func TestVersionOmitsTheCommitLineWhenThereIsNone(t *testing.T) {
	info := buildInfoFrom(&debug.BuildInfo{Main: debug.Module{Version: "v0.3.0"}}, "", "linux/amd64", "go1.26")
	got := formatVersion(info)
	if lines := strings.Split(got, "\n"); len(lines) != 2 {
		t.Fatalf("줄 %d개, want 2개 (커밋 줄이 없다):\n%s", len(lines), got)
	}
	if strings.Contains(got, "commit") {
		t.Errorf("커밋을 모르는 빌드가 커밋 줄을 냈다:\n%s", got)
	}
	// 시각만 있고 커밋이 없는 일은 없다 — 시각은 커밋 줄의 괄호이므로 혼자 나가지 않는다.
	if strings.Contains(got, "(") {
		t.Errorf("커밋 없는 출력에 괄호가 있다:\n%s", got)
	}
}

// 커밋만 있고 시각이 없으면 괄호를 비워 두지 않는다.
func TestVersionCommitWithoutTime(t *testing.T) {
	info := buildInfoFrom(&debug.BuildInfo{
		Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "feda13ce5d780470789b86c378272d543d8ac60d"}},
	}, "v0.3.0", "linux/amd64", "go1.26")
	if got, want := strings.Split(formatVersion(info), "\n")[1], "commit feda13c"; got != want {
		t.Errorf("커밋 줄 = %q, want %q", got, want)
	}
}

// 빌드 정보를 아예 못 읽는 빌드(있을 수 있다)도 **플랫폼 줄은 낸다** — 그 줄의 사실은 런타임이
// 소유하므로 언제나 안다.
func TestVersionWithoutBuildInfoStillSaysThePlatform(t *testing.T) {
	got := formatVersion(buildInfoFrom(nil, "", "windows/amd64", "go1.26"))
	want := "balpyo (devel)\nwindows/amd64 · go1.26"
	if got != want {
		t.Errorf("버전 출력 =\n%s\nwant\n%s", got, want)
	}
}

// 그리고 실제 명령이 그 함수를 쓰는지 — 세 줄 중 **마지막 줄은 실행 중인 런타임의 사실**이므로
// 테스트 바이너리에서도 단정할 수 있다 (커밋 줄은 `go test -c`에 없다 — 이 파일 머리 주석).
func TestVersionCommandPrintsRuntimeFacts(t *testing.T) {
	var stdout, stderr strings.Builder
	if err := run([]string{"version"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	out := strings.TrimRight(stdout.String(), "\n")
	lines := strings.Split(out, "\n")
	if !strings.HasPrefix(lines[0], "balpyo ") {
		t.Errorf("첫 줄 = %q, want \"balpyo …\"", lines[0])
	}
	last := lines[len(lines)-1]
	if want := runtime.GOOS + "/" + runtime.GOARCH + " · " + runtime.Version(); last != want {
		t.Errorf("마지막 줄 = %q, want %q", last, want)
	}
}
