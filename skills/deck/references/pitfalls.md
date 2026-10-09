# Pitfalls and known differences

Rough edges measured by writing real decks, with the workaround for each. These are current
facts about the tool, not style advice — the ones under "the build will not catch this" are
the reason a deck still deserves one human look.

## The build will not catch this

### Korean (CJK) line breaking diverges between the two targets

Chromium breaks Korean **between syllables**; LibreOffice breaks only at **word boundaries**.
The geometry pass measures the browser, so a paragraph that exactly fits its shape on the web
can take one more line in PowerPoint — and that line **spills outside the shape**. Measured
at 8% slack, so slack does not save it. `check --check-layout --strict` still **exits 0**.

What to do when the deck has CJK text in sized boxes:

- Leave real vertical room in `<shape h="…">` — budget one extra line (body 24px ⇒ ~36px).
- Keep paragraphs inside narrow shapes short; a card is a label, not a paragraph.
- Open the PPTX, or run `balpyo preview`, before calling a CJK deck done.

### A centred slot has only 4% slack

Where a theme centres a slot, grid shrinks the box to the **content width**, so there is no
room left for the viewer's own typesetting. A one-line text there can wrap in LibreOffice when
the web shows one line. The centred slots in the built-in themes are: `minimal` → `title`,
`stat`, `end`; `editorial` → `stat`; `ink` → `stat`, `section-divider`.

**Keep those lines short** — one number, one headline phrase. If the text has to be a sentence,
pick a layout that does not centre.

### Script spacing around punctuation (LibreOffice only)

Opening the PPTX in LibreOffice shows a gap at every CJK↔Latin boundary — `선언으로 , 색과` —
about ¼ em each. It is the viewer's CJK typesetting rule (JIS X 4051), it is not in the file,
and DrawingML has no attribute to turn it off. **Nothing to do**: boxes, coordinates, and line
counts match the web, and PowerPoint and Keynote may not do it at all. It is also why
`balpyo preview` diff percentages **overstate** real divergence.

### Other viewer-side differences

| Difference | Where | Size |
|---|---|---|
| Baselines sit 2–5px lower | LibreOffice | +2–5px |
| Pure-Korean lines render ~1.2% narrower | LibreOffice | −1.2% |

## Layout defaults that make a weak slide

| Layout | What happens | Workaround |
|---|---|---|
| `quote` | The quote renders at **body size** — the layout has no quote typography yet, and all three themes give `blockquote` only `color: muted`. The slide reads as unfinished. | `variant="lead"` on the quote (passes `--strict`) |
| `stat` | The `label` (18px) is **smaller than** `body` (24px), inverting the hierarchy | Set `size` on `label`, or move the sentence out of `body` |
| `code` | The caption sits against the footer band — same left edge, same muted colour, so it reads as a two-line footer (not an overlap) | Shorten or drop the caption, or set a different `color` |
| `title` | A large gap between the title block and the author/date lines | Accept it, or go flow mode with `justify` |

## Diagnostics: what they will and will not tell you

- **No "did you mean".** Real typos cost a full round trip: `img`→`image`,
  `two-colum`→`two-column`, `heding`→`heading`, `elipse`→`ellipse`. Look names up in
  `references/vocabulary.md` instead of typing from memory.
- **An unknown *tag* lists the whole 31-tag global vocabulary**, not the tags legal at that
  position — so inside `<ul>` you get 31 names when the answer is `<li>`. Slot and attribute
  diagnostics *do* narrow correctly. Child-position errors (`<ul>` holding a `<p>`) narrow too.
- **Diagnostics are not sorted by source position.** Across several files they can arrive
  `002:3` · `002:5` · `001:2`. Read the whole list before you start editing.
- **A slide-ID collision has no `line:col`** — the one diagnostic that does not. It names the
  other file: `슬라이드 ID "cover"가 slides/001_cover.slide와 충돌한다`. Delete or rename.
- **Duplicate content in one slot** surfaces as a geometry overlap warning
  (`<text>가 형제 <text>와 95px×36px 겹친다`), not a vocabulary error. Two elements with the
  same `slot=` is the usual cause.

## One accident, many warnings

A list overflowing the stage by 279px reports **7 warnings**: one on `<ul>` and one on each
`<li>` past the edge. Content-box overflow collapses into a single warning, but **stage
overflow is reported by the container and every descendant**.

So triage containers first and re-run. Reading the deepest warning first means fixing six
`<li>`s that were never the problem.

## Fixing overflow, in the order to try it

For `<p>가 부모 <shape>를 아래로 34px 넘친다` (a sized box that cannot hold its content):

1. **Grow the box** — `h="140"` → `h="200"`. The height needed is content plus `pad` twice
   (`pad="lg"` is 36px top and bottom).
2. **Shrink the type** — `size`, or a smaller `variant`. Changing size twice on one slide is
   usually a signal that there is too much content.
3. **Cut the words.** A slide is not a document.

For `<text>가 콘텐츠 박스를 넘친다` (a slide-level element eating the 64px margin): split the
slide, turn it sideways with `<grid>`/`<row>`, or reduce `size`. If using the margin is the
design, `layout="canvas"` is the sanctioned way — it is never warned about.

**Stage overflow has no exception**, even with explicit `x`/`y`: the web clips it and the PPTX
puts the shape off-slide, so the two targets break differently. Build an off-screen effect
with a `<stack>` and sizes inside the stage instead.

Intentional overlap is not warned about either: elements using `at`/`x`/`y` (and their
descendants), anything on `layout="canvas"`, and children of `<stack>`. Table cells are exempt
from containment judgement — the browser's table algorithm owns that grid — so a table that
does not fit shows up as the `<table>` frame overflowing instead.

## CLI traps

- **`preview --gallery` must differ from `--out`.** When they are the same directory the
  gallery wipes its own build output and then fails to read it:
  `검수 산출물을 읽을 수 없다: open …/*.pptx: no such file or directory`, exit 1. The message
  does not name the cause.
- **`init` scaffold IDs are `cover`, `what`, `end`** — not the filename stems. Delete the three
  scaffold slides before writing your own, or the first `check` starts with 2 errors.
- **`preview` is never a gate**: slides over the tolerance still exit 0. Exit 1 means nothing
  was produced to review.
- **`watch` needs `html` in `--target`** and refuses to start without it.
- **`init --theme` takes built-ins only** (`minimal` · `editorial` · `ink`); other commands
  also accept a local `themes/<name>/`, and never a path.

## What is known to work — don't go looking for trouble here

Eleven slides pass `--check-layout --strict` in 0.63s. `dist/index.html` has zero external
references. Structural validation is tight: `<p>` inside `<ul>`, `<li>` outside a list, a
mistyped slot name, and two elements in one slot are all caught. `watch` rebuilds in 3–5ms and
never blanks the tab on a broken save. `reveal` steps survive into the markup, and the overview
and print views show everything expanded. The `end` layout drops the footer and page number,
as its theme declares.

The open items above are known open issues in balpyo itself; the
viewer-side differences in full are in `${CLAUDE_PLUGIN_ROOT}/docs/guides/known-differences.md`,
and the warning triage guide is `docs/guides/layout-warnings.md`.
