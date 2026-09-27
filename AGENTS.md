# AGENTS.md — balpyo

A guide for coding agents working in this repository. For the full product definition see
`PRD.md`; for what to work on next see `TASKS.md`; for the design rationale see
`docs/decisions/` (ADRs). Project docs, ADRs, diagnostics and commit subjects are written in
**Korean** — keep new ones in Korean too.

## What balpyo is

**SOML** (Slide Oriented Markup Language) is an XML-like markup for slides (`.slide` files).
**balpyo** is the Go render engine + CLI that parses SOML, applies a theme, and renders:

- **Web** — one self-contained `index.html` (inlined CSS + viewer runtime; presenter view,
  overview grid, notes, deep links, print-to-PDF). This is the **reference target**.
- **PPTX** — a native, **editable** PowerPoint file (real text boxes, preset shapes, tables,
  placeholders, `schemeClr` theme colors — not images).

The core idea: authors lay out slides with flow layout (`<row>`/`<col>`/`<grid>`/`<stack>`),
a headless browser computes the geometry (the **geometry pass**), and the PPTX renderer bakes
those measured rects into EMU coordinates. Authors never touch coordinates. Themes are
**design systems** (tokens + component defaults + variants + layout CSS), compiled to both CSS
and the OOXML theme.

PDF rendering is out of scope (browser print covers it).

## A SOML project

```
my-talk/
  balpyo.json            # title, author, lang, aspect (16:9 | 4:3), theme, outDir, footer, pageNumber
  slides/
    001_title.slide      # file order = slide order (numeric prefix is sorting only)
    chapter/…            # subdirectories are chapters; `_`-prefixed files are ignored
  assets/…               # images referenced by <image src>
  themes/<name>/         # optional local theme (theme.json, tokens.json, components.json, layouts/*.css)
```

```xml
<slide id="cover" layout="title">
  <text slot="kicker" variant="kicker">SLIDE ORIENTED MARKUP LANGUAGE</text>
  <h1 slot="title">balpyo</h1>
  <text slot="subtitle" role="heading" color="muted">Write once, get web + editable PPTX</text>
  <note>Speaker notes go here.</note>
</slide>
```

Vocabulary (closed — unknown tags/attributes are diagnostics):

- `<slide>` (`layout`, `id`, `skip`, `justify`), `<note>`
- `<text>` (`role`: title/heading/body/caption/code, `variant`, `size`, `color`, `align`, `anchor`, `w`/`min-w`/`max-w`)
- Text subset: `h1`–`h3`, `p`, `ul`/`ol`/`li`, `blockquote`, `pre`, `strong`, `em`, `code`, `a`, `br`, `span`; `reveal` for stepwise appearance
- Layout: `row`, `col`, `grid`, `spacer`, `stack` (overlap; `at`, `x`/`y`), plus a `style` escape hatch
- `<shape>` (12 preset types), `<image>` (`src`, `fit`, `alt`), `<table>`/`tr`/`th`/`td`, `<chart>`/`<point>` (desugared into shapes + text, not a native chart part)
- Layouts (11): `content`, `title`, `section-divider`, `two-column`, `image-full`, `image-text`, `quote`, `code`, `stat`, `canvas`, `end` — each with named **slots** (`slot="title"` …) that become web `grid-area`s and PPTX `p:ph` placeholders
- Built-in themes: `minimal` (default), `editorial`, `ink` (`core/themes/`)

Authoritative reference: `docs/reference/` (only implemented behavior is documented there).

## CLI

```bash
go run ./cmd/balpyo init my-talk --author "…"                  # scaffold a deck that builds warning-free
go run ./cmd/balpyo build --project my-talk --target html,pptx   # --strict, --check-layout, --theme, --out
go run ./cmd/balpyo check --project my-talk --check-layout --strict   # build without writing (CI gate)
go run ./cmd/balpyo watch --project my-talk                      # rebuild + live reload on http://127.0.0.1:4600
go run ./cmd/balpyo preview --project my-talk                    # side-by-side web/PPTX review gallery
```

Precedence: CLI flag > `balpyo.json` > default. `--check-layout` reports overflow / off-stage /
overlap warnings from measured geometry; `--strict` promotes warnings to errors.

## Architecture

```
balpyo.json + slides/*.slide → soml.Collect → soml parse → deck model (soml.Deck)
  → core.Resolve (appearance cascade) → core.ResolveAssets → core.Validate
  → render/web (HTML)  ─┬→ internal/pipeline writes files
  → render/geom (Chromium via CDP measures the HTML) → render/pptx (OOXML)
```

| Package | Owns | Must not import |
|---|---|---|
| `soml/` | lexer, parser (lossless trivia), deck model, element catalog, diagnostics — **public API** | any other balpyo package |
| `core/` | config, appearance cascade, asset resolution, cross-file validation, themes | renderers |
| `render/` | target contract (`Target`, `Renderer`, `Files`) | `core` |
| `render/web/` | HTML + generated CSS, element ID contract, embedded viewer runtime | `core`, `render/pptx` |
| `render/geom/` | geometry pass (CDP), geometry side table | `core`, `render/pptx` |
| `render/pptx/` | OOXML assembly, unit conversion (1px = 9525 EMU, pt = px × 0.75) | `core`, `render/web` |
| `internal/pipeline/` | wiring + file writes (`pipeline.Build` — used by both CLI and tests) | — |
| `internal/{preview,watch,scaffold,release,testkit}` | gallery, watch server, `init` template, release, test harness | — |
| `cmd/balpyo/` | CLI surface only (flags, output) | build logic |

Invariants to preserve:

1. **Appearance is resolved once**, in `core.Resolve` (inline > variant > component default > role/token).
   Renderers read only resolved values (`Element.Appearance`), never re-interpret `Attrs`.
2. **Renderers are pure**: deck model in, `map[path]bytes` out. Only `core.ResolveAssets` reads files.
3. **One coordinate system**: logical stage px (16:9 = 1280×720, 4:3 = 960×720). EMU/pt only at the OOXML boundary.
4. **Theme tokens stay names** to the renderer boundary (`var(--balpyo-color-*)` / `a:schemeClr`).
5. **Geometry is a side table** (`geom.Geometry`) injected into the PPTX renderer, not stored in the deck.
6. **Determinism**: same input → same bytes (zip timestamps fixed to 1980-01-01, ordered lists not maps).
7. **Diagnostics come from one place** (parser for in-file, `core.Validate` for cross-file) with `line:col`, shared by `build`/`check`/`watch`.

### Adding a tag / vocabulary

Vocabulary must exist in four layers at once: `soml` catalog (`soml/subset.go` `ElementKind`),
`core` value resolution, and both renderers' `RenderedKinds()`. Order: add to `soml` (list it in
`unrenderedKinds` if not rendered yet) → `core` → both renderers, **removing the
`unrenderedKinds` entry in the same commit**. A cross-check test in `package render_test` enforces this.

## Testing

Tests must run in the pinned Docker image (Chromium, LibreOffice, poppler, Noto fonts only,
hinting off) — pixel and text-metric comparisons are only meaningful in a deterministic environment.

```bash
make test        # full suite in container (the real gate; also what CI runs)
make test-host   # go test ./... on host — skips Chromium/LibreOffice tests (not a real green)
make examples    # examples/* must build with zero warnings
make review      # web/PPTX/diff gallery → review/index.html
make golden      # regenerate goldens (review the diff; commit with the implementation)
make docs        # mkdocs build --strict (new pages must be in mkdocs.yml nav)
```

- Fixtures live in `testdata/fixtures/NN_name/` (`balpyo.json`, `slides/`, `golden/` with
  `diagnostics.txt`, `index.html`, `pptx/`).
- `examples/` (balpyo-intro 12, incident-postmortem 10, quarterly-review 6 slides) are
  regression decks gated by a strict zero-warning build, not goldens.
- `core/docs_test.go` reads closed lists and quoted diagnostics from the docs — changing a
  vocabulary or message means updating the docs too.
- Before committing: `gofmt -l .` empty, `go vet ./...`, `go test ./...`, `make test`.

## Workflow conventions

- Follow `TASKS.md` in order; each item is RED (tests incl. error/warning cases) → GREEN
  (`soml` → `core` → renderers) → REVIEW (`make review`, open the PPTX) → DOC. Tick its
  checkboxes when done.
- New design decisions get an ADR in `docs/decisions/` (Korean; context → options → decision →
  rationale → consequences). **ADRs are never edited after the fact** — a later ADR supersedes.
- `docs/reference/` documents implemented behavior only.
- Diagnostic-wording changes keep the input that produced the message as a fixture.
- Micro commits with `feat:` / `fix:` / `refactor:` / `test:` / `doc:` prefixes; no AI/author
  attribution in commit messages.
- Don't commit generated output: `dist/`, `review/`, `/preview/`, `site/`, `.make/`, `.scratch/`, `workspace/`.
