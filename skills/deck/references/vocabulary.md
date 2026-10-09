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
| Inline | `strong` `em` `code` `a` `br` `span` |
| Containers | `row` `col` `grid` `stack` |
| Leaf | `spacer` `shape` `image` |
| Table | `table` > `tr` > `th` `td` |
| Chart | `chart` > `point` |

`<h1>`/`<h2>`/`<h3>`/`<p>`/`<blockquote>`/`<pre>` are shorthands that carry a role implicitly;
`<text role="…">` says it outright. Use whichever reads better — they compile to the same node.

## Attributes by tag

Shared groups, referenced below as *box* and *child*:

- **box** — `pad` `w` `h` `min-w` `min-h` `max-w` `max-h` `ratio`
- **child** (placement inside the parent) — `grow` `shrink` `basis` `span` `self` `order`
  `at` `x` `y` `style`

Every tag that can be a direct child of `<slide>` also takes `slot`.

| Tag | Own attributes | Plus |
|---|---|---|
| `text`, `h1`–`h3`, `p`, `blockquote`, `pre` | `role` `variant` `size` `color` `align` `anchor` `w` `min-w` `max-w` (`pre` adds `lang`) | child, slot |
| `row`, `col` | `gap` `align` `justify` `wrap` `pad` | box, child, slot |
| `grid` | `cols` `rows` + everything `row` takes | box, child, slot |
| `stack` | `align` `justify` `pad` (no `gap`/`wrap`/`cols`/`rows`) | box, child, slot |
| `shape` | `type` `id` `alt` `variant` `fill` `border` `radius` `shadow` `opacity` `rotate` `flip` | box, child, slot |
| `image` | `src` `fit` `alt` `variant` `border` `radius` `shadow` `opacity` `rotate` `flip` | box, child, slot |
| `table` | `alt` `variant` | box, child, slot |
| `th`, `td` | `colspan` `rowspan` `fill` `border` `color` `align` `anchor` | — |
| `chart` | `type` `unit` `max` `series` `alt` | box, child, slot |
| `point` | `label` `value` `fill` | — |
| `ul`, `ol` | — | child, slot |
| `li` | `reveal` | — |
| `a` | `href` | — |
| `span` | `color` `weight` | — |
| `spacer` | none of its own — it is not a container | child, slot |
| `note`, `tr`, `strong`, `em`, `code`, `br` | none | — |

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
| `shape type` | `rect` `rounded-rect` `ellipse` `diamond` `cylinder` `chevron` `arrow-right` `hexagon` `parallelogram` `callout` `star` `cloud` |
| `chart type` | `bar` |
| `image src` | a **project-root-relative** path to one file; `.png` `.jpg` `.jpeg` only. Absolute paths, `../` outside the project, URLs, and `data:` are all errors — the deck must carry its own images. `assets/` is the convention, not a rule |

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

They do not nest (`shade(gradient(…), …)` is an error). A gradient border is two pieces:
`border="4 gradient(accent6 accent1)"`. If the same treatment repeats across slides, it belongs
in a theme variant (`variant="hero"`), not on every element.

## Variants present in all three built-in themes

| Where | Variants |
|---|---|
| text | `kicker` `lead` |
| shape | `card` `card-header` `card-body` `card-backdrop` `stat-card` `badge` |
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

A bar chart (the only chart type):

```html
<chart type="bar" unit="%" max="100" series="Share">
  <point label="Web" value="62" fill="accent1"/>
  <point label="iOS" value="28" fill="accent2"/>
</chart>
```

Free placement — legal only on `layout="canvas"` or inside `<stack>`:

```html
<slide id="diagram" layout="canvas">
  <shape type="ellipse" at="center" w="240" h="240" fill="accent3"/>
  <text at="bottom-right" role="caption">placed by hand</text>
</slide>
```

## Per-tag depth

`${CLAUDE_PLUGIN_ROOT}/docs/reference/soml/` has one page per tag group, plus
`appearance.md` (the colour/size/spacing cascade and its escape hatches) and
`layouts.md` (slot geometry and the placement-vs-appearance boundary). Those pages are Korean.
