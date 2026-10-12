# Elements, one by one

The rules each element follows, the mistakes the diagnostics catch, and snippets to copy.
Attribute lists and value sets are in `vocabulary.md`; values that come from the theme
(sizes, colours) are in `appearance.md`. Numbers below are the `minimal` theme's unless said
otherwise.

## `<slide>`

```html
<slide id="architecture" layout="content" justify="between" enter="rise" transition="fade">
```

- **`id`** — unique in the deck; the web deep link (`index.html#/architecture`) and the PPTX
  slide name. Omitted, it is the file name without its numeric prefix and extension
  (`010_architecture.slide` → `architecture`), so reordering files keeps links working. `id=""`
  is an error (remove it to derive the id). A collision is reported on the later slide and
  names the earlier one.
- **`layout`** — one of the 11; omitted means `content`.
- **`skip`** — the slide is parsed and checked but leaves the deck; later slides renumber with
  no gap.
- **`justify`** — vertical distribution of a **flow-mode** slide's children (`between` puts the
  first at the top of the content box and the last at the bottom). An error on a slot-mode
  slide. `gap` and `align` are not `<slide>` attributes.
- **`enter`**, **`transition`** — see Motion. **`invert`** — see Inverted slides.
- The flow stage is a vertical flex column with a 64px margin and a 24px gap between
  children, so direct children take `grow`, `self`, `order` and so on.
- No bare text and no inline tags directly in `<slide>` — wrap them in `<text>` or a block.

## `layout="canvas"`

Free placement. The stage is one overlapping cell: direct children are placed by `x`/`y`
(numbers, `%` of the stage axis, or negative) measured from the inside of the 64px margin —
`x="0" y="0"` is the content box's top-left, `x="-64"` reaches the slide edge. Document order is
z-order. `at` anchors a direct child to one of the 9 points of the content box, like a stack
child. Canvas content is never warned about for overlap or for using the margin, but it is still
warned about **leaving the stage**.

```html
<slide id="diagram" layout="canvas">
  <shape type="ellipse" fill="accent1" w="96" h="96" x="50%" y="40%"/>
  <text w="140" align="center" x="106" y="216">placed by hand</text>
  <text at="bottom-right" role="caption">anchored to the content box</text>
</slide>
```

## `<note>`

Speaker notes: one per slide, a direct child of `<slide>`, text only, no attributes. Line
breaks collapse to spaces like any text. Hidden on the web slide (shown in the presenter's
notes panel, `n`) and written as a real PowerPoint notes page. An empty `<note></note>` is no
note.

## Text: `<text>` and the block tags

- **Role** sets the whole baseline (font, size, weight, line height, colour). Tag defaults:
  `h1` title, `h2` heading, `h3` heading at 28px, `p` body, `blockquote` body in `muted`, `pre`
  code, `li` body. Only `<text>` takes `role` (default `body`).
- **`size`** — a unitless number > 0 (no `px`, no `%`). Every numeric attribute takes a plain
  decimal only — `12`, `-3`, `0.5`, `.5`; `1e3`, `NaN`, `Inf`, `+5`, `1_000` are
  `size value is not a number ("1e3")`.
- **`color`** — a token, a colour function, `#RRGGBB`, or `#RGB`.
- **`align`** — `start` (default) `center` `end` `justify`; `left`/`right` are errors (the
  meaning holds in right-to-left text too).
- **`anchor`** — `top` `center` `bottom`; it only does something inside a `<shape>`, where the
  shape's **first** child decides for the whole shape.
- **Width only**: text takes `w`/`min-w`/`max-w` but no height, `pad` or `ratio` — wrap it in
  a `<shape>` or a container to fix a height. Without `w`, a direct child of a flow slide is the
  full content width (so `align` moves the text inside it); in slots, containers, stacks and
  canvas the width is `auto` and the layout owns it.
- **Whitespace** collapses: runs of spaces and newlines become one space and the ends are
  trimmed. Spaces at the inside edges of an inline tag are trimmed too — write
  `<strong>deploy</strong> pipeline`, not `<strong>deploy </strong>pipeline`. `&#32;` survives
  collapsing.
- With `lang: "ko"` (or any `ko-…` tag), the web breaks lines at word boundaries, not inside
  words; a word longer than its box still breaks.
- An empty `<text></text>` still produces a text box in the PPTX.

## Inline markup

| Tag | Does |
|---|---|
| `strong` | bold (weight at least 700) |
| `em` | italic |
| `code` | the `mono` font at the surrounding size |
| `a href="…"` | underlined, `accent1`, a real hyperlink in both targets (`href` required). `#<slide id>`, `#/<slide id>` or `#/<number>` links inside the deck — a PPTX slide jump; a target not in the deck (skipped slides aren't) is a warning |
| `span color weight size` | recolour, `weight="bold"`/`"normal"` (no numbers), or a **smaller or equal** size (`<span size="28">` inside 56px text — a bigger span is an error) |
| `mark` | highlighter in the theme's highlight colour (no attributes; a theme without one makes `<mark>` an error) |
| `br` | a line break within the paragraph — `<br/>` |

```html
<h1 size="44">EC2 gives you <mark>resizable compute</mark> in minutes.</h1>
<text role="title" size="56">750<span size="28">+</span></text>
<p>Read the <a href="https://example.com/guide">guide</a> or run <code>balpyo check</code>.</p>
```

## Lists and `reveal`

```html
<ul size="20" color="muted">
  <li>Pricing experiments</li>
  <li reveal>Onboarding redesign</li>
  <li reveal="2">Self-serve billing</li>
</ul>
```

- `ul`/`ol` hold `<li>` only, and lists do not nest (a list inside `<li>` is an error). List
  attributes style every item; `<li>` takes only `reveal`. Markers are `•` and `1.`, with a 36px
  indent. In PPTX a list is one text box, one paragraph per item.
- **`reveal`** — valueless means "the next step on this slide"; a number puts items in the same
  step. Numbering runs across every list on the slide. Hidden items keep their space, so
  revealing never moves anything. Only the web presenter view steps; the overview, print and
  the PPTX show every item (use `enter` for PowerPoint motion).
- Presenter: next (→, Space) uses up the steps before changing slide; back (←) rewinds one step.
  A slide entered backwards is fully revealed; a deep link starts at step 0. Home/End restart
  the steps even on the slide already shown; arrows in the overview move tiles, not steps.

## Code: `<pre>`

```html
<pre slot="code" lang="go">func main() {
	fmt.Println("hello")
}</pre>
```

- Character data only — no tags inside (no `<pre><code>`). Use CDATA to show markup:
  `<pre lang="soml"><![CDATA[<slide id="x">…</slide>]]></pre>`.
- Whitespace is kept. One newline right after `<pre>` and one right before `</pre>` are dropped.
  Tabs expand to the theme's tab stops (2 columns in the built-ins), so pasted tab-indented code
  is fine.
- `lang`: `go` `xml` `soml` `json` `bash`, or omit it. Any other value is an error. Token
  colours come from the theme and stay theme-colour references in the PPTX. This is the code's
  language and unrelated to the deck's `lang` in `balpyo.json` (`<pre lang="ko">` is an error).
  For an unlisted language (`yaml`, `python` …) omit `lang` rather than borrowing a neighbour's.
- A code block does not wrap, on either target: long lines run past the box (a layout warning:
  `… text that cannot wrap (a long word, URL or pre)`). Break lines by hand or lower `size`.

## Containers: `<row>` `<col>` `<grid>` `<spacer>`

They are CSS flex/grid under new names: `row` lays out left→right, `col` top→bottom, `grid`
in tracks. Containers produce **no shape** in the PPTX — their children become shapes at slide
coordinates.

- `gap` — between children; `pad` — inside the container; `align` — cross axis; `justify` —
  main axis; `wrap` — let a row wrap (no effect on `grid`).
- Vertical position has two names: `justify`/`align` on containers, `anchor` on text and
  shapes. `<col anchor>` and `<shape justify>` are errors.
- `cols`/`rows` exist on `grid` only. A grid child spans tracks with `span="2"`.
- `<spacer/>` is `flex: 1` — it eats leftover space. `grow="2"` takes twice the share; a fixed
  gap is `<spacer grow="0" basis="120"/>`.
- Space tokens stay theme references on the web; numbers are frozen. In the PPTX all spacing is
  baked into coordinates.

```html
<row gap="lg" align="center">
  <col w="40%" gap="sm">
    <text role="heading">Now</text>
    <p>Two lines of context.</p>
  </col>
  <spacer/>
  <grid cols="2fr 1fr" gap="md" grow="1">
    <col span="2"><p>spans both tracks</p></col>
    <p>left</p><p>right</p>
  </grid>
</row>
```

## `<stack>` — overlap on purpose

Children share one cell; document order is z-order (first written is at the bottom). A stack
sizes to its **largest child**, so give it `w`/`h` or `ratio` when anchors need room.

- `at` places a child at one of 9 anchors inside the stack (minus its `pad`); `center-center`
  is not a value, write `center`.
- `x`/`y` offset a child from the stack's top-left (negative allowed, `%` of the stack axis);
  you may set only one axis.
- Children with neither follow the stack's `align` (vertical) and `justify` (horizontal). A
  stack has one cell, so `justify="between"` is `start` and `around`/`evenly` are `center`.
- In the PPTX a stack is nothing — no group; each child is its own shape.
- Overlap inside a stack is never warned about.

```html
<stack w="360" h="240">
  <image src="assets/screen.png" fit="cover" w="fill" h="fill" alt="The new dashboard"/>
  <shape type="rounded-rect" fill="accent1" pad="sm" at="top-right">
    <text size="14" color="bg">NEW</text>
  </shape>
</stack>
```

A watermark: write it **first** in a stack, under the content —
`<shape type="rect" fill="none" border="none" opacity="0.12" rotate="-20" at="center"><text size="96">DRAFT</text></shape>`.

## `<shape>`

`type` is required. With no variant the theme still styles a shape (minimal: `surface` fill,
`1 solid muted` border, `md` radius); `fill="none"` / `border="none"` remove them.

**Two modes**, decided by the shape's direct children:

- **Text only** (`<text>`, blocks, `<pre>`): one PowerPoint shape holding the paragraphs.
  Vertical position is the first child's `anchor`.
- **A container inside** (`row`/`col`/`grid`/`stack`): a **backdrop** — one shape for the
  outline plus sibling shapes for each child, no group. This is how to build a card with a
  pinned footer:

```html
<shape variant="card" type="rounded-rect" w="fill" h="260" pad="lg">
  <col h="fill" justify="between">
    <col gap="xs">
      <h3>Throughput</h3>
      <p color="muted">Requests per second at p99.</p>
    </col>
    <text role="caption">Updated hourly</text>
  </col>
</shape>
```

Lists, images, tables and charts reach a shape only through a container. A shape with no
content and no size measures 0×0 and produces nothing. A shape variant does **not** style the
text inside it (`card-header` does not bold its text) — put typography on the child.

Attribute rules:

- `radius` only on `type="rounded-rect"` (a token or a number).
- `opacity` (0–1) fades fill and border; text stays crisp.
- `rotate` turns the text too, and is an error on a shape that holds a container.
- `flip` (`h`/`v`/`hv`) does not flip the text.
- `border` is drawn on the outline, half outside the box.
- `pad` takes no `%` on shapes.
- `alt` names the shape for screen readers (it does not replace the text); `alt=""` is an
  error — omit it.
- `id` (unique per slide) is only for connectors.

**Presets need a height when they hold content** — `ellipse`, `diamond`, `cylinder`, `chevron`
(unless `notch="50%"`), `arrow-right`, `hexagon`, `parallelogram`, `star`, `cloud`, `triangle`,
`cube`: give `h` or `ratio`. `rect`, `rounded-rect`, `callout` size to content. Text flows inside
each preset's own text rectangle (an ellipse loses ~15% on each side, a diamond 25%, a star much
more), then `pad` applies inside that — so a labelled diamond needs room.

- **`chevron`** — `notch` (`20%` default, up to `50%`) is the depth of the point; the text sits
  between the two notches, so a short label on a narrow chevron wants a lower notch.
- **`cube`** — needs a definite size (`w`+`h` or `ratio`); depth is 25% of the short side, the
  text sits on the front face, the right face is 20% darker and the top 20% lighter.
- **`left-brace` / `right-brace`** — line shapes: use `fill="none"`; `rotate="90"` makes an
  under-brace.
- **`path`** — a custom outline. `d` is drawn in a **0–100 box** stretched to the shape's box
  (axes stretch independently). Absolute commands only: `M L H V C Q Z` (lowercase, `A`, `S`, `T`
  are errors), numbers separated by spaces or commas, coordinates may leave 0–100. An open path
  is still filled, so lines need `fill="none"`. It is editable with PowerPoint's Edit Points.
  Several subpaths (each new `M`) fill **even-odd** on both targets, whatever their direction,
  so an inner outline cuts a hole: `d="M0,0 L100,0 L100,100 L0,100 Z M25,25 L75,25 L75,75 L25,75 Z"`
  is a frame.

```html
<stack w="560" h="280">
  <shape type="path" fill="pattern(wide-diagonal-up muted/35 bg/0)" border="none" w="fill" h="fill"
         d="M0,85 H40 V55 H72 V30 H100 V100 H0 Z"/>
  <shape type="path" fill="none" border="3 solid fg" w="fill" h="fill"
         d="M0,98 C15,85 25,90 40,78 C50,70 52,58 62,56 C70,55 75,40 85,36 C93,33 97,30 100,22"/>
</stack>
```

## `<image>`

```html
<image src="assets/architecture.png" w="640" ratio="16:9" fit="cover" variant="framed" alt="Request path"/>
```

- **`src`** — a path from the **project root** (not from the slide file), one file, `.png`
  `.jpg` `.jpeg` `.svg`. Absolute paths, `../` outside the project, URLs, `data:`, backslashes,
  directories, and a missing extension are errors. `.gif`, `.webp`, `.avif`, `.heic`, `.bmp`,
  `.tif` are rejected — export PNG or JPEG. The extension must match the content. A symbolic
  link pointing outside the project is an error too (`src goes through a symbolic link that
  points outside the project (…)`) — the commit would hold only the link.
- **`theme:<path>`** — a picture from the active theme's `assets/`, looked up child-first
  through its `extends` chain (`theme:icons/lambda.png`). Works for installed themes with no
  copy in the project; a missing file lists the chain it searched.
- **Size** — intrinsic pixels are logical px (a 2× screenshot draws twice as large); scale with
  `w`/`h`/`ratio`. Bytes are never resized or recompressed — shrink big photos first. A JPEG's
  EXIF orientation is honoured on both targets (the PPTX keeps the stored pixels and adds a
  rotation/flip), so the intrinsic size and `rotate`/`flip` are those of the upright picture;
  PowerPoint's rotation box shows the EXIF turn added in.
- **`fit`** — `contain` (default) shrinks the box to the picture (no letterbox); `cover` crops
  the overflow equally on both sides. Without a size, `fit` does nothing. In the `media` slot of
  `image-full` / `image-text`, an image without a size fills the slot, so
  `<image slot="media" fit="cover" src="…"/>` is a full-bleed picture.
- **Look** — nothing by default; `variant="framed"` adds a frame. `radius` works on any image;
  `opacity` fades picture, border and shadow together. Images have no `fill` — put a `<shape>`
  behind it in a `<stack>`.
- **`alt`** — `alt=""` (or omitted) means decorative; a valueless `alt` is an error.
- An image is void, so a caption is a sibling `<text>`. It cannot sit directly in a `<shape>`
  (use `<shape><col><image/></col></shape>`).
- PPTX: one media part per distinct file, so "Change Picture" updates every use.

**SVG** (icons, logos) must be self-contained and carry an intrinsic size: the root `<svg>`
needs `width`/`height` in px or unitless (one of them plus a `viewBox` is enough; a
`viewBox`-only SVG is an error and the diagnostic suggests numbers). No `<script>`,
`<foreignObject>`, `on…` attributes, or references outside the file (`href`, CSS `url(…)`,
`@import`); in-document `#id` references and PNG/JPEG `data:` URIs are fine. `contain` never
enlarges, so to draw an icon bigger give `w`/`h` with `fit="cover"`. The PPTX carries the vector
plus a PNG fallback at 2× intrinsic size.

## `<table>`

```html
<table variant="striped" alt="Quarterly revenue">
  <tr><th rowspan="2">Region</th><th colspan="2">H1</th><th colspan="2">H2</th></tr>
  <tr><th>Q1</th><th>Q2</th><th>Q3</th><th>Q4</th></tr>
  <tr><td>EMEA</td><td>12</td><td>14</td><td align="center" fill="accent1" color="bg">19</td><td>21</td></tr>
</table>
```

- A real, editable PowerPoint table. Cells hold text and inline tags only — no blocks, lists,
  or shapes.
- `<table>` takes no size or look attributes — it sizes to its content and fills a slot it is
  placed in; to narrow it, size its container (`<col w="480"><table>…`). `<tr>` takes nothing.
- The **first row's colspan sum sets the column count**, and every row must fill it exactly;
  overlapping or out-of-bounds `rowspan`/`colspan` are errors that say where.
- A row of only `<th>` is the header (theme header style, excluded from striping).
  `variant="striped"` fills every other body row.
- Cell attributes override the theme: `fill` (token, `#rrggbb`, `none`), `color`, `border` (all
  four sides), `align`, `anchor` (`top` default). `border="none"` removes the theme's border.
- Column widths follow content. Cells are exempt from overflow judgement; a table that does not
  fit shows up as the `<table>` frame overflowing.
- An empty cell (or one holding only spaces or empty inlines) still keeps one line at the cell
  font size, so an empty row is as tall as a one-line row.

## `<chart>`

A chart is expanded into ordinary editable shapes and text — not a native PowerPoint chart (no
data sheet).

```html
<chart type="bar" unit="%" max="100" series="muted" h="320">
  <point label="Web" value="62"/>
  <point label="iOS" value="28" fill="accent3"/>
</chart>
```

```html
<chart type="line" alt="Demand vs capacity" w="fill" h="320">
  <series name="Demand" type="area" fill="accent4">
    <point label="6am" value="12"/><point label="noon" value="55"/><point label="6pm" value="70"/>
  </series>
  <series name="Capacity" type="step" fill="accent1">
    <point label="6am" value="30"/><point label="noon" value="60"/><point label="6pm" value="80"/>
  </series>
</chart>
```

- `h` or `ratio` is required (`h="fill"` counts). `type` is required. `line` and `area` need at
  least two points (`a line chart needs at least 2 points — …`); for one value use `bar` or
  `step`.
- `<point>`: `label` and `value` (a number ≥ 0, no unit — the unit goes in `<chart unit>`).
  `value="0"` draws no bar but keeps the label. `max` defaults to the data maximum ÷ 0.9; a
  value above an explicit `max` is an error.
- Colour: `series=` on the frame colours every bar (default `accent1`); `<point fill>` colours
  one bar *and* its labels. Idiom: `series="muted"` with one highlighted point. `none` is not a
  colour here.
- **Several series**: each `<series name>` lists the same categories in the same order; bars
  group, `line`/`area`/`step` overlay, and a series may pick its own `type` on a line-family
  chart. Series colours default to `accent1`…`accent6`. A legend appears under the categories;
  it wraps onto more lines when the names don't fit (swatches keep their size) and the plot
  gets shorter by that much.
  With `<series>`, the frame's `series=` is an error; never mix bare `<point>`s and `<series>`.
- Line-family charts draw a 3px line through category centres (`area` fills under it at 20%);
  they have no value labels and no `<point fill>`.
- The chart frame has no `fill`, `border`, `radius`, `shadow` or `variant` — wrap it in a
  `<shape>` (through a container) for a card. Labels use the caption role.
- Every category column is the same width: a label wider than its column overflows in place (a
  text-overflow warning on that `<point>`) instead of pushing its neighbours off their bars.
- A chart too small for its labels and legend gets one warning on the `<chart>` naming the part
  that overflows, and says so when no height is left for the data:
  `<chart> is too small for its content — the plot and legend overflow the chart at the bottom by 41px and nothing is left for the bars (give the chart more room or shorten its labels)`.

## `<connector>`

A line between two measured boxes: a direct child of `<slide>`, taking no space in the flow and
drawn after layout, so it may name a box declared later. In PPTX it is a real connector glued
to the boxes.

```html
<slide id="lifecycle">
  <h2>Instance lifecycle</h2>
  <row gap="xl" align="center">
    <shape type="rounded-rect" id="pending" w="200" h="72"><p align="center">pending</p></shape>
    <shape type="rounded-rect" id="running" w="200" h="72" fill="accent3"><p align="center" color="bg">running</p></shape>
    <shape type="rounded-rect" id="stopped" w="200" h="72"><p align="center">stopped</p></shape>
  </row>
  <connector from="pending" to="running" enter="draw"/>
  <connector from="running" to="stopped" variant="accent" label="stop"/>
  <connector from="stopped" to="pending" route="curved" from-side="top" to-side="top" variant="dashed"/>
</slide>
```

- `from` and `to` are required and must be `id`s of a `<shape>` or `<image>` on the **same
  slide**, and different from each other.
- `auto` sides face the other box (left/right when the boxes are further apart horizontally,
  else top/bottom); ends attach at side midpoints. `elbow`/`curved` between opposite sides is a
  mid-step, between perpendicular sides an L, and between the same side (top → top) a U outside
  both boxes. A line that would cut through a box goes around instead: sides facing away from
  each other (right → left with the target on the left) get an S that leaves each side by 32px,
  and a U blocked by the other box detours 32px past both — leave that room around the boxes.
- `line` is the border grammar (`"2 dashed accent1"`) — one colour, no gradient, no `none`.
  Theme default: `2 solid muted` with an end arrow; `start`/`end` markers are `none` `arrow`
  `dot`.
- `label` is one short line beside the path midpoint, in caption style.
- `flow` animates the dashes on the web presenter view only (always dashed); PPTX shows a still
  dash.
- Connectors paint above all content, below the footer and page number.
- PPTX glues the ends only to `rect`, `rounded-rect`, `diamond`, `ellipse`, and images; other
  presets get a line in the right place that does not follow a drag in PowerPoint.
- Connectors and labels are not layout-judged — see `pitfalls.md`.

## Motion: `enter` and `transition`

```html
<slide id="plan" enter="rise" transition="fade">
  <h2>The plan</h2>
  <grid cols="3" gap="lg" enter="zoom">…</grid>   <!-- the grid arrives as one -->
  <p enter="none">Always visible.</p>
</slide>
```

- `enter` on `<slide>` applies the effect to every direct child (connectors excepted), staggered
  in document order. On any content element it applies to that element and everything inside
  it; a nested `enter` arrives on its own; `enter="none"` opts out. An element belongs to its
  nearest ancestor with `enter`. Inside a `<shape>`, a paragraph with its own `enter` arrives
  separately from the shape (PPTX builds that shape by paragraph).
- Effects: `fade`; `rise` (fade + up from 24px below; PowerPoint Float In); `zoom` (fade + from
  90%); `wipe` (left to right). Connectors take `draw` (a wipe in PPTX) or `fade`.
- Timing comes from the theme (built-ins: each 600ms, 90ms apart, after 150ms). A theme without
  `motion` makes any `enter` an error.
- `transition` on `<slide>`: `fade` cross-fades, `push` slides in from the right (from the left
  going back); default is the theme's (built-ins: none).
- Motion never moves anything — layout and PPTX coordinates are the final state. The web runs
  entrances only when entering a slide forward in the presenter view; going back, the overview,
  print, and `prefers-reduced-motion` show the final state.

## Inverted slides

`<slide invert>` swaps `bg` ↔ `fg` and `surface` ↔ `muted` for that slide; the six accents stay.
The markup is unchanged — `color="fg"` simply becomes light on a dark slide — and the footer and
page number swap too. In PPTX it is a colour-map override, so theme colours still drive it.
Accents now sit on the theme's `fg`, so keep small accent text off inverted slides.

## `style` — the escape hatch

`style="…"` adds CSS declarations to one element, applied last. Layout declarations are
measured, so they reach the PPTX:

`display` `flex` `flex-*` `grid` `grid-*` `gap` `row-gap` `column-gap` `padding*` `margin*`
`width` `min-width` `max-width` `height` `min-height` `max-height` `box-sizing` `align-*`
`justify-*` `place-*` `order` `aspect-ratio` `position` `inset*` `top` `right` `bottom` `left`
`translate`

Anything else (`background`, `color`, `border`, `rotate`, `z-index`, …) is web-only and a
warning: `the "background" declaration in style is not layout — it applies only to the web and does not carry over to PPTX; …`.
A malformed declaration (no colon, no value) is an error and drops the whole `style`. Values
are not checked. Use it for what the vocabulary lacks (`grid-template-columns: minmax(0, 1fr) 2fr`),
not for appearance.
