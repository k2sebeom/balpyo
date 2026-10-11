# SOML vocabulary

Everything legal in a `.slide` file, at a glance. The vocabulary is **closed**: a name absent
from these tables is a build error, not a fallback, and the error lists what *is* legal at that
position. When something here disagrees with the binary, the binary is right. Per-element
behaviour and snippets are in `elements.md`; how values are decided is in `appearance.md`.

## File

One file is one slide: exactly one root `<slide>`. Comments (`<!-- … -->`) are allowed
anywhere. There is no front matter — slide metadata is attributes on `<slide>`.

## Layouts and their slots

| `layout` | Slots (in order) | For |
|---|---|---|
| `title` | `kicker` `title` `subtitle` `author` `date` | the cover |
| `section-divider` | `title` `subtitle` | a chapter break |
| `content` | `title` `body` | the default workhorse (also what an omitted `layout` means) |
| `two-column` | `title` `left` `right` | side-by-side comparison |
| `image-full` | `media` `caption` | a full-bleed image (no stage margin) |
| `image-text` | `title` `media` `body` | image beside prose |
| `quote` | `quote` `attribution` | a pull quote |
| `code` | `title` `code` `caption` | a code listing |
| `stat` | `title` `value` `label` `body` | one big number |
| `canvas` | — | free placement with `x`/`y` |
| `end` | `title` `contact` | the closing slide |

Themes cannot add layout names. Rules:

- `slot="<name>"` goes on a **direct child** of `<slide>` only — a slot on a grandchild is an
  error, and so is a slot the layout does not declare (the error lists that layout's slots).
- One `slot=` anywhere puts that slide in **slot mode**; a slide with none stays in **flow
  mode** (children stacked top to bottom). The choice is per slide.
- Two elements in one slot: write one container (`<col slot="body">`) holding both.
- **A slot child that is a single frame** (`<text>`, a block, a list, `<shape>`, `<image>`)
  becomes a real PowerPoint placeholder, which keeps a slotted deck
  editable in PowerPoint's outline view. A **container** carrying the slot (`<col slot="left">`)
  is a web area only — no placeholder.
- **`justify` on a slot-mode slide is a build error.** Vertical rhythm there belongs to the
  theme. Use `<slide justify>` on flow-mode slides only.
- **Horizontal placement in a slot belongs to the theme.** Where the theme centres a slot (see
  `appearance.md`), the child shrinks to its content and `align` has nothing to move — leave it
  off. In a slot the theme leaves at full width, `align` moves the text as usual
  (`<p slot="attribution" align="end">`). To centre text inside a specific width, give that
  width to the text: `<text w="140" align="center">`.
- `slot=` on `layout="canvas"` is an error (it declares no slots). If the active theme has no
  slot CSS for a layout, a slot-mode slide warns and renders in flow mode (the built-in themes
  have slot CSS for all ten slotted layouts).
- In flow mode, `two-column` and `image-text` put their children into the two columns in
  document order.

## Tags and what they may contain

| Tag | Kind | May contain |
|---|---|---|
| `slide` | root | every block, list, container, leaf, `table`, `chart`, `connector`, `note` — no bare text, no inline tags |
| `note` | speaker note (one per slide) | text only |
| `text` | text element (explicit role) | text and inline tags |
| `h1` `h2` `h3` `p` `blockquote` | blocks (implicit role) | text and inline tags (blocks do not nest) |
| `pre` | code block | character data only (no tags; use CDATA for markup) |
| `ul` `ol` | lists | `li` only |
| `li` | list item | text and inline tags (no nested lists) |
| `strong` `em` `code` `a` `span` `mark` | inline | text and inline tags (`a` may not contain `a`) |
| `br` | inline, void | nothing — write `<br/>` |
| `row` `col` `grid` `stack` | containers | `text`, blocks, lists, containers, `shape`, `image`, `table`, `chart`, `spacer` — no bare text |
| `spacer` | void | nothing — write `<spacer/>` |
| `shape` | leaf with content | `text`, `h1`–`h3`, `p`, `blockquote`, `pre`, `row`, `col`, `grid`, `stack` (lists, images, tables, charts go through a container) |
| `image` | void | nothing |
| `table` | table | `tr` only |
| `tr` | row | `th` `td` only |
| `th` `td` | cells | text and inline tags (no blocks, no lists, no shapes) |
| `chart` | chart | `point`s, or `series` (never both) |
| `series` | chart series | `point` only |
| `point` | data point, void | nothing |
| `connector` | line between two boxes, void | nothing (direct child of `<slide>` only) |

`<h1>`/`<h2>`/`<h3>`/`<p>`/`<blockquote>`/`<pre>` are shorthands that carry a role implicitly
(`h1` title, `h2`/`h3` heading, `p`/`blockquote`/`li` body, `pre` code); `<text role="…">`
says it outright and is the only tag that takes `role`. There is no caption block — write
`<text role="caption">`. HTML habits get a suggestion: `<img>` → `<image>`, `<b>` → `<strong>`,
`<i>` → `<em>`, `<h4>`–`<h6>` → `<h3>`.

## Attributes by tag

Shared groups, referenced below:

- **box** — `pad` `w` `h` `min-w` `min-h` `max-w` `max-h` `ratio`
- **child** (placement inside the parent) — `grow` `shrink` `basis` `span` `self` `order`
  `at` `x` `y` `style`
- **slot** and **enter** — on every tag that can be a direct child of `<slide>` (except
  `connector`, which has its own `enter`, and `note`)

| Tag | Own attributes | Plus |
|---|---|---|
| `slide` | `layout` `id` `skip` `justify` `enter` `transition` `invert` | — |
| `text` | `role` `variant` `size` `color` `align` `anchor` `w` `min-w` `max-w` | child, slot, enter |
| `h1`–`h3`, `p`, `blockquote` | `variant` `size` `color` `align` `anchor` `w` `min-w` `max-w` | child, slot, enter |
| `pre` | `lang` + the same as `p` | child, slot, enter |
| `ul`, `ol` | `size` `color` `align` `anchor` `w` `min-w` `max-w` — the whole list's look; items inherit it (no `variant`) | child, slot, enter |
| `li` | `reveal` | — |
| `row`, `col` | `gap` `align` `justify` `wrap` | box, child, slot, enter |
| `grid` | `cols` `rows` `gap` `align` `justify` `wrap` | box, child, slot, enter |
| `stack` | `align` `justify` (no `gap`/`wrap`/`cols`/`rows`) | box, child, slot, enter |
| `spacer` | none of its own (no `w`/`h` — size it with `grow` or `basis`) | child, slot, enter |
| `shape` | `type` `id` `alt` `d` `notch` `variant` `fill` `border` `radius` `shadow` `opacity` `rotate` `flip` | box, child, slot, enter |
| `image` | `src` `id` `fit` `alt` `variant` `border` `radius` `shadow` `opacity` `rotate` `flip` (no `fill`) | box, child, slot, enter |
| `table` | `alt` `variant` (no box attributes — size its container) | child, slot, enter |
| `tr` | none | — |
| `th`, `td` | `colspan` `rowspan` `fill` `border` `color` `align` `anchor` | — |
| `chart` | `type` `unit` `max` `series` `alt` | box, child, slot, enter |
| `series` | `name` (required) `fill` `type` | — |
| `point` | `label` (required) `value` (required) `fill` | — |
| `connector` | `from` `to` `route` `from-side` `to-side` `line` `start` `end` `variant` `label` `enter` `flow` | — (no box, no child, no slot) |
| `a` | `href` (required) | — |
| `span` | `color` `weight` `size` | — |
| `note`, `strong`, `em`, `code`, `br`, `mark` | none | — |

Where the child attributes work:

- `x`/`y` — only on children of `<stack>` and direct children of a `layout="canvas"` slide.
- `at` — only on children of `<stack>`; cannot be combined with `x`/`y`.
- `span` — only on children of `<grid>`.
- `grow` `shrink` `basis` `self` `order` — on children of `row`/`col`/`grid` and direct
  children of a flow slide (the flow stage is a vertical flex column).

## Value vocabularies

| Attribute | Legal values |
|---|---|
| `role` | `title` `heading` `body` `caption` `code` (default `body`) |
| `align` (text, list, cell) | `start` `center` `end` `justify` — logical names, not `left`/`right` |
| `align` (container, cross axis) | `start` `center` `end` `stretch` `baseline` |
| `self` | same as container `align` |
| `justify` (container main axis, `<slide>`) | `start` `center` `end` `between` `around` `evenly` — not `space-between` |
| `anchor` (text in a shape, cell) | `top` `center` `bottom` |
| `at` | `top-left` `top-center` `top-right` `center-left` `center` `center-right` `bottom-left` `bottom-center` `bottom-right` |
| `fit` | `contain` (default) `cover` |
| `skip`, `invert`, `flow` | valueless, `"true"`, or `"false"` |
| `wrap` | valueless or `"true"` only — remove it to turn it off |
| `reveal` | valueless (next step on this slide), or an integer ≥ 1 (`reveal="2"`; equal numbers appear together). Not `"true"` |
| `shape type` | `rect` `rounded-rect` `ellipse` `diamond` `cylinder` `chevron` `arrow-right` `hexagon` `parallelogram` `callout` `star` `cloud` `triangle` `cube` `left-brace` `right-brace` — plus `path` (custom outline, needs `d`). Required |
| `notch` (`chevron` only) | a percentage of the width, `0%` < notch ≤ `50%`; default `20%` |
| `radius` (`rounded-rect` and `image` only) | `sm` `md` `lg` or a number |
| `shadow` | `soft` (the only shadow token) |
| `opacity` | a number from 0 to 1 (no `%`) |
| `rotate` | unitless degrees, clockwise, negative allowed (no `deg`) |
| `flip` | `h` `v` `hv` |
| `border` | `none`, or `<width> <style> <colour>` with style `solid` `dashed` `dotted` (`"1 solid muted"`), or `<width> gradient(…)` |
| `chart type` | `bar` `line` `area` `step` (required) |
| `series type` | `line` `area` `step` (line-family charts only) |
| `pre lang` | `go` `xml` `soml` (= `xml`) `json` `bash`; omit for no highlighting |
| `span weight` | `normal` `bold` |
| `enter` (content, `<slide>`) | `fade` `rise` `zoom` `wipe` `none` |
| `enter` (`<connector>`) | `draw` `fade` `none` |
| `transition` | `fade` `push` `none` |
| connector `route` | `straight` (default) `elbow` `curved` |
| connector `from-side` `to-side` | `auto` (default) `top` `right` `bottom` `left` |
| connector `start` `end` | `none` `arrow` `dot` |
| `image src` | a project-root-relative path, `.png` `.jpg` `.jpeg` `.svg`, or `theme:<path>` — see `elements.md` |

Attributes may not be empty: `variant=""`, `id=""`, `alt` with no value on a shape or table,
`layout=""` are all errors. (On `<image>`, `alt=""` is legal and means decorative.)

## Sizes and spacing

Sizes and spacing do **not** share one value vocabulary:

| Attribute | Values |
|---|---|
| `w` `h` `min-w` `min-h` `max-w` `max-h` | number · `N%` · `fill` |
| `gap` | space token (`md`) · number · `N%` |
| `pad` | 1–4 values (CSS order: top, right, bottom, left), each a space token · number · `N%` — `%` only on containers, not on `shape`/`image` |
| `grow` `shrink` | integer ≥ 0 |
| `span` | integer ≥ 1 |
| `order` | integer (negative allowed) |
| `basis` | number · `N%` (**not** `fill` — use `100%`) |
| `x` `y` | number · `N%` · negative |
| `ratio` | `W:H`, positive integers (`16:9`) |
| `cols` `rows` (`grid` only) | a single integer = that many equal tracks (`cols="3"`), or a list of two or more tracks from number · `Nfr` · `N%` · `auto` (`cols="2fr 1fr"`) |
| `size` (text) | a number > 0 (no `%`) |

Numbers are **unitless logical px**. A `px`/`pt`/`em` suffix is an error, `%` is the only
suffix, and a negative length is an error everywhere except `x`/`y`. A space token in `w` is
an error; a number in `gap` is fine. `cols="200"` means two hundred tracks — for one sized
track write `cols="1fr"`.

`fill` is **100% of that axis**, not `flex-grow: 1` — `w="fill"` fills the parent, `grow="1"`
splits the leftover space. They are different results and both are often right.

## Tokens — write these, not literals

| Kind | Names (every theme has these) |
|---|---|
| `color` | `fg` `bg` `muted` `surface` `accent1` `accent2` `accent3` `accent4` `accent5` `accent6` |
| `font` | `heading` `body` `mono` (chosen by role — there is no `font=` attribute) |
| `space` (`gap`, `pad`) | `xs` `sm` `md` `lg` `xl` `2xl` |
| `radius` | `sm` `md` `lg` |
| `shadow` | `soft` |

A theme may add names (a project theme's README says so); an unknown token is an error that
lists the theme's names. A literal (`#1a1a1a`, `#abc`, `size="31"`) is accepted but opts that
element out of the design system — a theme swap will not touch it, and PowerPoint's theme-colour
editor will not move it. Reach for one only when the deck needs a one-off.

## Colour functions

Colour attributes (`color`, `fill`, `border`'s colour, `<series fill>`, `<point fill>`) also
take five **colour functions** whose arguments are tokens or single literals — they keep the
token reference on both targets:

| Function | Example |
|---|---|
| `gradient(c1 c2 [c3 …] [, Ndeg])` | `fill="gradient(accent1 accent6, 135deg)"` — at least two colours, even stops, CSS angle, default `180deg` (top→bottom) |
| `radial(c1 c2 … [, at X% Y%])` | `fill="radial(accent4/30 bg/0, at 0% 100%)"` — first colour at the centre, last at the farthest side; centre defaults to the middle |
| `shade(c, p%)` / `tint(c, p%)` | `fill="tint(accent1, 40%)"` — darker / lighter by HSL lightness, `0 < p ≤ 100`; hue and saturation are kept (so tints stay vivid) |
| `alpha(c, 0–1)` | `color="alpha(fg, 0.6)"` — a fraction, not a percentage |

**Gradient stops take an opacity** as `colour/percent`, 0–100 with no `%`: `gradient(fg fg/0, 90deg)`
fades from `fg` to transparent. Literals too: `#000000/50`.

Functions do not nest (`shade(gradient(…), …)` is an error). A gradient border is two pieces:
`border="4 gradient(accent6 accent1)"`. Text may take a gradient colour. Diagonal gradient
angles drift slightly between the targets on non-square boxes — keep diagonals on square-ish
shapes. If the same treatment repeats across slides, it belongs in a theme variant
(`variant="hero"`), not on every element.

## Pattern fills

A shape `fill` only: `pattern(<pattern> <fg> <bg>)`, both colours taking the same `/percent`
opacity — `fill="pattern(wide-diagonal-up accent5/60 bg/0)"` is a red hatch the background shows
through. Patterns: `horizontal` `vertical` `diagonal-up` `diagonal-down` `wide-diagonal-up`
`wide-diagonal-down` `cross` `diagonal-cross` `dots`. On text colour, borders, connector lines,
table cells, chart fills, or backgrounds it is an error. In PPTX it is a real pattern fill,
editable in PowerPoint's "Pattern fill".

## Character references and CDATA

The closed set of references (each ending in `;`): `&amp;` `&lt;` `&gt;` `&quot;` `&apos;`
`&#NNN;` `&#xHHH;`. `&nbsp;` and other HTML entities are errors. An `&` that does not look like
a reference is literal (`R&D`, `?a=1&b=2`), and `<` only opens a tag before a letter, `/`, `!--`
or `![CDATA[` (so `a < b` is fine). `<![CDATA[ … ]]>` is taken verbatim anywhere text is
allowed — the way to show markup inside `<pre>` or `<code>`.

## Variants present in all three built-in themes

| Where | Variants |
|---|---|
| text | `kicker` (small, bold, `accent1`, letter-spaced, upper-cased by the theme) `lead` (larger body) |
| shape | `card` `card-header` `card-body` `card-backdrop` `stat-card` `badge` |
| image | `framed` |
| table | `striped` (alternate body rows filled; built into the language, its colour from the theme) |
| connector | `dashed` `accent` `plain` (no arrowhead); default is a 2px `muted` line with an end arrow |

Variant names are per element kind (`<image variant="card">` is not a thing). An unknown
`variant` is a **warning**, not an error:
`variant "kickr" is not in the theme (did you mean "kicker"?) — continuing with the default appearance (variants in this theme: kicker, lead)`.
The element renders with the default appearance and the build still exits 0 — which `--strict`
turns into a failure. What each variant sets in each built-in theme is in `appearance.md`.
