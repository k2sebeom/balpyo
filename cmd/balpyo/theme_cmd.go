package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"text/tabwriter"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/internal/themestore"
)

// themeClient는 `balpyo theme`이 저장소를 받는 쪽이다 — 테스트가 로컬 서버로 바꾼다.
var themeClient = themestore.Client{}

// themeHome은 빌드가 보는 설치한 테마의 홈이다. 홈을 정할 수 없으면 빈 문자열이다 — 설치한 테마를
// 보지 않을 뿐 빌드는 돈다 (테마 이름을 못 찾으면 그 에러가 말한다).
func themeHome() string {
	dir, err := themestore.Home()
	if err != nil {
		return ""
	}
	return dir
}

func openStore() (*themestore.Store, error) {
	dir, err := themestore.Home()
	if err != nil {
		return nil, err
	}
	return &themestore.Store{Dir: dir, Client: themeClient}, nil
}

// runTheme은 `balpyo theme <install|list|remove|update>`다.
func runTheme(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("theme 명령이 필요하다: install · list · remove · update")
	}
	switch args[0] {
	case "install":
		return runThemeInstall(args[1:], stdout, stderr)
	case "list":
		return runThemeList(args[1:], stdout, stderr)
	case "remove":
		return runThemeRemove(args[1:], stdout, stderr)
	case "update":
		return runThemeUpdate(args[1:], stdout, stderr)
	}
	return fmt.Errorf("알 수 없는 theme 명령 %q (쓸 수 있는 명령: install, list, remove, update)", args[0])
}

// parseInterleaved는 플래그와 위치 인자가 섞인 인자를 읽는다 (`install <URL> --force`). flag 패키지는
// 첫 위치 인자에서 멈추므로 남은 것을 다시 읽는다.
func parseInterleaved(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		args = fs.Args()
		if len(args) == 0 {
			return positional, nil
		}
		positional = append(positional, args[0])
		args = args[1:]
	}
}

func themeContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

func runThemeInstall(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("theme install", flag.ContinueOnError)
	fs.SetOutput(stderr)
	force := fs.Bool("force", false, "다른 출처에서 설치한 같은 이름의 테마를 갈아 끼운다")
	positional, err := parseInterleaved(fs, args)
	if err != nil {
		return err
	}
	if len(positional) == 0 {
		return fmt.Errorf("theme install에는 GitHub 저장소 주소나 로컬 디렉토리가 필요하다 (예: balpyo theme install github.com/<owner>/<repo>, balpyo theme install ./my-theme)")
	}
	src, err := themestore.ResolveSource(positional[0])
	if err != nil {
		return err
	}
	store, err := openStore()
	if err != nil {
		return err
	}
	ctx, stop := themeContext()
	defer stop()
	res, err := store.Install(ctx, src, themestore.InstallOptions{Names: positional[1:], Force: *force})
	if err != nil {
		return err
	}
	printDiags(stderr, res.Lint)
	printInstalled(stdout, store, res)
	if len(res.Themes) > 0 {
		fmt.Fprintf(stdout, "balpyo.json에 \"theme\": %q처럼 이름을 쓴다.\n", res.Themes[0].Name)
	}
	return nil
}

func printInstalled(w io.Writer, store *themestore.Store, res themestore.InstallResult) {
	from := res.Source.String()
	if res.Commit != "" {
		from += " (" + shortCommit(res.Commit) + ")"
	}
	verb := "설치했다"
	if res.Source.IsLocal() {
		verb = "복사해 설치했다"
	}
	fmt.Fprintf(w, "%s에서 테마 %d개를 %s → %s\n", from, len(res.Themes), verb,
		filepath.Join(store.Dir, "themes"))
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, r := range res.Themes {
		fmt.Fprintf(tw, "  %s\t%s\n", r.Name, r.Path)
	}
	tw.Flush()
}

func shortCommit(c string) string {
	if len(c) > 7 {
		return c[:7]
	}
	return c
}

func runThemeList(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("theme list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	projectDir := fs.String("project", ".", "프로젝트 루트 (프로젝트 themes/의 테마도 나열한다)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("theme list는 인자를 받지 않는다: %v", fs.Args())
	}
	store, err := openStore()
	if err != nil {
		return err
	}
	installed, err := store.List()
	if err != nil {
		return err
	}
	project := core.ThemeNames(os.DirFS(*projectDir))
	local := map[string]bool{}
	for _, name := range project {
		if !core.IsBuiltinTheme(name) {
			local[name] = true
		}
	}

	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "내장")
	for _, name := range core.BuiltinThemes() {
		fmt.Fprintf(tw, "  %s\n", name)
	}
	fmt.Fprintf(tw, "설치한 테마 (%s)\n", store.Dir)
	if len(installed) == 0 {
		fmt.Fprintln(tw, "  (없음 — balpyo theme install <GitHub URL 또는 경로>로 설치한다)")
	}
	for _, r := range installed {
		origin := r.Origin()
		if origin == "" {
			origin = "(출처 기록 없음)"
		} else if r.Commit != "" {
			origin += " (" + shortCommit(r.Commit) + ")"
		}
		var notes []string
		if core.IsBuiltinTheme(r.Name) {
			notes = append(notes, "내장 테마와 이름이 같아 쓸 수 없다")
		}
		if local[r.Name] {
			notes = append(notes, "프로젝트 themes/와 겹친다 — 이 이름을 요청하면 에러")
		}
		line := "  " + r.Name + "\t" + origin
		if len(notes) > 0 {
			line += "\t← " + strings.Join(notes, ", ")
		}
		fmt.Fprintln(tw, line)
	}
	var names []string
	for name := range local {
		names = append(names, name)
	}
	if len(names) > 0 {
		fmt.Fprintf(tw, "프로젝트 (%s)\n", filepath.Join(*projectDir, "themes"))
		for _, name := range project {
			if local[name] {
				fmt.Fprintf(tw, "  %s\n", name)
			}
		}
	}
	return tw.Flush()
}

func runThemeRemove(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("theme remove", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("theme remove에는 지울 테마 이름이 필요하다")
	}
	store, err := openStore()
	if err != nil {
		return err
	}
	if err := store.Remove(fs.Args()); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "지웠다: %s\n", strings.Join(fs.Args(), ", "))
	return nil
}

func runThemeUpdate(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("theme update", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	store, err := openStore()
	if err != nil {
		return err
	}
	ctx, stop := themeContext()
	defer stop()
	results, err := store.Update(ctx, fs.Args())
	for _, res := range results {
		printDiags(stderr, res.Lint)
		printInstalled(stdout, store, res)
	}
	if err != nil {
		return err
	}
	if len(results) == 0 {
		fmt.Fprintln(stdout, "다시 받을 테마가 없다 (출처를 기록한 설치한 테마가 없다)")
	}
	return nil
}
