package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/fs"
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

// runTheme은 `balpyo theme <install|list|info|remove|update>`다.
func runTheme(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("a theme command is required: install · list · info · remove · update")
	}
	switch args[0] {
	case "info":
		return runThemeInfo(args[1:], stdout, stderr)
	case "install":
		return runThemeInstall(args[1:], stdout, stderr)
	case "list":
		return runThemeList(args[1:], stdout, stderr)
	case "remove":
		return runThemeRemove(args[1:], stdout, stderr)
	case "update":
		return runThemeUpdate(args[1:], stdout, stderr)
	}
	return fmt.Errorf("unknown theme command %q (available commands: install, list, info, remove, update)", args[0])
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
	force := fs.Bool("force", false, "replace a theme of the same name installed from a different source")
	positional, err := parseInterleaved(fs, args)
	if err != nil {
		return err
	}
	if len(positional) == 0 {
		return fmt.Errorf("theme install needs a GitHub repository URL or a local directory (e.g. balpyo theme install github.com/<owner>/<repo>, balpyo theme install ./my-theme)")
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
		fmt.Fprintf(stdout, "use the name in balpyo.json, e.g. \"theme\": %q\n", res.Themes[0].Name)
	}
	printInfoHint(stdout, res)
	return nil
}

// runThemeInfo는 `balpyo theme info <이름>`이다 — 테마를 만든 사람이 README.md에 적은 쓰임새를 낸다.
//
// 첫 줄은 이름이 풀린 자리, 부모가 있으면 둘째 줄은 부모를 볼 명령이고, 빈 줄 뒤는 README의 바이트
// 그대로다: 덱을 쓰는 사람(과 에이전트)이 읽는 것이 테마 저장소의 README와 같은 글이어야 한다. 부모의
// README를 합치지 않는 이유는 자식이 부모의 의도를 바꾸는 테마일 수 있다는 것이다.
func runThemeInfo(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("theme info", flag.ContinueOnError)
	flags.SetOutput(stderr)
	projectDir := flags.String("project", ".", "project root (its themes/ are looked up too)")
	positional, err := parseInterleaved(flags, args)
	if err != nil {
		return err
	}
	switch {
	case len(positional) == 0:
		return fmt.Errorf("theme info needs the name of a theme (balpyo theme list shows them)")
	case len(positional) > 1:
		return fmt.Errorf("theme info takes one theme name: %v", positional)
	}
	var home fs.FS
	homeDir := themeHome()
	if homeDir != "" {
		home = os.DirFS(homeDir)
	}
	info, err := core.ReadThemeInfo(core.WithInstalledThemes(os.DirFS(*projectDir), home), positional[0])
	if err != nil {
		return err
	}
	switch info.Source {
	case core.ThemeBuiltin:
		fmt.Fprintf(stdout, "%s — built-in theme\n", info.Name)
	case core.ThemeFromProject:
		fmt.Fprintf(stdout, "%s — project theme (%s)\n", info.Name, filepath.Join(*projectDir, filepath.FromSlash(info.Dir)))
	case core.ThemeInstalled:
		fmt.Fprintf(stdout, "%s — installed theme (%s)\n", info.Name, filepath.Join(homeDir, filepath.FromSlash(info.Dir)))
	}
	if info.Extends != "" {
		fmt.Fprintf(stdout, "extends %s — see: balpyo theme info %s\n", info.Extends, info.Extends)
	}
	fmt.Fprintln(stdout)
	if !info.HasReadme {
		fmt.Fprintln(stdout, "this theme has no README.md")
		return nil
	}
	_, err = stdout.Write(info.Readme)
	return err
}

func printInstalled(w io.Writer, store *themestore.Store, res themestore.InstallResult) {
	from := res.Source.String()
	if res.Commit != "" {
		from += " (" + shortCommit(res.Commit) + ")"
	}
	verb := "installed"
	if res.Source.IsLocal() {
		verb = "copied and installed"
	}
	fmt.Fprintf(w, "%s %s from %s → %s\n", verb, count(len(res.Themes), "theme"), from,
		filepath.Join(store.Dir, "themes"))
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, r := range res.Themes {
		fmt.Fprintf(tw, "  %s\t%s\n", r.Name, r.Path)
	}
	tw.Flush()
}

// printInfoHint는 방금 받은 테마의 README를 읽는 명령을 가리킨다 — 테마를 쓰기 전에 읽을 글이 그것이다.
func printInfoHint(w io.Writer, res themestore.InstallResult) {
	if len(res.Themes) > 0 {
		fmt.Fprintf(w, "see: balpyo theme info %s\n", res.Themes[0].Name)
	}
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
	projectDir := fs.String("project", ".", "project root (also lists the themes in the project's themes/)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("theme list takes no arguments: %v", fs.Args())
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
	fmt.Fprintln(tw, "Built-in")
	for _, name := range core.BuiltinThemes() {
		fmt.Fprintf(tw, "  %s\n", name)
	}
	fmt.Fprintf(tw, "Installed (%s)\n", store.Dir)
	if len(installed) == 0 {
		fmt.Fprintln(tw, "  (none — install with balpyo theme install <GitHub URL or path>)")
	}
	for _, r := range installed {
		origin := r.Origin()
		if origin == "" {
			origin = "(no recorded source)"
		} else if r.Commit != "" {
			origin += " (" + shortCommit(r.Commit) + ")"
		}
		var notes []string
		if core.IsBuiltinTheme(r.Name) {
			notes = append(notes, "unusable: same name as a built-in theme")
		}
		if local[r.Name] {
			notes = append(notes, "collides with project themes/ — requesting this name is an error")
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
		fmt.Fprintf(tw, "Project (%s)\n", filepath.Join(*projectDir, "themes"))
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
		return fmt.Errorf("theme remove needs the names of the themes to remove")
	}
	store, err := openStore()
	if err != nil {
		return err
	}
	if err := store.Remove(fs.Args()); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "removed: %s\n", strings.Join(fs.Args(), ", "))
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
		printInfoHint(stdout, res)
	}
	if err != nil {
		return err
	}
	if len(results) == 0 {
		fmt.Fprintln(stdout, "nothing to update (no installed theme has a recorded source)")
	}
	return nil
}
