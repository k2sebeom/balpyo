# Pitfalls, layout warnings, and known differences

Current facts about the tool, not style advice. The ones under "the build will not catch this"
are the reason a deck still deserves one look before it is called done.

## What the two targets promise

The web and the PPTX agree on **components and layout**: the same kind of object, at the
measured position and size, in the same paint order, with the same resolved appearance (font,
size, weight, colour token, alignment), links, list markers, and text runs. They do **not**
promise the same line breaks, glyph widths, kerning, script spacing, anti-aliasing, or the exact
vertical position of a line. Don't chase those — they are each viewer's business.

## The build will not catch this

### Line breaks are each engine's business

The web breaks a Korean deck (`lang: ko`) at word boundaries; a PPTX text box wraps inside its
width by the viewer's own rules, so a paragraph that is two lines on the web can be three in
PowerPoint.

It does not spill: PPTX text bodies shrink to fit (`normAutofit`), so when the viewer needs an
extra line it shrinks the text in that one box, and neighbours never move. Code blocks are the
exception — they never wrap and never shrink, on either target. So:

- Leave real room in sized boxes (`<shape h="…">`) instead of fitting text to the last pixel.
- Keep paragraphs in narrow shapes short; a card is a label, not a paragraph.
- Don't chase identical line breaks. Open the PPTX or run `balpyo preview` to see whether the
  shrink is acceptable. `balpyo snap` will not show it: it draws the web layout, not the PPTX.

### A centred slot has no slack

Where a theme centres a slot, the box shrinks to the **content width**, so the box is exactly as
wide as the line. A box filled by one measured line is written not to wrap: if the viewer sets
the line a few px wider (PowerPoint does no kerning by default) it pokes slightly past the box
instead of dropping its last word onto a second line. A longer text in such a slot still gets
its lines re-broken and shrunk. The centred slots in the built-in themes are: `minimal` →
`title`, `stat`, `end`; `editorial` → `stat`; `ink` → `stat`, `section-divider`.

**Keep those lines short** — one number, one headline phrase. If the text has to be a sentence,
pick a layout that does not centre.

### Script spacing around punctuation (LibreOffice only)

Opening the PPTX in LibreOffice shows a gap at every CJK↔Latin boundary — `선언으로 , 색과` —
about ¼ em each, including before Latin punctuation and around ASCII words. It is the viewer's
CJK typesetting rule, it is not in the file, and nothing in the file format turns it off.
**Nothing to do**: boxes and coordinates match the web, the text holds no extra spaces, and
PowerPoint and Keynote may not do it at all.

### Connectors and their labels are not layout-judged

A `<connector>` takes no space in the flow, so overflow and overlap warnings never look at it,
and neither do its labels. A curved or elbow U-turn between two `top` sides arcs above both
boxes, so its label can leave the stage without a warning. Leave room above (for example
`<slide justify="center">`, or move the boxes down) and look at the slide. An elbow line that is
drawn in (`enter="draw"`) may be revealed in a different order on the two targets: PowerPoint
has no draw-along-path effect, so it wipes in from the start side.

### Viewer-side differences in fills and images

- **An off-centre radial gradient** (`radial(…, at 0% 100%)`) is written with its focus point,
  but OnlyOffice ignores the focus and spreads it from the centre, so a corner glow becomes a
  smudge in the middle. PowerPoint draws it right. A centred `radial(…)` matches everywhere.
- **Gradients with three or more stops** are approximated by LibreOffice.
- **Diagonal gradient angles** drift slightly between the targets on non-square boxes.
- **Pattern density differs**: the web tile scales with the stage; PowerPoint and OnlyOffice
  draw the pattern in screen pixels. Same pattern and colours, different line spacing.
- **Image borders**: the web draws them inside the box, PowerPoint centred on the edge, so a
  framed picture looks one border-width larger in the PPTX.
- **SVG images** carry a PNG fallback at 2× intrinsic size for viewers older than PowerPoint
  2016. Blown up far past that, the fallback blurs. The vector itself does not.
- **`flow`** connectors move on the web presenter view only; PPTX, print, and the overview show
  the still dash.
- **`reveal`** steps exist on the web presenter view only; the PPTX shows every item (use
  `enter` for PowerPoint motion).

### A warning-free build is not a finished deck

Layout judgement asks whether a box escaped a box. Overflow that ends inside a parent's `pad`
is silent (up to the pad's size) — that is the price of having no false positives, which is
what makes `--strict` usable. Zero warnings means nothing measurable is wrong, not that nothing
is wrong. Look at `balpyo snap` output, and open the PPTX once before calling it done.

## Layout warnings, and how to fix each

`--check-layout` (or any `pptx` build) measures every box in a headless browser. Warnings never
stop a build by themselves; `--strict` makes them errors. Amounts are logical px: the stage is
1280×720 and the content box 1152×592 (64px margin), so "34px" is about one body line (24px
type on a 36px line).

**One accident, one warning.** A box that escapes is reported once; descendants that escaped
with it are counted at the end: `… (5 <li> inside escape too)`. **Cause first**: a box that
overflows its parent and the stage is one warning about the parent, with the stage in the same
sentence (`<col> overflows its parent <shape> at the bottom by 40px and leaves the stage at the bottom by 11px — …`).
Fix the box the warning names and re-run — geometry is measured, so one fix moves every box
below it. Charts are reported by the tag you wrote: bars and labels are `<point>`, legend
entries `<series>`, the frame `<chart>`.

### `<p> overflows its parent <shape> at the bottom by 34px — the parent box is too small for its content`

A sized box cannot hold its content. In this order:

1. **Grow the box** — `h="140"` → `h="200"`. The height needed is content plus `pad` twice
   (`pad="lg"` is 36px top and bottom in `minimal`).
2. **Shrink the type** — `size`, or a smaller `variant`. Changing size twice on one slide is
   usually a signal that there is too much content.
3. **Cut the words.** A slide is not a document.

Table cells are exempt (the table algorithm owns them); a table that does not fit shows up as
the `<table>` frame overflowing.

### `<text> overflows the content box at the bottom by 34px — this layout's content box is (64.0,64.0 1152.0×592.0) …`

A slide-level element ate the 64px margin. Split the slide, turn it sideways with
`<grid>`/`<row>`, or reduce `size`. If using the margin is the design, `layout="canvas"` is the
sanctioned way. The content box differs by layout (`image-full` has no margin).

### `<shape> leaves the stage on the right by 120px — off-stage content is clipped on screen and placed off the slide in PPTX`

**No exception**, even with explicit `x`/`y` and even on `canvas`: the web clips it and the PPTX
puts the shape off-slide, so the two targets break differently. Build an off-screen effect with
a `<stack>` and sizes inside the stage instead.

### `<shape> overlaps its sibling <shape> by 164px×64px — if the overlap is intended, use <stack> or x/y`

In a flow container the declared sizes add up to more than the room (add up the `w`s). If the
overlap is the design (a badge, a tag, a watermark), say so with `<stack>`. Never warned:
elements placed with `at`/`x`/`y` and their descendants, anything on `layout="canvas"`, and
children of `<stack>`. Two elements in one slot also show up as an overlap warning — the usual
cause is a duplicated `slot=`.

### `<pre> text overflows its box on the right by 120px — text that cannot wrap (a long word, URL or pre)`

An unbreakable line: a long URL or identifier, or a `<pre>` line. Break the line by hand
(`<pre>` keeps your line breaks) or reduce `size`.

### Chrome warnings

`deck chrome (footer) leaves the slide …` — the footer text in `balpyo.json` is too long for the
band; shorten it. `theme "…" has no footer chrome declaration, so footer cannot be used …` —
the theme does not place a footer; drop it from `balpyo.json` or use a theme that has one.

## Diagnostics: what they will and will not tell you

- **Near misses get one candidate.** A name one or two letters off (one letter for names of
  four letters or fewer) is answered with `— did you mean "heading"?`; HTML habits are mapped
  too (`<img>` → `<image>`, `<b>`, `<i>`, `<h4>`–`<h6>`). Anything further off gets only the
  list.
- **An unknown tag lists what fits that spot** — the tags the parent allows, not the whole
  vocabulary: `unknown tag <txt> — did you mean <text>? (tags available inside <col>: <text>, <shape>, …)`.
  Value and attribute diagnostics list their legal set the same way.
- **Diagnostics are sorted by file, line, and column**, so the first one printed is the
  earliest in the deck — usually the one to fix first.
- **A slide-ID collision is reported on the later slide** and names the earlier one by
  `file:line:col`:
  `slide ID "cover" collides with slides/001_title.slide:2:12 (give one of them a different id, or remove the slide you do not need)`.
  A slide without `id=` takes its ID from the filename. Give one of them a different `id`, or
  delete the one you don't need.
- **A missing theme font is a warning** on every measuring build, and `--strict` makes it an
  error:
  `theme "minimal" font "Noto Sans CJK KR" (heading.ea, body.ea) is missing from the measuring browser — …`.
  It is an environment problem, not a deck problem: install the font, don't edit slides.
- **Theme gaps are errors on the element that needs them.** A custom theme without motion:
  `<p> has enter="rise", but theme "corp" defines no motion — add "motion": { … } to its tokens.json`.
  Without a highlight colour, `<mark>` fails; without a default connector line, a
  `<connector>` without `line=` fails. Fix the theme or drop the feature.

## Things that look legal and are not

- `at` directly on a canvas child (use `x`/`y`, or wrap in `<stack>`); `x`/`y` in a `<row>`.
- `role` on `<p>`/`<h2>` (only `<text>` takes `role`); `w`/`pad` on `<table>`; `fill` on
  `<image>` or `<chart>`; `radius` on anything but `rounded-rect` and images; `h` on `<text>`.
- `align="left"` (use `start`), `justify="space-between"` (use `between`), `weight="700"` (use
  `bold`), `size="24px"`, `opacity="40%"`, `rotate="45deg"`.
- A list inside `<li>`; a `<p>` inside a `<td>` or another `<p>`; `<pre><code>`; `<br>` without
  the slash; bare text in a container.
- `justify` on a slot-mode slide; `slot=` on a grandchild or on `canvas`.
- `&nbsp;` (use a regular space or `&#160;`).

## CLI traps

- **`preview --gallery` may not overlap `--out`** (same directory, or one inside the other —
  including an `outDir` of `preview`). It stops before building:
  `preview gallery directory … overlaps build output directory … — …`. Pick two separate
  directories. The gallery is cleared on every run.
- **Relative paths differ**: `build --out` is relative to the project root, but
  `preview --gallery` and `snap --out` are relative to the current directory.
- **`init` scaffold IDs are `cover`, `what`, `end`** — not the filename stems. Delete the three
  scaffold slides before writing your own, or the first `check` starts with collisions.
- **`snap` is never a gate either.** A clean `snap` run means the builtin renderer found no
  layout warning. The gate is still `check --check-layout --strict`: its browser measurement is
  the one the build uses. `snap` writes into `<outDir>/snap/` by default — inside the build
  output, so don't ship `dist/` with it, or pass `--out`.
- **`snap` does not warn about missing fonts.** It picks the nearest installed font silently. A
  slide that looks wrong only in `snap` may just be missing a theme font. `check --check-layout`
  reports missing fonts.
- **`preview` is never a gate**: there is no diff or tolerance, and exit 0 only means the
  gallery was made.
- **`watch` needs `html` in `--target`** and refuses to start without it. It does not notice
  `balpyo theme update`; save a project file or restart it.
- **`init --theme` takes built-ins only** (`minimal` · `editorial` · `ink`); other commands
  also accept a local `themes/<name>/` or an installed theme, and never a path. Builds don't
  download: an installed theme missing on CI is
  `balpyo.json: theme: unknown theme "acme" — not a built-in theme, a theme in the project's themes/, or an installed theme (available themes: …)`
  — add the same `balpyo theme install` step before the build.
- **`balpyo.json` rejects unknown keys**, and its `theme` must resolve even when `--theme`
  overrides it.

## What is known to work — don't go looking for trouble here

A dozen slides pass `--check-layout --strict` in about a second. `dist/index.html` has zero
external references. Structural validation is tight: `<p>` inside `<ul>`, `<li>` outside a list,
a mistyped slot name, and two elements in one slot are all caught. `watch` rebuilds in
milliseconds and never blanks the tab on a broken save. `reveal` steps survive into the markup,
and the overview and print views show everything expanded. The built-in themes give the
`quote`, `stat`, and `code` layouts finished typography (a 40px quote, a value larger than the
title, a label at least body size, a caption right under its code), and the `end` layout drops
the footer and page number, as its theme declares.
