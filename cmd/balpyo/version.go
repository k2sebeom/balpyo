package main

import (
	"runtime"
	"runtime/debug"
	"strings"
)

// `balpyo version`은 **버그 리포트에 그대로 붙는 세 줄**이다 (ADR 0061 결정 5):
//
//	balpyo v0.3.0
//	커밋 feda13c (2026-09-21T01:27:36Z)
//	linux/amd64 · go1.26
//
// 출처는 둘뿐이다. 태그는 릴리스 빌드가 `-X main.version=`으로 주입하고, 커밋·시각·더러움은
// 툴체인이 박은 `vcs.*`에서 읽는다 — 같은 사실의 두 번째 출처를 만들지 않으므로 `go install`로
// 받은 바이너리도 커밋을 안다. 플랫폼과 툴체인은 런타임의 사실이라 언제나 안다.

// version은 릴리스 빌드가 -ldflags로 주입하는 **태그 하나**다 (internal/release.LDFlags).
var version = ""

// buildInfo는 버전 줄이 말하는 사실들이다. 모르는 것은 빈 문자열이고, 빈 것은 줄에 실리지
// 않는다 (`알 수 없음`을 찍으면 리포트에 쓸모없는 줄이 하나 늘 뿐이다).
type buildInfo struct {
	// Tag는 주입된 태그이거나 모듈 버전이다 (없으면 개발 빌드).
	Tag string
	// Commit은 vcs.revision 전체 해시다 (짧게 자르는 것은 형식의 일이다).
	Commit string
	// Time은 vcs.time (RFC3339)이다.
	Time string
	// Dirty는 vcs.modified다.
	Dirty bool
	// Platform은 `GOOS/GOARCH`, Toolchain은 `go1.26`이다.
	Platform  string
	Toolchain string
}

// collectBuildInfo는 실행 중인 바이너리의 사실을 모은다. 순수 함수와 갈라 두는 이유는 테스트다:
// `go test -c`로 만든 바이너리에는 `vcs.*`가 하나도 없으므로(ADR 0061 실측) 판정은 아래
// buildInfoFrom에 값을 넣어서 한다.
func collectBuildInfo() buildInfo {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		info = nil
	}
	return buildInfoFrom(info, version, runtime.GOOS+"/"+runtime.GOARCH, runtime.Version())
}

// buildInfoFrom은 빌드 정보와 주입값에서 사실을 고른다. 주입한 태그가 모듈 버전을 **이긴다** —
// 아카이브로 받은 바이너리는 사람이 아는 이름을 말해야 한다.
func buildInfoFrom(info *debug.BuildInfo, tag, platform, toolchain string) buildInfo {
	b := buildInfo{Tag: tag, Platform: platform, Toolchain: toolchain}
	if info == nil {
		return b
	}
	if b.Tag == "" {
		b.Tag = info.Main.Version
	}
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			b.Commit = setting.Value
		case "vcs.time":
			b.Time = setting.Value
		case "vcs.modified":
			b.Dirty = setting.Value == "true"
		}
	}
	return b
}

// shortCommitLen은 커밋을 사람이 읽는 길이로 자른다 — git의 기본 축약과 같은 7자다.
const shortCommitLen = 7

// formatVersion은 세 줄을 만든다. 커밋을 모르면 그 줄이 **없다** (시각은 커밋 줄의 괄호이므로
// 혼자 나가지 않는다).
func formatVersion(b buildInfo) string {
	name := "balpyo (devel)"
	if b.Tag != "" {
		name = "balpyo " + b.Tag
	}
	lines := []string{name}
	if b.Commit != "" {
		commit := b.Commit
		if len(commit) > shortCommitLen {
			commit = commit[:shortCommitLen]
		}
		if b.Dirty {
			commit += "+dirty"
		}
		line := "커밋 " + commit
		if b.Time != "" {
			line += " (" + b.Time + ")"
		}
		lines = append(lines, line)
	}
	return strings.Join(append(lines, b.Platform+" · "+b.Toolchain), "\n")
}

func versionString() string { return formatVersion(collectBuildInfo()) }
