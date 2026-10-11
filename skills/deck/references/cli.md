# The `balpyo` command line

Commands: `init` `build` `check` `watch` `preview` `snap` `theme` `version` `help`. Exit codes
are 0 (success) and 1 (anything else; the reason is on stderr). `balpyo` with no arguments or
`balpyo help` prints usage; `balpyo <command> --help` prints that command's flags (and exits 1).
**Precedence: CLI flag > `balpyo.json` > default.**

## `balpyo init [dir]`

Creates a project that builds warning-free (`build --target html,pptx --check-layout --strict`,
given the theme fonts are installed).

| Flag | Default | Meaning |
|---|---|---|
| `--project` | `.` | the directory (same as the positional argument; giving both is an error) |
| `--title` | directory name | deck title — also the PPTX file name |
| `--author` | `Presenter` | presenter |
| `--theme` | `minimal` | **built-in themes only** (`minimal` `editorial` `ink`) |
| `--aspect` | `16:9` | `16:9` or `4:3` |
| `--lang` | `en` | deck language |

Writes `balpyo.json`, `.gitignore` (`dist/`, `preview/`; an existing one is left alone) and
`slides/001_title.slide`, `002_content.slide`, `003_end.slide` (ids `cover`, `what`, `end`).
Refuses a directory that already has `balpyo.json` or a `.slide` file
(`already a balpyo project: …`); there is no `--force`. Exit 1 means nothing was written.

## `balpyo build`

| Flag | Default | Meaning |
|---|---|---|
| `--project` | `.` | project root |
| `--target` | `html` | comma list of `html`, `pptx` |
| `--out` | `outDir` (`dist`) | output directory; relative paths are from the project root |
| `--theme` | `theme` in `balpyo.json` | a theme name: built-in, project `themes/<name>/`, or installed |
| `--strict` | off | warnings become errors |
| `--check-layout` | off | measure the layout and warn about overflow, off-stage, overlap (a `pptx` target measures anyway) |

Output: `dist/index.html` (+ `dist/assets/…` when the deck has images, `dist/themes/<theme>/assets/…`
for theme pictures) and `dist/<title-slug>.pptx` (title lower-cased, every run of other
characters → `-`; `deck.pptx` if nothing is left). Nothing is written unless the whole build
succeeds — a failed build never half-updates the output directory.

**Diagnostics** go to stderr as `file:line:col: severity: message` (1-based, columns in runes,
severity `error` or `warning`), sorted by slide order, line, column; file-level ones first in
their file. Messages without a position print as `warning: …` (missing fonts) or with a config
prefix (`balpyo.json: error: theme.json chrome[1]: …`, `balpyo.json: theme: unknown theme "…" — …`,
`--theme: unknown theme "…" — …`). A failed build ends with `build failed: N errors`.

A wrong name lists the legal set for its position; a near miss (one or two letters off, one for
names of four letters or fewer) adds `— did you mean "x"?`.

**Warnings** (exit 0 unless `--strict`): unknown `variant`; the four layout warnings (with
`--check-layout` or a `pptx` target); a theme font missing from the measuring browser; a
non-layout `style` declaration; a slot-mode slide whose layout has no slot CSS in the theme;
footer/page number turned on in `balpyo.json` but not declared by the theme; chrome leaving
the slide or overlapping other chrome.

**The browser** is used when the target includes `pptx`, with `--check-layout`, and for an
HTML-only build whose deck has a `<connector>`, `cube`, `left-brace` or `right-brace`; a PPTX
with SVG images also uses it for their PNG fallbacks. It is found through `BALPYO_CHROMIUM`, then
`chromium`, `chromium-browser`, `google-chrome` on `PATH`, then the macOS app bundles. Startup
costs about half a second.

**Fonts**: a measuring build checks every font token of the theme (even unused ones — the
whole font set goes into the PPTX theme) and warns per missing family.

## `balpyo check`

The same pipeline as `build` without writing anything. Flags: `--project`, `--theme`,
`--strict`, `--check-layout` (no `--target`, no `--out`). The theme changes the layout, so it
changes the verdict.

```bash
balpyo check --project my-talk --check-layout --strict      # the CI gate
```

Prints a summary: `3 slides · no diagnostics · layout checked`, `… · 1 warning · …`, or
`… · layout not checked (--check-layout)`; on failure `check failed: N errors`. Exit 0 when
there are no errors.

## `balpyo watch`

All `build` flags plus `--addr` (default `127.0.0.1:4600`). `--target` must include `html`
(`watch requires the html target: add html to --target`); with `html,pptx` it rebuilds the PPTX
on every save, which is slower.

- Prints `watching my-talk → http://127.0.0.1:4600  (Ctrl+C to stop)`, then one line per build:
  `✓ 14:31:02  3 slides · 1 file (41ms)` or `✗ 14:31:33  build failed: 1 error (12ms)` with the
  diagnostics.
- The open page reloads after each successful build. A failed build keeps the last good render
  with a diagnostics overlay (bottom-left); a build with only warnings shows a badge that opens
  the list.
- It writes the same output directory as `build`. The reload script is injected into the HTTP
  response only, so `dist/index.html` on disk stays the deployable file.
- `--check-layout` keeps one browser running for the whole session.
- It polls for changes every 250ms (size + modification time); an edit that keeps both is missed —
  save again. It skips the output directory and dot-directories and does not watch installed
  themes (after `balpyo theme update`, save a project file or restart).
- `GET /_balpyo/status` returns the last build as JSON (`ok`, `error`, `diagnostics[]` with
  `severity` `file` `line` `column` `message`, `slides`, `builtAt`); `/_balpyo/events` streams it.
- It does not open a browser for you. Ctrl+C exits 0.

## `balpyo snap`

Draws slides to PNG (and GIF) with a builtin renderer — **no browser, no office suite**.

| Flag | Default | Meaning |
|---|---|---|
| `--project` | `.` | project root |
| `--theme` | from `balpyo.json` | theme name |
| `--out` | `<outDir>/snap` | output directory (a relative path is from the **current directory**) |
| `--slides` | all | 1-based list and ranges: `1,3-5` |
| `--scale` | `1` | pixel scale (`2` for small text) |
| `--gif` | off | also write `deck.gif`, one 1.5s frame per slide and per `reveal` step |
| `--outline` | off | draw every element's box, coloured by depth, labelled with its `data-balpyo-id` |
| `--strict` | off | layout warnings fail the run (images are still written) |

Files are named by slide number (`--slides 4` writes `004.png`) at stage size × scale. Every
`reveal` step is shown; every `enter` is in its final state. It prints the same layout warnings
as `--check-layout`, measured by the builtin renderer, and ends with
`N images → dir (620ms, builtin renderer)`. A slide number past the end is an error
(`slide 9 does not exist (the deck has 3)`). Fonts come from the system with silent
substitution; colour emoji and text inside SVG images are not drawn.

## `balpyo preview`

Builds a review gallery: each slide's web render beside its PPTX render.

| Flag | Default | Meaning |
|---|---|---|
| `--project` | `.` | project root |
| `--out` | `outDir` | build output |
| `--gallery` | `<project>/preview` | gallery directory — **cleared on every run**; a relative path is from the current directory |
| `--theme` | from `balpyo.json` | theme name |
| `--target` | `html`, plus `pptx` when a rasterizer exists | columns |
| `--strict` | off | warnings become errors |

Layout is always measured (a browser is required). The PPTX column needs OnlyOffice Document
Builder (`BALPYO_ONLYOFFICE`, default `/opt/onlyoffice/documentbuilder`) or LibreOffice
`soffice` + `pdftoppm` (`BALPYO_SOFFICE`, `BALPYO_PDFTOPPM`; on macOS
`brew install --cask libreoffice; brew install poppler`). Without them it prints
`dropping the PPTX column — …` and builds a web-only gallery, unless `--target pptx` was given
explicitly, which fails. `--gallery` and `--out` may not overlap. It prints
`N slides → …/preview/index.html` and copies the built files next to the gallery. Exit 0 means
the gallery was made — it is never a gate.

Comparing themes: `balpyo preview --theme ink --gallery preview-ink`.

## `balpyo theme`

| Command | Does |
|---|---|
| `theme install <github.com/owner/repo[/sub/path][@ref] \| path> [name...] [--force]` | install themes into `~/.balpyo/themes/` (`$BALPYO_HOME/themes/`); with names, only those plus their same-source `extends` parents |
| `theme list [--project .]` | built-in, installed (with source and commit), and project themes; flags name collisions |
| `theme info <name> [--project .]` | where the theme comes from, its parent (`extends acme — see: balpyo theme info acme`), then its `README.md` |
| `theme update [name...]` | re-fetch / re-copy from the recorded source |
| `theme remove <name...>` | remove (refuses while another installed theme extends it) |

Installing fetches a public GitHub tarball (no `git` needed); private repositories and other
hosts: check them out and install the directory by path (a copy, not a link). Builds never use
the network — on another machine or in CI, run the same `balpyo theme install` first. The theme
skill covers designing and publishing themes.

## `balpyo version`

Three lines — `balpyo vX` (or `balpyo (devel)`), the commit, and `os/arch · go version`. Paste
all three into bug reports.

## Environment variables

| Variable | Meaning |
|---|---|
| `BALPYO_CHROMIUM` | path to the Chromium/Chrome binary |
| `BALPYO_CHROMIUM_NO_SANDBOX=1` | launch the browser without its sandbox (root, containers) |
| `BALPYO_HOME` | where installed themes live (default `~/.balpyo`) |
| `BALPYO_ONLYOFFICE` | OnlyOffice Document Builder directory, for `preview` |
| `BALPYO_SOFFICE`, `BALPYO_PDFTOPPM` | LibreOffice and poppler binaries, for `preview` |

## Presenting `index.html`

- **Navigate**: → ← (also ↓ ↑, Space, PageDown/PageUp, the ‹ › buttons); Home / End jump to the
  first / last slide. "Next" uses up a slide's `reveal` steps before moving on; "back" rewinds
  one step.
- **Notes**: `n` (or the Notes button) shows the current slide's `<note>`.
- **Overview**: `o` (or Grid) shows every slide on one screen; click a tile to go there; `Esc`
  returns.
- **Deep links**: `index.html#/<slide id>` or `#/<number>` (1-based); the URL follows
  navigation.
- **Offline**: one self-contained file (plus `assets/` for images).
- **Print to PDF**: Cmd/Ctrl+P → Save as PDF, from any view. One slide per page, paper sized to
  the stage (16:9 → 960×540pt), margins already zero; enable "background graphics". Controls and
  the notes panel are not printed. For handouts with notes, export the PPTX from PowerPoint or
  Keynote.
