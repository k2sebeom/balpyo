package core

import (
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// 이 파일은 **스킬 게이트**다: skills/ 아래 에이전트 스킬은 바이너리 · 설치한 테마와 함께 배포되는
// 유일한 문서라서, 저장소에만 있는 파일을 가리키면 사용자의 에이전트는 그 길을 따라갈 수 없다.
// 그래서 두 가지를 판정한다 — 스킬이 저장소 경로를 말하지 않는지, 그리고 스킬에 실린 내장 테마
// 사본이 바이너리에 임베드된 테마와 같은지.

const skillsDir = "../skills"

// builtinThemesSkill은 내장 테마의 값을 담은 스킬 참고 파일이다. 상속할 부모의 값을 아는 길이
// 이 사본뿐이다 — 이름 단위 병합에서 역할 하나를 다시 적으면 부모의 나머지 필드가 남지 않으므로
// 저자는 부모의 값을 보고 한 벌을 통째로 적어야 한다.
const builtinThemesSkill = "theme/references/builtin-themes.md"

// repoPathPattern은 스킬에 나오면 안 되는 저장소 안의 자리다. 사용자에게는 저장소가 없다.
// 앞에 `/`가 붙은 자리는 빼는데, 설치 명령(`go install github.com/…/cmd/balpyo`)은 저장소를
// 읽으라는 말이 아니라 바이너리를 얻는 길이기 때문이다.
var repoPathPattern = regexp.MustCompile(
	`CLAUDE_PLUGIN_ROOT|(?:^|[^/\w.])(docs|contributing|testdata|examples|core|soml|render|internal|cmd)/[\w.{}-]` +
		`|_test\.go|\bgoldens?\b`)

func TestSkillsStaySelfContained(t *testing.T) {
	err := filepath.WalkDir(skillsDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(data), "\n") {
			if m := repoPathPattern.FindString(line); m != "" {
				t.Errorf("%s:%d refers to the repository (%q) — skills ship without it; inline the fact instead", p, i+1, m)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestGoldenSkillBuiltinThemes는 스킬의 내장 테마 사본을 임베드된 테마에서 다시 만들어 비교한다.
// BALPYO_UPDATE_GOLDEN=1(`make golden`)이면 파일을 다시 쓴다.
func TestGoldenSkillBuiltinThemes(t *testing.T) {
	got, err := renderBuiltinThemesSkill()
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(skillsDir, builtinThemesSkill)
	if os.Getenv("BALPYO_UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(target, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("updated %s", target)
		return
	}
	want, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("%v — to create it, run `make golden`", err)
	}
	if string(want) != got {
		t.Errorf("%s differs from the built-in themes — to update, run `make golden`", target)
	}
}

// 주석은 사본에서 뺀다: 내장 테마의 주석은 기여자를 위한 것이고(한국어), 스킬의 독자는 값을 본다.
var (
	jsonCommentPattern = regexp.MustCompile(`"\$comment[^"]*"\s*:\s*"(?:[^"\\]|\\.)*"\s*,?[ \t]*`)
	cssCommentPattern  = regexp.MustCompile(`(?s)/\*.*?\*/[ \t]*\n?`)
	blankRunPattern    = regexp.MustCompile(`\n[ \t]*\n(?:[ \t]*\n)+`)
	openBlankPattern   = regexp.MustCompile(`\{\n[ \t]*\n`)
)

func stripSkillJSONComments(name string, src []byte) (string, error) {
	out := jsonCommentPattern.ReplaceAllString(string(src), "")
	out = blankRunPattern.ReplaceAllString(out, "\n\n")
	out = openBlankPattern.ReplaceAllString(out, "{\n")
	if !json.Valid([]byte(out)) {
		return "", &fs.PathError{Op: "strip comments", Path: name, Err: fs.ErrInvalid}
	}
	return strings.TrimSpace(out), nil
}

func stripSkillCSSComments(src []byte) string {
	return strings.TrimSpace(cssCommentPattern.ReplaceAllString(string(src), ""))
}

func renderBuiltinThemesSkill() (string, error) {
	var b strings.Builder
	b.WriteString(`# Built-in theme files

The exact files of the three built-in themes (` + "`minimal`" + `, ` + "`editorial`" + `, ` + "`ink`" + `) as this
version of the binary embeds them, with comments removed. Use them to see what a child theme
inherits and to copy a complete entry before changing it: re-declaring a role, variant, token or
chrome kind replaces **that whole entry**, so start from the parent's entry and edit the fields
you mean to change. ` + "`balpyo theme info <name>`" + ` prints each theme's README (what it is for).

Files that hold only placement are listed per layout: ` + "`<layout>.css`" + ` is flow mode and
` + "`<layout>.slots.css`" + ` is slot mode. ` + "`canvas`" + ` has no slot file.
`)
	for _, name := range BuiltinThemes() {
		dir := path.Join(themeDir, name)
		b.WriteString("\n## `" + name + "`\n")
		for _, file := range []string{"theme.json", "tokens.json", "components.json"} {
			data, err := fs.ReadFile(builtinThemes, path.Join(dir, file))
			if err != nil {
				return "", err
			}
			src, err := stripSkillJSONComments(path.Join(dir, file), data)
			if err != nil {
				return "", err
			}
			b.WriteString("\n### `" + file + "`\n\n```json\n" + src + "\n```\n")
		}
		entries, err := fs.ReadDir(builtinThemes, path.Join(dir, "layouts"))
		if err != nil {
			return "", err
		}
		b.WriteString("\n### `layouts/`\n")
		for _, e := range entries {
			data, err := fs.ReadFile(builtinThemes, path.Join(dir, "layouts", e.Name()))
			if err != nil {
				return "", err
			}
			b.WriteString("\n`" + e.Name() + "`\n\n```css\n" + stripSkillCSSComments(data) + "\n```\n")
		}
	}
	return b.String(), nil
}
