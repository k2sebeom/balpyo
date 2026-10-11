# AGENTS.md — balpyo

A guide for coding agents working in this repository. Two audiences, two directories:

- `docs/` — the public user documentation (the mkdocs site): overview, guides, reference.
- `contributing/` — for people changing balpyo: `internals/` (architecture, geometry pass,
  testing, release), `decisions/` (ADRs), `backlog.md` (open work).

Docs, ADRs and code comments are written in **Korean** — keep new ones in Korean too. Everything
the program outputs (diagnostics, error messages, CLI text, the `init` template, comments emitted
into generated files) and commit messages are written in **English**.

## What balpyo is

**SOML** (Slide Oriented Markup Language) is an XML-like markup for slides (`.slide` files).
**balpyo** is the Go render engine + CLI that parses SOML, applies a theme, and renders:

- **Web** — one self-contained `index.html` (inlined CSS + viewer runtime; presenter view,
  overview grid, notes, deep links, print-to-PDF). This is the **reference target**.
- **PPTX** — a native, **editable** PowerPoint file (real text boxes, preset shapes, tables,
  placeholders, `schemeClr` theme colors — not images).

The core idea: authors lay out slides with flow layout (`<row>`/`<col>`/`<grid>`/`<stack>`),
a headless browser computes the geometry (the **geometry pass**), and the PPTX renderer bakes
those measured rects into EMU coordinates. Authors never touch coordinates.

**The two targets agree on components and layout, not pixels** (see `contributing/decisions/0064-*`): the same element
becomes the same kind of object, at the measured rect, with the same resolved appearance, in the
same order. Where a line wraps inside a box is each engine's business — PPTX text bodies use
`normAutofit`. Don't add fixes or tests whose goal is glyph/line-break/pixel parity. Themes are
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

- `<slide>` (`layout`, `id`, `skip`, `justify`, `enter`, `transition`, `invert`), `<note>`; `enter` (fade/rise/zoom/wipe) on any content element — native PPTX entrance effects, staggered by the theme's `motion` tokens
- `<text>` (`role`: title/heading/body/caption/code, `variant`, `size`, `color`, `align`, `anchor`, `w`/`min-w`/`max-w`)
- Text subset: `h1`–`h3`, `p`, `ul`/`ol`/`li`, `blockquote`, `pre`, `strong`, `em`, `code`, `a`, `br`, `span`, `mark`; `reveal` for stepwise appearance
- Layout: `row`, `col`, `grid`, `spacer`, `stack` (overlap; `at`, `x`/`y`), plus a `style` escape hatch
- `<shape>` (16 preset types + `type="path"` custom geometry; `fill` also takes `pattern(…)` hatches), `<image>` (`src` — PNG, JPEG or self-contained SVG; `id`, `fit`, `alt`), `<table>`/`tr`/`th`/`td`, `<chart>`/`<series>`/`<point>` (desugared into shapes + text, not a native chart part), `<connector>` (joins two measured boxes by `id`; drawn after the geometry pass)
- Layouts (11): `content`, `title`, `section-divider`, `two-column`, `image-full`, `image-text`, `quote`, `code`, `stat`, `canvas`, `end` — each with named **slots** (`slot="title"` …) that become web `grid-area`s and PPTX `p:ph` placeholders
- Built-in themes: `minimal` (default), `editorial`, `ink` (`core/themes/`). A theme name resolves to
  built-in ∪ project `themes/<name>/` ∪ installed `~/.balpyo/themes/<name>/` (`BALPYO_HOME`); `balpyo.json`
  only ever holds a name — installing from GitHub is `balpyo theme install`, never the build

Authoritative reference: `docs/reference/` (only implemented behavior is documented there).

## CLI

```bash
go run ./cmd/balpyo init my-talk --author "…"                  # scaffold a deck that builds warning-free
go run ./cmd/balpyo build --project my-talk --target html,pptx   # --strict, --check-layout, --theme, --out
go run ./cmd/balpyo check --project my-talk --check-layout --strict   # build without writing (CI gate)
go run ./cmd/balpyo watch --project my-talk                      # rebuild + live reload on http://127.0.0.1:4600
go run ./cmd/balpyo preview --project my-talk                    # side-by-side web/PPTX review gallery
go run ./cmd/balpyo theme install github.com/acme/themes@v1      # install themes into ~/.balpyo (list/remove/update)
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
| `render/route/` | connector routing shared by both renderers (pure functions) | every other balpyo package |
| `render/pptx/` | OOXML assembly, unit conversion (1px = 9525 EMU, pt = px × 0.75) | `core`, `render/web` |
| `internal/pipeline/` | wiring + file writes (`pipeline.Build` — used by both CLI and tests) | — |
| `internal/{preview,watch,scaffold,release,testkit}` | gallery, watch server, `init` template, release, test harness | — |
| `internal/themestore/` | installed-theme home: GitHub tarball fetch, theme discovery, atomic install, `themes.json` | renderers |
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
8. **Component/layout parity**: both targets emit the same component tree — kind, rect, resolved appearance, document order, text runs. Line breaks are outside the contract.

### Adding a tag / vocabulary

Vocabulary must exist in four layers at once: `soml` catalog (`soml/subset.go` `ElementKind`),
`core` value resolution, and both renderers' `RenderedKinds()`. Order: add to `soml` (list it in
`unrenderedKinds` if not rendered yet) → `core` → both renderers, **removing the
`unrenderedKinds` entry in the same commit**. A cross-check test in `package render_test` enforces this.

## Testing

The gate is plain `go test ./...` on the host — no container. The only external tool
is Chromium/Chrome, because the build itself needs it (geometry pass). Tests that need it skip
when it is missing; `BALPYO_TEST_STRICT=1` (set in CI) turns those skips into failures. The
built-in and example themes' Noto fonts must be installed too (`fonts-noto-core`,
`fonts-noto-mono`, `fonts-noto-cjk`) — a measuring build warns about theme fonts the browser
lacks, and the example gate requires zero warnings.

What the suite judges, per fixture (`TestFixtures`) and per example (`TestExamplesBuildClean`):

- **Parity**: the component tree read back from the PPTX XML equals the one derived
  from the deck + measured geometry — kind, rect (EMU), resolved appearance, order, text runs.
- **Goldens**: only `golden/layout.txt` (the tree without rects) and `golden/diagnostics.txt`.
- **PPTX validity** (`testkit.AssertValidPPTX`): every XML part validates against the vendored
  ECMA-376 Transitional XSDs (`internal/testkit/ooxml/`, via `github.com/knroy/go-xml`), plus
  cross-part rules the schema can't express (`crosspart.go`) and package invariants
  (`package.go`). Mutation tests prove the validator still rejects broken parts.
- **Determinism**: two builds of the same input produce the same bytes.

```bash
make test        # go test ./... (the gate; same command CI runs)
make examples    # example decks and themes must build with zero warnings (also part of make test)
make qa          # web | PPTX (OnlyOffice) galleries for every example deck and fixture → preview/qa/
make showcase    # re-render examples/showcase/ (README screenshots + committed html/pptx)
make golden      # regenerate goldens (review the diff; commit with the implementation)
make docs        # mkdocs build --strict (new pages must be in mkdocs.yml nav)
```

- Don't add pixel, ink or line-count assertions. OnlyOffice (preferred) and LibreOffice/poppler are
  only used by `balpyo preview` rasterizing; their tests skip when the tools are absent.
- **Look before you commit a renderer change**: `make qa` renders every example deck and fixture
  side by side (web | PPTX via OnlyOffice) into `preview/qa/`. Parity reads our own XML with our own
  reader; a second engine is how cross-reader defects (wrapping, preset formulas, color spaces) show up.
- Fixtures live in `testdata/fixtures/NN_name/` (`balpyo.json`, `slides/`, `golden/`).
- `examples/` is user-facing: `examples/decks/` (balpyo-intro 12, incident-postmortem 10,
  quarterly-review 8, ec2-intro 13 slides) and `examples/themes/` (standalone themes — `extends` empty:
  harbor, aurora). They are
  gated by a strict zero-warning build, not goldens: every deck with its own theme, and every
  example theme against every deck.
- `core/docs_test.go` reads closed lists and quoted diagnostics from the docs — changing a
  vocabulary or message means updating the docs too.
- Before committing: `gofmt -l .` empty, `go vet ./...`, `go test ./...`.

## Workflow conventions

- Open work lives in `contributing/backlog.md` (unordered). Each item is tests → implementation
  (`soml` → `core` → renderers) → docs; delete the entry when it lands.
- Decisions that change a contract get an ADR in `contributing/decisions/` (Korean; context →
  options → decision → rationale → consequences). **ADRs are never edited after the fact** — a
  later ADR supersedes.
- **Code comments are self-sufficient.** Say the reason in the comment itself; don't cite ADRs,
  docs pages, or planning artifacts (task/iteration/milestone ids). The same goes for test names,
  fixture comments and emitted CSS comments.
- **`docs/` is self-sufficient and user-facing.** It documents implemented behavior only and never
  links to `contributing/`, ADRs, or planning artifacts. Contributor docs may link to `docs/`.
- Diagnostic-wording changes keep the input that produced the message as a fixture.
- Micro commits with `feat:` / `fix:` / `refactor:` / `test:` / `doc:` prefixes, written in English
  (older history is in Korean — don't rewrite it); no AI/author attribution in commit messages.
- Don't commit generated output: `dist/`, `/preview/`, `site/`, `workspace/`. The one exception is
  `examples/showcase/` (README showcase) — regenerate it with `make showcase` and commit it when a
  change alters rendered output.
