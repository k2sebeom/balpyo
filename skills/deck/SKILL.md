---
name: deck
description: Build a presentation deck with Balpyo — write SOML `.slide` files that render to a self-contained HTML deck and an editable native PPTX from one source. Use when the user asks for slides, a deck, a presentation, a .pptx/PowerPoint/Keynote file, or mentions balpyo, SOML, or `.slide` files.
---

# Authoring a Balpyo deck

Balpyo compiles `.slide` files (SOML — an XML dialect) into two targets from one source: a
self-contained `index.html` presenter and an **editable** native `.pptx`. The author writes
*meaning* (`role="heading"`, `color="accent1"`); the theme owns every value.

Two things make this different from writing HTML, and both are in your favour:

- **The vocabulary is closed.** Every tag, attribute, and value is a finite list. An invented
  name is a build error with a file, line, and column — it never silently renders wrong.
- **The build is the ground truth.** `balpyo check` judges structure *and* measured geometry
  (overflow, off-stage, overlap) in under a second. You do not need to open the deck to know
  whether it is broken.

So: never guess a name, and never declare the deck done on a non-zero exit code.

## 0. Get the binary on PATH

```bash
balpyo version
```

If that fails, install it (one binary, no runtime dependencies) and make sure it is on `PATH`:

```bash
# Go toolchain
go install github.com/k2sebeom/balpyo/cmd/balpyo@latest   # lands in $(go env GOPATH)/bin

# or from a clone
git clone https://github.com/k2sebeom/balpyo && cd balpyo
go build -o balpyo ./cmd/balpyo && sudo mv balpyo /usr/local/bin/
```

From the first `v*` tag onward there are also release archives for `darwin/{arm64,amd64}` and
`linux/{amd64,arm64}` and a Homebrew tap (`brew install k2sebeom/balpyo/balpyo`) — check
GitHub Releases before recommending either. Ask the user to run the install themselves if it
needs `sudo`.

**A `pptx` target and `--check-layout` need Chromium or Chrome installed** — a headless browser
measures every box, and the PPTX uses those exact rects. Without one you get
`지오메트리 패스 실패: 브라우저를 찾을 수 없다`; point `BALPYO_CHROMIUM` at a binary to fix it.
An HTML-only build never launches a browser. If no browser is available, say so and build
`--target html`; do not quietly drop the PPTX the user asked for.

**The theme's fonts must be installed where the build runs.** A measuring build checks every
font token against the browser and warns about each missing one (`테마 "minimal"의 글꼴 "…"가
측정 브라우저에 없다`) — the boxes were measured with a substitute, so PowerPoint's text will not
fit them. `--strict` turns that into an error. The built-in themes use Noto Sans, Noto Sans Mono,
Noto Sans CJK KR and Noto Sans Mono CJK KR (Debian/Ubuntu: `fonts-noto-core`
`fonts-noto-mono` `fonts-noto-cjk`). Install them, or tell the user why the gate fails.

## 1. Outline before markup

Write the outline first — as a table, in your reply, before any file. One row per slide:

| # | id | layout | mode | content |
|---|----|--------|------|---------|

`layout` must come from the 11-name catalogue and `mode` is `slots` or `flow`. This is the
only real design step: picking the layout *is* picking the slide. Doing it after the markup
means rewriting the markup.

Hold to **one idea per slide**. The stage is 1280×720 logical px with a 64px margin, which is
about 20 lines of body text — a slide that needs more is two slides.

## 2. Create or locate the project

For a new deck:

```bash
balpyo init my-talk --author "Name" --theme minimal --lang en --aspect 16:9
```

`--theme` takes a built-in only (`minimal` · `editorial` · `ink`); `minimal` is the right
default when unsure. To use a shared team theme, install it once per machine
(`balpyo theme install github.com/acme/slide-themes@v1`, or a local directory path), then put
its name in `balpyo.json`; `balpyo theme list` shows what is available. `init` refuses to touch a directory that already holds a project — there
is no `--force`.

**Then delete the three scaffold slides before writing your own.** They carry ids `cover`,
`what`, and `end`, and a hand-written slide reusing one collides:
`슬라이드 ID "cover"가 slides/001_title.slide:1:12와 충돌한다`. Read them first if you want to see
the idioms, then remove them.

Slides live in `slides/*.slide` and are **ordered by filename** (`001_`, `002_`, …). Deck-level
facts go in `balpyo.json`: `title` (names the PPTX file), `author`, `lang`, `aspect`, `outDir`,
`theme`, `footer`, `pageNumber`.

## 3. Write the slides

One file per slide. Use the slots a layout declares when the content maps onto them, and flow
otherwise — see `references/vocabulary.md` for both, the complete tag and attribute tables,
and the value vocabularies.

```html
<slide id="kpis" layout="content">
  <text slot="title" role="title">Where we are</text>
  <grid slot="body" cols="3" gap="lg">
    <shape variant="stat-card" type="rounded-rect" fill="surface" w="fill" h="200" pad="lg">
      <text role="title" color="accent1" align="center">+38%</text>
      <p align="center">Monthly active users</p>
    </shape>
  </grid>
  <note>Speaker-only. Never rendered on the slide.</note>
</slide>
```

Standing rules while writing:

- **Look a name up; don't infer it.** Diagnostics list the legal set for that position and,
  for a near miss, suggest one candidate (`알 수 없는 role "heding" — 혹시 "heading"?`) — but
  each guess still costs a round trip.
- **Write meaning, not pixels.** `role=`, `variant=`, and token names (`accent1`, `lg`,
  `surface`) are what make a theme swap work. A literal `#RRGGBB` or `size="31"` is legal but
  it opts that element out of the design system.
- **Never hand-place** with `x`/`y`/`at` outside `layout="canvas"` or a `<stack>`. Let
  containers (`<row>` `<col>` `<grid>`) and the theme's slots do the placing.
- **Don't set `align` on slot children.** The theme owns the horizontal channel there; writing
  `align` fights it.
- `style=` is a web-only escape hatch — it does **not** reach the PPTX. Two targets diverging
  is almost never what the user wants.

## 4. The gate loop — run this after every batch of edits

```bash
balpyo check --project my-talk --check-layout --strict
```

Repeat until it exits 0. Triage in this order:

1. **Errors before warnings** — a parse or vocabulary error means later judgements never ran.
2. **One accident is one warning.** An overflowing list is reported once, on the `<ul>`, with
   the escaped descendants counted at the end (`… (안의 <li> 6개도 함께 벗어난다)`). Fix the box
   the warning names; its children come back with it.
3. **Re-run after each fix.** Geometry is measured, so one change moves every box below it.

Diagnostics are `file:line:col: severity: message`, 1-based, columns counted in runes.
`--strict` promotes warnings to errors — keep it on, since the warnings here are real
(off-stage content is clipped on the web and lands outside the slide in PowerPoint).

For the live loop instead, `balpyo watch --project my-talk` serves `http://127.0.0.1:4600` and
rebuilds in milliseconds on save; a failed build leaves the last good render up with the
diagnostics overlaid.

## 5. Build

```bash
balpyo build --project my-talk --target html,pptx --strict
```

`dist/index.html` has **zero external references** — one file (plus `assets/` when the deck has
images) presents offline. The `.pptx` has real text boxes, real shapes, and theme-colour
references that survive editing in PowerPoint or Keynote. Nothing is written unless the whole
build succeeds.

There is no PDF target: open `index.html` and print to PDF — one slide per page, paper sized to
the stage, with "background graphics" enabled.

## 6. Review when it matters

`balpyo preview` builds a gallery putting each web render beside its PPTX render. The PPTX
column needs a rasterizer: **OnlyOffice Document Builder** first (`BALPYO_ONLYOFFICE`, default
`/opt/onlyoffice/documentbuilder`; it reads OOXML with PowerPoint's defaults), otherwise
LibreOffice (`soffice`) plus `pdftoppm`. Without either it builds a web-only gallery and says
so — unless you passed `--target pptx` explicitly, which then fails before building. Check the
tools before promising a side-by-side. `--gallery` and `--out` may not overlap; `preview`
refuses before building.

There is no pixel diff and no score — `preview` is never a gate. Line breaks and glyphs may
differ between the columns (outside the two targets' contract), and `style=` is web-only.
Look for what *is* in the contract: a missing or misplaced object, a wrong colour, wrong order.

## Reference files

- `references/vocabulary.md` — the complete closed vocabulary: 11 layouts and their slots,
  every tag with its legal attributes, and the value sets. Read it before writing markup.
- `references/pitfalls.md` — what the build will not catch (centred slots, line breaking
  in PPTX), what diagnostics tell you, how to fix overflow, and CLI traps.

Deeper detail lives in the Korean reference docs shipped with this plugin:
`${CLAUDE_PLUGIN_ROOT}/docs/reference/` (one page per tag under `soml/`, plus `appearance.md`,
`layouts.md`, `cli.md`). Worked decks: `${CLAUDE_PLUGIN_ROOT}/examples/decks/` and
`${CLAUDE_PLUGIN_ROOT}/testdata/fixtures/`.
