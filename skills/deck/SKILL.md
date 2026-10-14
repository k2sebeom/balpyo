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

This skill and its reference files are complete on their own — everything the tool accepts is
written here. When something here disagrees with the binary, the binary is right: its
diagnostics always list the legal set for the position you got wrong.

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

Every `v*` release also has archives (`.tar.gz` for `darwin/{arm64,amd64}` and
`linux/{amd64,arm64}`, a `.zip` holding `balpyo.exe` for `windows/amd64`; each holds just the binary;
verify with `sha256sum -c SHA256SUMS`, or on Windows compare `Get-FileHash` with the archive's line in
`SHA256SUMS`) and, for Linux and macOS, an install script that fetches and verifies one
(`curl -fsSL https://raw.githubusercontent.com/k2sebeom/balpyo/main/install.sh | sh`) — prefer the
script. Ask the user to run the install themselves if it needs `sudo`.

**A `pptx` target and `--check-layout` need Chromium or Chrome installed** — a headless browser
measures every box, and the PPTX uses those exact rects. The binary looks at `BALPYO_CHROMIUM`,
then `chromium`, `chromium-browser`, `google-chrome` on `PATH`, then the macOS app bundles. On
Windows it looks for `chrome`, `chromium`, `msedge` on `PATH`, then Chrome's and Edge's install
folders (under `%ProgramFiles%`, `%ProgramFiles(x86)%`, `%LocalAppData%`) — Edge ships with
Windows, so a Windows machine normally has one. Without one you get
`geometry pass failed: no browser found (tried: chromium, …) — set its path with BALPYO_CHROMIUM`.
Running as root or in a container, the browser's sandbox may refuse to start: set
`BALPYO_CHROMIUM_NO_SANDBOX=1`.

An HTML-only build launches a browser only when the web output itself needs measured boxes:
a `<connector>` (its ends come from the measured boxes) or a `cube` / `left-brace` / `right-brace`
shape (drawn from the measured box). A PPTX with SVG images also uses the browser to render
their PNG fallbacks. If no browser is available, say so and build `--target html` (with no
connectors or measured-box shapes); do not quietly drop the PPTX the user asked for.
`balpyo snap` (§4) needs no browser at all — it is how you still *see* the slides on a
machine without one.

**The theme's fonts must be installed where the build runs.** A measuring build checks every
font token against the browser and warns about each missing one —
`warning: theme "minimal" font "Noto Sans" (heading.latin, body.latin) is missing from the measuring browser — …`
— because the boxes were measured with a substitute, so PowerPoint's text will not fit them.
`--strict` turns that into an error. The built-in themes use Noto Sans, Noto Serif (editorial
headings), Noto Sans Mono, and their CJK KR variants (Debian/Ubuntu: `fonts-noto-core`
`fonts-noto-mono` `fonts-noto-cjk`). Install them, or tell the user why the gate fails.

## 1. Outline before markup

Write the outline first — as a table, in your reply, before any file. One row per slide:

| # | id | layout | mode | content |
|---|----|--------|------|---------|

`layout` must come from the 11-name catalogue (`references/vocabulary.md`) and `mode` is `slots`
or `flow`. This is the only real design step: picking the layout *is* picking the slide. Doing
it after the markup means rewriting the markup.

Hold to **one idea per slide**. The stage is 1280×720 logical px (960×720 at 4:3) with a 64px
margin, which is about 20 lines of body text — a slide that needs more is two slides.

## 2. Create or locate the project

For a new deck:

```bash
balpyo init my-talk --author "Name" --theme minimal --lang en --aspect 16:9
```

It writes `balpyo.json`, a `.gitignore` (`dist/`, `preview/`), and three slides, and the result
builds warning-free. `--theme` takes a built-in only (`minimal` · `editorial` · `ink`);
`minimal` is the right default when unsure. `init` refuses a directory that already holds
`balpyo.json` or a `.slide` file — there is no `--force`. For a Korean deck pass `--lang ko`:
the web breaks Korean at word boundaries only then, and with `en` words break between
syllables.

```text
my-talk/
  balpyo.json
  slides/
    001_title.slide        # one slide per file; filename order = slide order
    010_problem.slide      # leave gaps (001, 010, 020) so inserting is one new file
    02_deep-dive/          # a subdirectory is a chapter, ordered with the files
      010_detail.slide
    _draft.slide           # a leading _ means ignored
  assets/                  # pictures for <image src="assets/…"> (a convention, not a rule)
  themes/<name>/           # optional project-local theme (see the theme skill)
```

`balpyo.json` holds deck-level facts. Every key is optional and **an unknown key is an error**:

| Key | Default | Meaning |
|---|---|---|
| `title` | directory name | deck title; also names the PPTX file (lower-cased, non-alphanumerics → `-`: `my-talk.pptx`) |
| `author` | — | presenter |
| `lang` | `en` | a BCP 47 tag (`ko`, `en`, `ja`, `ko-KR` …): `<html lang>` and the PPTX text language; `ko` makes the web break lines at word boundaries. Not the code language of `<pre lang>` |
| `aspect` | `16:9` | `16:9` (1280×720) or `4:3` (960×720) |
| `theme` | `minimal` | a theme **name** — built-in, project `themes/<name>/`, or installed; never a path |
| `outDir` | `dist` | build output, relative to the project root |
| `footer` | `""` | footer text; the theme decides where it sits and which layouts skip it |
| `pageNumber` | `false` | turn page numbers on |
| `slidesDir` | `slides` | where the slides live — a relative path inside the project |

To use a shared team theme, install it once per machine
(`balpyo theme install github.com/acme/slide-themes@v1`, or a local directory path), then put
its name in `balpyo.json`; `balpyo theme list` shows what is available. If the theme ships an
icon or illustration library under its `assets/`, point at it with `<image src="theme:icons/…">`
— never copy the files into the deck or spell out `themes/<name>/…`.

**Read the theme before writing a slide:**

```bash
balpyo theme info <theme> --project my-talk
```

It prints the theme's `README.md`, where the theme's author says what decks the theme suits, what
each variant is for, and which `theme:` assets it ships and why. Treat that as the design brief.
Use variants and icons for the purpose the README gives them, follow its layout advice, and
prefer what it recommends over generic habits from this skill. If the README says the theme fits
a different kind of deck than the one the user asked for, tell the user before writing. When
the theme `extends` another, the second line names the parent: read the parent's README too
(`balpyo theme info <parent>`) for anything the child does not cover. A theme without a README
prints `this theme has no README.md`; then an unknown `variant` warning lists the variant names
the theme has, and an installed theme's files are readable under `~/.balpyo/themes/<name>/`.
What the three built-in themes look like is summarised in `references/appearance.md`.

**Then delete the three scaffold slides before writing your own.** They carry ids `cover`,
`what`, and `end`, and a hand-written slide reusing one collides:
`slide ID "cover" collides with slides/001_title.slide:2:12 (give one of them a different id, or remove the slide you do not need)`.
Read them first if you want to see the idioms, then remove them.

## 3. Write the slides

One file per slide, one root `<slide>` per file. Use the slots a layout declares when the
content maps onto them, and flow otherwise.

```html
<slide id="kpis" layout="content">
  <text slot="title" role="title">Where we are</text>
  <grid slot="body" cols="3" gap="lg">
    <shape variant="stat-card" type="rounded-rect" w="fill" h="200" pad="lg">
      <text role="title" color="accent1" align="center">+38%</text>
      <p align="center">Monthly active users</p>
    </shape>
  </grid>
  <note>Speaker-only. Never rendered on the slide.</note>
</slide>
```

Standing rules while writing:

- **Look a name up; don't infer it.** Diagnostics list the legal set for that position and,
  for a near miss, suggest one candidate
  (`unknown role "heding" — did you mean "heading"? (available roles: title, heading, body, caption, code)`)
  — but each guess still costs a round trip.
- **Write meaning, not pixels.** `role=`, `variant=`, and token names (`accent1`, `lg`,
  `surface`) are what make a theme swap work. A literal `#RRGGBB` or `size="31"` is legal but
  it opts that element out of the design system (and out of PowerPoint's theme-colour editing).
- **Never hand-place.** `x`/`y` are legal only on direct children of a `layout="canvas"` slide
  and on children of `<stack>`; `at` only on children of `<stack>`. Let containers
  (`<row>` `<col>` `<grid>`) and the theme's slots do the placing.
- **Don't set `align` on slot children to centre them.** The theme owns the horizontal channel
  in slots and centres the ones meant to be centred (the cover, a `stat` number); writing
  `align` there fights it. `align="end"` on a full-width slot such as a quote's attribution is
  fine.
- **`style=` is a layout-only escape hatch.** Layout declarations (`flex`, `grid-*`, `width`,
  `margin`, …) are measured, so they reach the PPTX; anything else (`background`, `color`,
  `border`, `z-index`, …) is web-only and a warning. Reach for it last.
- **Order is meaning.** Document order is paint order (later on top), PPTX shape order, and
  reading order for screen readers. `order`, `at`, `x`/`y` change only the picture.

Diagrams, charts and motion all become native, editable PowerPoint objects. Reach for these
before faking them with images or hand-placed rectangles:

- **Cards** → a `<shape>` holding a `<col>` is a *backdrop*: its children become their own
  shapes laid over it. A shape holding only text is one text-bearing shape.
- **Lines between boxes** → give the boxes an `id` and add `<connector from to>` as a direct child
  of `<slide>`. It takes no space in the flow, so lay the boxes out first and then add the lines.
  It becomes a real PowerPoint connector glued to the boxes.
- **Links to another slide** → `<a href="#<slide id>">` (or `#/<number>`); a PPTX slide jump too.
- **Curves, steps, filled areas, odd silhouettes** → `<shape type="path" d="…">` in a 0–100 box;
  overlay several in a `<stack>`. Hatching is `fill="pattern(…)"`.
- **Data** → `<chart>` (`bar` / `line` / `area` / `step`, several `<series>` with a legend).
- **Tables** → `<table>`, a real editable PowerPoint table, with `colspan`/`rowspan`.
- **Code** → `<pre lang="go|xml|soml|json|bash">`, highlighted in theme colours.
- **Icons** → the theme's own icons first (`theme:icons/…`, as its README describes them), else SVG
  files via `<image src="assets/x.svg">`.
- **Overlap on purpose** (badges, watermarks, captions on pictures) → `<stack>` with `at`.
- **A dark slide mid-deck** → `<slide invert>`, not literal colours.
- **Highlighted words / a big number with a small unit** → `<mark>` and `<span size="…">` (smaller only).
- **Stepwise lists** → `<li reveal>` (web presenter only); **motion** → `enter="rise"` (or
  `fade` / `zoom` / `wipe`) on `<slide>` staggers its children; `enter="draw"` on a connector
  draws it in; `transition="fade|push"` on `<slide>`. Use motion to build up an argument, not on
  every slide. It never changes layout, so it cannot break the gate.

The per-element rules — what each tag may contain, every attribute's grammar, and a working
snippet for each feature — are in `references/elements.md`. Read the section for an element
before using it for the first time.

## 4. The gate loop — run this after every batch of edits

```bash
balpyo check --project my-talk --check-layout --strict
```

Repeat until it exits 0. Triage in this order:

1. **Errors before warnings** — a parse or vocabulary error means later judgements never ran.
2. **One accident is one warning.** An overflowing list is reported once, on the `<ul>`, with
   the escaped descendants counted at the end (`… (5 <li> inside escape too)`). Fix the box
   the warning names; its children come back with it.
3. **Re-run after each fix.** Geometry is measured, so one change moves every box below it.

Diagnostics are `file:line:col: severity: message` (severity `error` or `warning`), 1-based,
columns counted in runes, sorted by slide order, line, column — the first one printed is the
first to fix. A clean run prints a summary like `3 slides · no diagnostics · layout checked`;
a failed one ends `check failed: N errors`. `--strict` promotes warnings to errors — keep it
on, since the warnings here are real (off-stage content is clipped on the web and lands outside
the slide in PowerPoint). What each layout warning means and how to fix it is in
`references/pitfalls.md`.

For the live loop instead, `balpyo watch --project my-talk` serves `http://127.0.0.1:4600` and
rebuilds on save; a failed build leaves the last good render up with the diagnostics overlaid.

### Then look at the slides

A clean gate means nothing overflows; it does not mean the slide reads well. Draw the slides
and read the images yourself:

```bash
balpyo snap --project my-talk --slides 3-5        # dist/snap/003.png … in well under a second
```

Open the PNGs (stage size — 1280×720 at 16:9; `--scale 2` for small text) and check what the gate cannot:
hierarchy, balance, a chart that says the right thing, a card that is mostly empty. `snap` needs
no browser, OnlyOffice or LibreOffice — a builtin renderer lays out the deck's own HTML/CSS and
paints it. It also prints the same layout warnings as `--check-layout`, measured by that
renderer. Every `reveal` step is shown and every `enter` is drawn in its final state.

When a warning names a box you cannot place, `--outline` draws every element's measured box
with its id, the `data-balpyo-id` in `index.html`. Ids count from 0 (`s3.1.0` is the 4th
slide → its 2nd child → that child's 1st child), while `--slides` and the PNG names count
from 1. `--gif` writes `deck.gif` with one frame per slide and per `reveal` step.

`snap` is a preview, not the gate: its boxes match the browser's, but its line breaks and glyphs
are not promised to, and a missing theme font is substituted silently. The verdict is still
`balpyo check --check-layout --strict`.

## 5. Build and present

```bash
balpyo build --project my-talk --target html,pptx --strict
```

`dist/index.html` has **zero external references** — one file (plus `assets/` when the deck has
images) presents offline. The `.pptx` has real text boxes, real shapes, real tables, speaker
notes, placeholders for slotted text (a slotted chart, or a shape with no text of its own —
empty, or a backdrop around a container — stays a plain shape at the slot's position), and
theme-colour references that survive editing in PowerPoint or Keynote. Nothing is written unless the whole build succeeds.

Tell the user how to present `index.html`: ←/→ (also Space, PageUp/PageDown, ↑/↓) step through
slides and `reveal` steps, Home/End jump, `n` opens the speaker notes, `o` opens the overview
grid, and `index.html#/<slide id>` (or `#/<number>`) deep-links. There is no PDF target: open
`index.html` and print to PDF — one slide per page, paper sized to the stage, with "background
graphics" enabled. Details: `references/cli.md`.

## 6. Review when it matters

`balpyo snap` (§4) is the quick look at the web layout. `preview` is the slower, two-column
review: it is the only way to see the **PPTX** drawn by another engine.

`balpyo preview` builds a gallery putting each web render beside its PPTX render. The PPTX
column needs a rasterizer: **OnlyOffice Document Builder** first (`BALPYO_ONLYOFFICE`, default
`/opt/onlyoffice/documentbuilder`; it reads OOXML with PowerPoint's defaults), otherwise
LibreOffice (`soffice`) plus `pdftoppm`. Without either it builds a web-only gallery and says
so — unless you passed `--target pptx` explicitly, which then fails before building. Check the
tools before promising a side-by-side. `--gallery` and `--out` may not overlap; `preview`
refuses before building.

There is no pixel diff and no score — `preview` is never a gate. Line breaks and glyphs may
differ between the columns (outside the two targets' contract), and non-layout `style=` is
web-only. Look for what *is* in the contract: a missing or misplaced object, a wrong colour,
wrong order.

## Reference files

- `references/vocabulary.md` — the closed vocabulary at a glance: 11 layouts and their slots,
  every tag with its legal attributes and children, value sets, sizes, tokens, colour
  functions, pattern fills. Read it before writing markup.
- `references/elements.md` — element by element: slide and canvas, notes, text and inline
  markup, lists and `reveal`, code blocks, containers and `stack`, shapes and paths, images,
  tables, charts, connectors, motion, inverted slides, `style`. Each with the rules the
  diagnostics enforce and snippets to copy.
- `references/appearance.md` — how a value is decided (the cascade), the built-in themes'
  roles, tokens, variants and layout placement, and what reaches the PPTX as a theme reference.
- `references/pitfalls.md` — what the build will not catch (line breaking in PPTX, centred
  slots), every layout warning and its fix, and CLI traps.
- `references/cli.md` — every command, flag, environment variable, output line and exit code,
  plus the presenter view and printing.
