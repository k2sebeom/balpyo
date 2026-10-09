# Pitfalls and known differences

Rough edges measured by writing real decks, with the workaround for each. These are current
facts about the tool, not style advice — the ones under "the build will not catch this" are
the reason a deck still deserves one human look.

## The build will not catch this

### Line breaks are each engine's business

The two targets agree on **components and layout** — the same object, at the measured rect,
with the same resolved appearance, in the same order. They do not promise the same line
breaks. The web breaks a Korean deck (`lang: ko`) at word boundaries (`keep-all`); a PPTX text
box wraps inside its width by the viewer's own rules, so a paragraph that is two lines on the
web can be three in PowerPoint.

It does not spill: PPTX text bodies carry `normAutofit`, so when the viewer needs an extra
line it shrinks the text to fit the box, and neighbours never move. The cost is a smaller
font on that one box. So:

- Leave real room in sized boxes (`<shape h="…">`) instead of fitting text to the last pixel.
- Keep paragraphs in narrow shapes short; a card is a label, not a paragraph.
- Don't chase identical line breaks — that is outside the contract. Open the PPTX or run
  `balpyo preview` to see whether the shrink is acceptable.

### A centred slot has no slack

Where a theme centres a slot (`justify-items: center`), grid shrinks the box to the **content
width**, so the box is exactly as wide as the line. A box filled by one measured line is
written `wrap="none"`: if the viewer sets the line a few px wider (PowerPoint does no kerning
by default) it pokes slightly past the box instead of dropping its last word onto a second
line. A longer text in such a slot still gets its lines re-broken and shrunk. The centred
slots in the built-in themes are: `minimal` → `title`, `stat`, `end`; `editorial` → `stat`;
`ink` → `stat`, `section-divider`.

**Keep those lines short** — one number, one headline phrase. If the text has to be a sentence,
pick a layout that does not centre.

### Script spacing around punctuation (LibreOffice only)

Opening the PPTX in LibreOffice shows a gap at every CJK↔Latin boundary — `선언으로 , 색과` —
about ¼ em each. It is the viewer's CJK typesetting rule (JIS X 4051), it is not in the file,
and DrawingML has no attribute to turn it off. **Nothing to do**: boxes and coordinates match
the web, the text holds no extra spaces, and PowerPoint and Keynote may not do it at all.

### A warning-free build is not a finished deck

Layout judgement asks whether a box escaped a box. Overflow that ends inside a parent's `pad`
is silent (up to 24px in a `pad="lg"` shape) — that is the price of having no false positives,
which is what makes `--strict` usable. Zero warnings means nothing measurable is wrong, not
that nothing is wrong. Look at the web output and the PPTX once before calling it done.

## Diagnostics: what they will and will not tell you

- **Near misses get one candidate.** A name one or two letters off (one letter for names of
  four letters or fewer) is answered with `— 혹시 "heading"?`; HTML habits are mapped too
  (`<img>` → `<image>`, `<b>`, `<i>`, `<h4>`–`<h6>`). Anything further off gets only the list.
- **An unknown tag lists what fits that spot** — the tags the parent allows, not the whole
  vocabulary: `알 수 없는 태그 <txt> — 혹시 <text>? (<col> 안에 쓸 수 있는 태그: …)`. Value and
  attribute diagnostics list their legal set the same way.
- **Diagnostics are sorted by file, line, and column**, so the first one printed is the
  earliest in the deck — usually the one to fix first.
- **A slide-ID collision is reported on the later slide** and names the earlier one by
  `file:line:col`: `슬라이드 ID "cover"가 slides/001_title.slide:1:12와 충돌한다`. A slide
  without `id=` takes its ID from the filename, so the position is its `<slide>` tag. Give one
  of them a different `id`, or delete the one you don't need.
- **Duplicate content in one slot** surfaces as a geometry overlap warning
  (`<text>가 형제 <text>와 95px×36px 겹친다`), not a vocabulary error. Two elements with the
  same `slot=` is the usual cause.
- **A missing theme font is a warning** on every measuring build, and `--strict` makes it an
  error. It is an environment problem, not a deck problem: install the font, don't edit slides.

## One accident, one warning

A box that escapes is reported once; the descendants that escaped with it are counted at the
end of that warning instead of reported on their own:
`<ul>가 스테이지를 아래로 120px 벗어난다 … (안의 <li> 6개도 함께 벗어난다)`. Fix the box the
warning names and re-run — geometry is measured, so one fix moves every box below it.

Amounts are logical px. The content box is 1152×592 (the 1280×720 stage minus a 64px margin)
and the warning states the box it measured on that slide, so "34px" is about one body line.

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

- **`preview --gallery` may not overlap `--out`** (same directory, or one inside the other —
  including an `outDir` of `preview`). It stops before building:
  `검수 갤러리 디렉토리 …가 빌드 출력 디렉토리 …와 겹친다`. Pick two separate directories.
- **`init` scaffold IDs are `cover`, `what`, `end`** — not the filename stems. Delete the three
  scaffold slides before writing your own, or the first `check` starts with 2 errors.
- **`preview` is never a gate**: there is no diff or tolerance, and exit 0 only means the
  gallery was made. Exit 1 means nothing was produced to review (build failed, an explicit
  `--target pptx` with no rasterizer, output that does not match the deck).
- **`watch` needs `html` in `--target`** and refuses to start without it.
- **`init --theme` takes built-ins only** (`minimal` · `editorial` · `ink`); other commands
  also accept a local `themes/<name>/`, and never a path.

## What is known to work — don't go looking for trouble here

Eleven slides pass `--check-layout --strict` in well under a second. `dist/index.html` has zero
external references. Structural validation is tight: `<p>` inside `<ul>`, `<li>` outside a list,
a mistyped slot name, and two elements in one slot are all caught. `watch` rebuilds in
milliseconds and never blanks the tab on a broken save. `reveal` steps survive into the markup,
and the overview and print views show everything expanded. The built-in themes give the
`quote`, `stat`, and `code` layouts finished typography (a 40px quote, a value larger than the
title, a label at least body size, a caption right under its code), and the `end` layout drops
the footer and page number, as its theme declares.

What lies outside the two targets' contract, in full, is in
`${CLAUDE_PLUGIN_ROOT}/docs/guides/known-differences.md`; the warning triage guide is
`docs/guides/layout-warnings.md`.
