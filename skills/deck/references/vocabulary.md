# SOML vocabulary

Everything legal in a `.slide` file. The vocabulary is **closed**: a name absent from these
tables is a build error, not a fallback. When something here disagrees with the binary, the
binary is right — `${CLAUDE_PLUGIN_ROOT}/docs/reference/` is the authority this sheet is
condensed from.

## Slide

```html
<slide id="kpis" layout="content">…<note>speaker only</note></slide>
```

| Attribute | Values | Notes |
|---|---|---|
| `layout` | the 11 names below | default `content` |
| `id` | free | must be unique in the deck; used by `index.html#/<id>` |
| `skip` | valueless or `"true"` | the slide is parsed but not rendered |
| `justify` | `start` `center` `end` `between` `around` `evenly` | vertical distribution of flow children |
| `enter` | `fade` `rise` `zoom` `wipe` `none` | the same entrance on every direct child (connectors excepted) — see "Motion" |
| `transition` | `fade` `push` `none` | how this slide comes in; default is the theme's `motion.transition` (built-ins: none) |
| `invert` | valueless or `"true"` | a dark slide in a light theme (or light in `ink`) — see "Inverted slides" |

`<note>` is the speaker note — never drawn, kept hidden in the web output and written as a real
PowerPoint notes slide.

A `<shape>` that holds a container becomes a **backdrop**: its children get their own
coordinates as sibling shapes in the PPTX. That is the supported way to build a card.

## Layouts and their slots

Put content in a slot with `slot="<name>"` on a **direct child** of `<slide>` — a slot on a
grandchild is an error. One `slot=` anywhere puts that slide in **slot mode**; a slide with
none stays in flow mode. The choice is per slide, so two slides can share a layout and use it
differently.

| `layout` | Slots (in order) | For |
|---|---|---|
| `title` | `kicker` `title` `subtitle` `author` `date` | the cover |
| `section-divider` | `title` `subtitle` | a chapter break |
| `content` | `title` `body` | the default workhorse |
| `two-column` | `title` `left` `right` | side-by-side comparison |
| `image-full` | `media` `caption` | a full-bleed image |
| `image-text` | `title` `media` `body` | image beside prose |
| `quote` | `quote` `attribution` | a pull quote |
| `code` | `title` `code` `caption` | a code listing |
| `stat` | `title` `value` `label` `body` | one big number |
| `canvas` | — | free placement with `at`/`x`/`y` |
| `end` | `title` `contact` | the closing slide |

Three consequences worth memorising:

- **A slot child that is a single frame** (`<text>`, a block, a list, `<shape>`, `<image>`)
  becomes a real PowerPoint placeholder, which is what keeps a slotted deck editable in
  PowerPoint's outline view. A **container** carrying the slot (`<col slot="left">`) is a web
  area only — no placeholder.
- **`justify` on a slot-mode slide is a build error.** Vertical rhythm there belongs to the
  theme's `layouts/<name>.slots.css`. Use `justify` on flow-mode slides only.
- **Horizontal placement in a slot belongs to the theme** (`justify-items: center`), and a
  slot child's width is `auto`. So don't put `align` on slot children — the built-in cover
  example has none. To centre text inside a specific width, give that width to the text:
  `<text w="140" align="center">`.

`slot=` on `layout="canvas"` is an error (it declares no slots). If the active theme is missing
a layout's `*.slots.css`, the whole slide falls back to flow mode in both targets — all three
built-in themes ship slot CSS for all ten slotted layouts.

## Tags

| Kind | Tags |
|---|---|
| Block | `h1` `h2` `h3` `p` `blockquote` `pre` |
| Text element | `text` (the explicit, role-carrying form of a block) |
| Lists | `ul` `ol` containing `li` |
| Inline | `strong` `em` `code` `a` `br` `span` `mark` |
| Containers | `row` `col` `grid` `stack` |
| Leaf | `spacer` `shape` `image` |
| Table | `table` > `tr` > `th` `td` |
| Chart | `chart` > `point`, or `chart` > `series` > `point` |
| Diagram | `connector` (direct child of `<slide>` only) |

`<h1>`/`<h2>`/`<h3>`/`<p>`/`<blockquote>`/`<pre>` are shorthands that carry a role implicitly;
`<text role="…">` says it outright. Use whichever reads better — they compile to the same node.

## Attributes by tag

Shared groups, referenced below as *box* and *child*:

- **box** — `pad` `w` `h` `min-w` `min-h` `max-w` `max-h` `ratio`
- **child** (placement inside the parent) — `grow` `shrink` `basis` `span` `self` `order`
  `at` `x` `y` `style`

Every tag that can be a direct child of `<slide>` also takes `slot`. Every content tag — text,
blocks, lists, containers, `shape`, `image`, `table`, `chart` — also takes `enter`.

| Tag | Own attributes | Plus |
|---|---|---|
| `text`, `h1`–`h3`, `p`, `blockquote`, `pre` | `role` `variant` `size` `color` `align` `anchor` `w` `min-w` `max-w` (`pre` adds `lang`) | child, slot |
| `row`, `col` | `gap` `align` `justify` `wrap` `pad` | box, child, slot |
| `grid` | `cols` `rows` + everything `row` takes | box, child, slot |
| `stack` | `align` `justify` `pad` (no `gap`/`wrap`/`cols`/`rows`) | box, child, slot |
| `shape` | `type` `id` `alt` `d` `variant` `fill` `border` `radius` `shadow` `opacity` `rotate` `flip` | box, child, slot |
| `image` | `src` `id` `fit` `alt` `variant` `border` `radius` `shadow` `opacity` `rotate` `flip` | box, child, slot |
| `table` | `alt` `variant` | box, child, slot |
| `th`, `td` | `colspan` `rowspan` `fill` `border` `color` `align` `anchor` | — |
| `chart` | `type` `unit` `max` `series` `alt` | box, child, slot |
| `series` | `name` (required) `fill` `type` | — |
| `point` | `label` `value` `fill` | — |
| `connector` | `from` `to` `route` `from-side` `to-side` `line` `start` `end` `variant` `label` `enter` `flow` | — (no box, no child, no slot) |
| `ul`, `ol` | — | child, slot |
| `li` | `reveal` | — |
| `a` | `href` | — |
| `span` | `color` `weight` `size` | — |
| `spacer` | none of its own — it is not a container | child, slot |
| `note`, `tr`, `strong`, `em`, `code`, `br`, `mark` | none | — |

`<spacer/>` is a void element (no children, no text) that behaves as `flex: 1`; size it with
the child attributes, usually `grow="2"`. A container may not hold bare text — wrap it in
`<text>` or a block tag, or the PPTX has no shape to put the words in. `<note>` is slide
metadata: one per slide, direct child of `<slide>` only, text only, no attributes.

## Value vocabularies

| Attribute | Legal values |
|---|---|
| `role` | `title` `heading` `body` `caption` `code` |
| `align` (text) | `start` `center` `end` `justify` |
| `align` (container, cross axis) | `start` `center` `end` `stretch` `baseline` |
| `anchor` | `top` `center` `bottom` |
| `justify` | `start` `center` `end` `between` `around` `evenly` |
| `at` | `top-left` `top-center` `top-right` `center-left` `center` `center-right` `bottom-left` `bottom-center` `bottom-right` |
| `fit` | `cover` `contain` |
| `wrap`, `skip`, `reveal` | valueless, or `"true"` |
| `shape type` | `rect` `rounded-rect` `ellipse` `diamond` `cylinder` `chevron` `arrow-right` `hexagon` `parallelogram` `callout` `star` `cloud` `triangle` `cube` `left-brace` `right-brace` — plus `path` (custom outline, needs `d`) |
| `chart type` | `bar` `line` `area` `step` |
| `series type` | `line` `area` `step` (line-family charts only) |
| `enter` | `fade` `rise` `zoom` `wipe` `none`; on `<connector>`: `draw` `fade` `none` |
| `transition` | `fade` `push` `none` |
| `invert`, `flow` | valueless, or `"true"` / `"false"` |
| connector `route` | `straight` (default) `elbow` `curved` |
| connector `from-side` `to-side` | `auto` (default) `top` `right` `bottom` `left` |
| connector `start` `end` | `none` `arrow` `dot` |
| `image src` | a **project-root-relative** path to one file; `.png` `.jpg` `.jpeg` `.svg`. Absolute paths, `../` outside the project, URLs, and `data:` are all errors — the deck must carry its own images. `assets/` is the convention, not a rule. **Or** `theme:<path>` — a picture from the active theme's `assets/` (searched through its `extends` chain, child first), e.g. `theme:icons/lambda.png`; works for installed themes with no `themes/` in the project, and a theme lacking the file is a build error |

**SVG images** (icons, logos) must be self-contained and carry an intrinsic size: the root `<svg>`
needs `width`/`height` in px or unitless (a `viewBox`-only SVG is an error — the diagnostic
suggests the numbers). No `<script>`, `<foreignObject>`, `on…` attributes, or references outside
the file (`href`, CSS `url(…)`, `@import`); in-document `#id` references and PNG/JPEG `data:` URIs
are fine. Without size attributes an image renders at its intrinsic size and `contain` never
enlarges, so to draw an icon bigger give `w`/`h` with `fit="cover"`. PPTX gets the vector plus a
browser-rendered PNG fallback (2× intrinsic size).

Sizes and spacing do **not** share one value vocabulary:

| Attribute | Values |
|---|---|
| `w` `h` `min-w` `min-h` `max-w` `max-h` | number · `N%` · `fill` |
| `gap` `pad` | space token (`md`) · number · `N%` (`pad` takes 1–4 values) |
| `grow` `shrink` `order` `span` | unitless number |
| `basis` | number · `N%` (**not** `fill`) |
| `x` `y` | number · `N%` · negative |
| `cols` `rows` | a track count (`cols="4"` = four equal tracks), or a list of two or more tracks from number · `Nfr` · `N%` · `auto` |

Numbers are **unitless logical px**. A `px`/`pt`/`em` suffix is an error, `%` is the only
suffix, and a negative length is an error everywhere except `x`/`y`. A space token in `w` is
an error; a number in `gap` is fine.

`fill` is **100% of that axis**, not `flex-grow: 1` — `w="fill"` fills the parent, `grow="1"`
splits the leftover space. They are different results and both are often right.

## Tokens — write these, not literals

| Kind | Names |
|---|---|
| `color` | `fg` `bg` `muted` `surface` `accent1` `accent2` `accent3` `accent4` `accent5` `accent6` |
| `font` | `heading` `body` `mono` |
| `space` (`gap`, `pad`) | `xs` `sm` `md` `lg` `xl` `2xl` |
| `radius` | `sm` `md` `lg` |
| `shadow` | `soft` |

A literal (`#1a1a1a`, `size="31"`) is accepted but opts that element out of the design system —
a theme swap will not touch it. Reach for one only when the deck needs a one-off.

Colour attributes (`color`, `fill`, `border`'s colour) also take four **colour functions** whose
arguments are tokens or single literals — they keep the token reference on both targets:

| Function | Example |
|---|---|
| `gradient(c1 c2 [c3 …] [, Ndeg])` | `fill="gradient(accent1 accent6, 135deg)"` — even stops, default `180deg` (top→bottom) |
| `shade(c, p%)` / `tint(c, p%)` | `fill="tint(accent1, 40%)"` — darker / lighter by HSL lightness, `0 < p ≤ 100` |
| `alpha(c, 0–1)` | `color="alpha(fg, 0.6)"` — a fraction, not a percentage |
| `radial(c1 c2 … [, at X% Y%])` | `fill="radial(accent4/30 bg/0, at 0% 100%)"` — first colour at the centre; centre defaults to the middle |

**Gradient stops take an opacity** as `colour/percent`, 0–100 with no `%`: `gradient(fg fg/0, 90deg)`
fades from `fg` to transparent. Literals too: `#000000/50`.

**Pattern fills** are a shape `fill` only: `pattern(<pattern> <fg> <bg>)`, both colours taking the
same `/percent` opacity — `fill="pattern(wide-diagonal-up accent5/60 bg/0)"` is a red hatch the
background shows through. Patterns: `horizontal` `vertical` `diagonal-up` `diagonal-down`
`wide-diagonal-up` `wide-diagonal-down` `cross` `diagonal-cross` `dots`. On text colour, borders,
connector lines, table cells, or backgrounds it is an error. In PPTX it is a real `a:pattFill`.

They do not nest (`shade(gradient(…), …)` is an error). A gradient border is two pieces:
`border="4 gradient(accent6 accent1)"`. If the same treatment repeats across slides, it belongs
in a theme variant (`variant="hero"`), not on every element.

## Variants present in all three built-in themes

| Where | Variants |
|---|---|
| text | `kicker` (18 / bold / `accent1`, letter-spaced, upper-cased by the theme) `lead` |
| shape | `card` `card-header` `card-body` `card-backdrop` `stat-card` `badge` |
| connector | `dashed` `accent` `plain` (no arrowhead); default is a 2px `muted` line with an end arrow |
| image | `framed` |
| table | `striped` |

An unknown `variant` is a **warning**, not an error: the element renders with the default
appearance and the build still exits 0 — which `--strict` turns into a failure. The warning
lists the variants the active theme actually has.

## Shapes worth copying

A stat card:

```html
<shape variant="stat-card" type="rounded-rect" fill="surface" w="fill" h="200" pad="lg">
  <text role="title" color="accent1" align="center">+38%</text>
  <p align="center">Monthly active users</p>
</shape>
```

Two columns of flow content:

```html
<row gap="lg">
  <col grow="1">
    <text role="heading">Now</text>
    <ul><li>Pricing experiments</li><li reveal>Onboarding redesign</li></ul>
  </col>
  <shape type="rect" fill="surface" w="360" h="200">
    <p align="center">a shape holds its own children</p>
  </shape>
</row>
```

A bar chart (one series — `series` colours every bar, `<point fill>` highlights one):

```html
<chart type="bar" unit="%" max="100" series="accent1" h="320">
  <point label="Web" value="62"/>
  <point label="iOS" value="28" fill="accent2"/>
</chart>
```

Several series share the categories (same labels, same order in each); a legend appears under
them. Bars group; `line`/`area`/`step` overlay, and a series may pick its own `type`:

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

Chart rules: `h` or `ratio` is required; with `<series>` the frame's `series=` is an error (each
series has its own `fill`, default `accent1`…`accent6` in order); never mix bare `<point>`s and
`<series>`; `<point fill>` is an error on line-family charts, and they draw no value labels.

A diagram — boxes with `id`, joined by connectors. Connectors are direct children of `<slide>`,
take no space in the flow, and are drawn from the measured boxes after layout, so they may name
a box declared later. In PPTX they are real connectors glued to the boxes:

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

- `id` lives on `<shape>` and `<image>`, unique per slide; a connector cannot cross slides.
- `auto` sides face the other box; `elbow`/`curved` between same-axis sides is a mid-step,
  between different axes an L, and between the same side (top → top) a U outside both boxes.
- `line` is the `border` grammar (`"2 dashed accent1"`) — no gradients, no `none`. `label` is
  one short line beside the path midpoint.
- PPTX glues the ends only to `rect`, `rounded-rect`, `diamond`, `ellipse`, and images; other
  presets get a line in the right place that does not follow a drag in PowerPoint.
- `flow` animates the dashes on the web presenter view only; PPTX shows the same dash, still.

A custom outline — `d` is drawn in a **0–100 box** stretched to the shape's box (axes stretch
independently). Absolute commands only: `M L H V C Q Z`; lowercase, `A`, `S`, `T` are errors.
Coordinates may leave 0–100. An open path is still filled, so lines need `fill="none"`:

```html
<stack w="560" h="280">
  <shape type="path" fill="pattern(wide-diagonal-up muted/35 bg/0)" border="none" w="fill" h="fill"
         d="M0,85 H40 V55 H72 V30 H100 V100 H0 Z"/>
  <shape type="path" fill="none" border="3 solid fg" w="fill" h="fill"
         d="M0,98 C15,85 25,90 40,78 C50,70 52,58 62,56 C70,55 75,40 85,36 C93,33 97,30 100,22"/>
</stack>
```

`cube`, `left-brace`, `right-brace` are drawn from the measured box (depth = 25% of the short
side); `cube` needs a definite size (`w`+`h` or `ratio`) and its text sits on the front face.
Braces are line shapes — use `fill="none"`; `rotate="90"` turns one into an under-brace.

Highlights and small units — `<mark>` takes no attributes (the theme owns the colour), and
`<span size>` may only be **smaller than or equal to** the surrounding text:

```html
<h1 size="44">EC2 gives you <mark>resizable compute</mark> in minutes.</h1>
<text role="title" size="56">750<span size="28">+</span></text>
```

Free placement — legal only on `layout="canvas"` or inside `<stack>`:

```html
<slide id="diagram" layout="canvas">
  <shape type="ellipse" at="center" w="240" h="240" fill="accent3"/>
  <text at="bottom-right" role="caption">placed by hand</text>
</slide>
```

## Motion

`enter` makes an element appear automatically when the slide starts, staggered in document order
by the theme's `motion` tokens (all built-ins: 600ms each, 90ms apart, after 150ms). Everything
inside an element with `enter` appears with it; a nested `enter` appears on its own; `enter="none"`
opts a child out of `<slide enter>`. In PPTX these are native entrance effects (`rise` = Float In,
`zoom` = Zoom, `wipe` = Wipe from left); connectors take `draw` (a wipe in PPTX) or `fade`.
Motion never moves anything — layout and PPTX coordinates are measured in the final state, and
the web's document view, overview, and print show everything.

`transition` on `<slide>` is how that slide comes in (`fade` cross-fades, `push` slides in from
the right). A theme without `motion` makes any `enter` a build error.

## Inverted slides

`<slide invert>` swaps `bg` ↔ `fg` and `surface` ↔ `muted` for that slide (accents stay). The
markup is unchanged — `color="fg"` simply becomes light on a dark slide — and the footer and page
number swap too. In PPTX it is a colour-map override, so theme colours still drive it.

## Per-tag depth

`${CLAUDE_PLUGIN_ROOT}/docs/reference/soml/` has one page per tag group, plus
`connector.md` and `enter.md` for diagrams and motion,
`appearance.md` (the colour/size/spacing cascade and its escape hatches) and
`layouts.md` (slot geometry and the placement-vs-appearance boundary). Those pages are Korean.
