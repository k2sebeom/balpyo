// Command release는 `make dist` 한 줄이 부르는 실행부다 — 절차 자체는 internal/release가
// 소유하고 테스트가 그것을 판정한다.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/k2sebeom/balpyo/internal/release"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("release", flag.ContinueOnError)
	version := fs.String("version", "", "release tag (default: the tag on HEAD)")
	outDir := fs.String("out", "dist", "output directory")
	repo := fs.String("repo", "k2sebeom/balpyo", "owner/name (for download URLs and the formula)")
	targetList := fs.String("targets", "", "platforms to build (default: all — e.g. linux/amd64,darwin/arm64)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	tag := *version
	if tag == "" {
		// 버전은 **태그가 소유한다**. HEAD에 태그가 없으면 만들 버전이 없다 —
		// 커밋 해시를 버전 자리에 넣으면 아카이브 이름이 우리가 발표하지 않은 이름이 된다.
		var err error
		if tag, err = describeTag(); err != nil {
			return fmt.Errorf("%w\ntag HEAD or pass the version explicitly: make dist VERSION=v0.3.0", err)
		}
	}

	targets, err := parseTargets(*targetList)
	if err != nil {
		return err
	}

	result, err := release.Build(context.Background(), release.Options{
		ModuleDir: ".",
		OutDir:    *outDir,
		Version:   tag,
		Repo:      *repo,
		Targets:   targets,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(stdout, "balpyo %s → %s\n", tag, *outDir)
	for _, artifact := range result.Artifacts {
		fmt.Fprintf(stdout, "  %s  %s\n", artifact.SHA256[:12], artifact.Name)
	}
	fmt.Fprintf(stdout, "  %s\n", release.ChecksumsName)
	if result.FormulaPath != "" {
		fmt.Fprintf(stdout, "  %s\n", release.FormulaName)
	} else {
		// 반쪽 포뮬러를 내지 않는 대신 **그것을 말한다** — 조용히 빠진 파일은 릴리스 날 찾게 된다.
		fmt.Fprintf(stdout, "  (the formula is only written when every platform is built — %d/%d this time)\n",
			len(result.Artifacts), len(release.Targets()))
	}
	return nil
}

// describeTag는 HEAD에 붙은 태그를 읽는다. `--exact-match`인 이유는 위와 같다: 가장 가까운
// 태그(`v0.2.0-7-g375c72c`)는 우리가 발표한 이름이 아니다.
func describeTag() (string, error) {
	out, err := exec.Command("git", "describe", "--tags", "--exact-match", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("HEAD has no tag")
	}
	return strings.TrimSpace(string(out)), nil
}

func parseTargets(list string) ([]release.Target, error) {
	if strings.TrimSpace(list) == "" {
		return nil, nil
	}
	known := make(map[string]release.Target, 4)
	var names []string
	for _, target := range release.Targets() {
		known[target.String()] = target
		names = append(names, target.String())
	}
	var targets []release.Target
	for _, name := range strings.Split(list, ",") {
		name = strings.TrimSpace(name)
		target, ok := known[name]
		if !ok {
			return nil, fmt.Errorf("unknown platform %s (available platforms: %s)", name, strings.Join(names, ", "))
		}
		targets = append(targets, target)
	}
	return targets, nil
}
