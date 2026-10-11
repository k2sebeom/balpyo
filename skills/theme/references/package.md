# The theme package

Four file groups whose jobs do not overlap, plus two optional ones. The complete built-in themes
are in `builtin-themes.md`.

| File | Owns |
|---|---|
| `theme.json` | identity — name, parent, the layouts implemented, deck chrome, slide backgrounds |
| `tokens.json` | **the names of values** — what `accent1`, `md`, `heading` are, plus the motion rhythm |
| `components.json` | **appearance of roles and variants** — how big `role="body"` is, what colour a code keyword gets |
| `layouts/*.css` | **placement** — where a layout puts its children |
| `README.md` | optional, not loaded — the theme's usage guide for deck authors, printed verbatim by `balpyo theme info` |
| `assets/` | optional pictures the theme owns — background images and the logo (PNG/JPEG), plus pictures slides use via `theme:` (PNG/JPEG/SVG) |

A theme is data only — JSON, CSS, pictures. It cannot run code and cannot bundle fonts.

**Key checking.** `tokens.json` and `components.json` reject unknown keys (the error names the
file, `line:col`, and the JSON path, e.g. `unknown key "lineHeight" at text.variants["stat"].lineHeight`);
any key starting with `$comment` is a comment, anywhere. `theme.json` is **not** checked — a
misspelt key there is silently ignored. Key names match case-insensitively. Wrong value types are
errors (`shape.variants["card"].fill must be a string, not number`); `weight`, `tabSize` and the
motion numbers must be integers.

## `theme.json`

```json
{
  "name": "corp",
  "extends": "minimal",
  "chrome": [
    { "kind": "footer", "align": "center", "role": "caption", "skipLayouts": ["title"] }
  ]
}
```

| Key | Value | Notes |
|---|---|---|
| `name` | string | must equal the directory name under a project's `themes/`; for an installed theme it *is* the installed name; empty is an error |
| `extends` | a theme name or `""` | the parent. Chains are allowed (`corp-dark` → `corp` → `minimal`); a missing parent (`theme "corp" extends theme "minimalx", which does not exist (available themes: …)`) and cycles (`theme.json extends forms a cycle: a → b → a`) are errors. A project theme may extend a built-in, installed, or project theme; an installed theme only a built-in or installed one |
| `layouts` | list of layout names | the layouts this theme implements; must match the `layouts/*.css` files both ways. Inherited — omit it when extending |
| `chrome` | array of declarations | below |
| `backgrounds` | array of background declarations (optional) | below; inherited as a list — if the child writes it, the child's list wins |

A standalone theme lists all eleven standard layouts:

```json
["title","section-divider","content","two-column","image-full","image-text",
 "quote","code","stat","canvas","end"]
```

The loader only checks that the list and the CSS files agree — it does not insist on eleven, and
it accepts names outside the catalogue (which no slide can request). A theme missing a layout
makes every slide using it fail; nothing falls back to another theme. Write all eleven.

### Slot names per layout

Slot CSS must name exactly these areas (the slot vocabulary is fixed by the language):

| Layout | Slots |
|---|---|
| `title` | `kicker` `title` `subtitle` `author` `date` |
| `section-divider` | `title` `subtitle` |
| `content` | `title` `body` |
| `two-column` | `title` `left` `right` |
| `image-full` | `media` `caption` |
| `image-text` | `title` `media` `body` |
| `quote` | `quote` `attribution` |
| `code` | `title` `code` `caption` |
| `stat` | `title` `value` `label` `body` |
| `end` | `title` `contact` |
| `canvas` | — (no slot file) |

### Chrome declarations

Chrome is the footer, the page number, and the logo. For the two text kinds the vocabulary is
**split between two files**: the deck owns the content and the on/off (`balpyo.json`'s `footer`
string and `pageNumber` bool), the theme owns placement, alignment, role, and exclusions. Both
are needed for anything to render; config-only produces a warning
(`theme "corp" has no footer chrome declaration, so footer cannot be used …`) and no chrome.
**The logo is the theme's alone** — declaring it turns it on for every slide outside
`skipLayouts`.

| Key | Values | Notes |
|---|---|---|
| `kind` | `footer` · `pageNumber` · `logo` | declaring the same `kind` twice is an error |
| `align` | `start` · `center` · `end` | **mandatory** — position inside the band. Two chrome kinds on the same `align` overlap (a `--check-layout` warning) |
| `role` | one of the five text roles | **mandatory** for `footer`/`pageNumber`, an error on `logo` — chrome has no size or colour of its own; reuse `caption` |
| `skipLayouts` | layout names of **this** theme (optional) | a name that is not a layout here is an error |
| `image` · `height` · `alt` | `logo` only — a picture under `assets/`, logical px height, alt text | `image` and `height` are required; width follows the picture's ratio. On a text kind these keys are an error |
| `format` | `pageNumber` only (optional) — text holding `{n}` exactly once, optionally `{total}` | `"{n} / {total}"` renders `3 / 13` |

```json
{ "kind": "logo", "align": "center", "image": "assets/logo.png", "height": 24,
  "alt": "Brand logo", "skipLayouts": ["title"] }
```

Chrome lives in the 64px band below the content box, so it never overlaps the body; its order
is fixed (footer, page number, logo). Text chrome never wraps — a footer too long for the band
warns under `--check-layout`. Page numbers count from 1 and **count excluded slides too** (the
PPTX uses a live slide-number field, and no viewer can count a cover out). `<slide skip>` is
different: that slide leaves the deck entirely and the following numbers shift up. The built-in
themes skip `title`, `section-divider`, `image-full`, and `end`; keep `image-full` in
`skipLayouts` (chrome would sit on the full-bleed picture).

With `format`, only `{n}` stays a live PPTX field. The text around it, `{total}` included, is
written at build time, so a slide added in PowerPoint renumbers but the total stays stale until
the next build. There is no zero-padding (`03`). A bad format is a load error:
`pageNumber format "{page}" needs {n} exactly once and may use {total} — e.g. "{n} / {total}"`.

Chrome merges by `kind`: a child that writes `footer` replaces the parent's whole footer entry
(including its `skipLayouts`) and inherits the parent's `pageNumber` untouched.

### Backgrounds

```json
"backgrounds": [
  { "layouts": ["title"], "image": "assets/cover.png" },
  { "layouts": ["section-divider"], "fill": "gradient(accent1 accent6, 135deg)" },
  { "layouts": ["end"], "fill": "bg", "overlay": "radial(accent1/35 bg/0, at 0% 100%)" },
  { "fill": "surface" }
]
```

| Key | Values | Notes |
|---|---|---|
| `layouts` | layout names of this theme (optional) | **omitted = the default background** for every other layout |
| `fill` | a token, a token variant, or a token gradient | the slide's base fill (not `none` — remove the declaration instead) |
| `image` | a picture under `assets/` | covers the slide (aspect kept, centre-cropped — like `fit="cover"`) |
| `overlay` | a colour value — usually a gradient fading to transparent (optional) | one layer painted **over** the fill or picture: a dark band over a photo, a corner glow |

- One declaration is `fill` **or** `image`, never both — a PPTX slide background holds one
  fill, so a fill under a picture would vanish there. Neither is an error too.
- A layout appears in **one** declaration only, and there is one default. Overlaps are errors —
  there is no precedence rule to remember.
- **`overlay` is how a picture gets a tint.** `{ "image": "assets/cover.jpg", "overlay":
  "gradient(fg fg/0, 90deg)" }` darkens the photo from the left. On the web it is the top
  background layer. In the PPTX it is a **locked full-slide rectangle** behind all content, and
  it cannot be selected or moved.
- Gradient stops take an opacity as `colour/percent` (`fg/0`, `accent1/35` — 0–100, no `%`), and
  `radial(c1 c2 … [, at X% Y%])` is a radial gradient (first colour at the centre). An
  off-centre radial is drawn correctly by PowerPoint but **from the centre by OnlyOffice**, so
  check corner glows in PowerPoint.
- On a `<slide invert>` the background stays, but a token fill shows that token's swapped colour.
- A layout with no background stays the `bg` colour.
- On the web it is the slide's CSS background; in the PPTX it is the slide's background ("Format
  Background" in PowerPoint), not a movable shape. Colours stay theme colours.

Pictures (backgrounds and the logo) must live under the theme's `assets/` (paths written
`assets/…`), be real PNG/JPEG files (SVG is not allowed there) whose extension matches the
content and that have an intrinsic size. Only referenced pictures are shipped (the web output
puts them under `themes/<theme>/assets/`). An inherited declaration's picture belongs to the
theme that has it; a child file at the same path wins.

### Pictures for slides — `theme:`

Slides may use pictures too: `<image src="theme:icons/a.png">` is `assets/icons/a.png` looked
up through the active theme's `extends` chain, child first — for built-in, project, and
installed themes alike. This is how a theme ships an icon or illustration library: a child theme
overrides one icon by placing a file at the same path, and the deck's `src` never changes. The
paths under `assets/` become public vocabulary — renaming or removing a file breaks every deck
that uses it. Loading only indexes `assets/`; bytes are read for referenced files only, so a
large library costs nothing until used. Icons may be **SVG**: self-contained (no script, no
`<foreignObject>`, no event attributes, no external references), with px `width`/`height` on the
root `<svg>`. The PPTX carries the vector plus a browser-rendered PNG fallback.

## `tokens.json`

Ordered **arrays** — order determines CSS variable order, `clrScheme` assembly, and the order
names appear in diagnostics. Required *names* (the loader checks names, not values):

| Category | Keys per entry | Required names | PPTX |
|---|---|---|---|
| `color` | `name` `rgb` `slot` `ref` | `fg` `bg` `muted` `surface` `accent1`…`accent6` | `clrScheme`, 12 places |
| `font` | `name` `latin` `ea` `generic` `scheme` | `heading` `body` `mono` | `fontScheme` major/minor |
| `space` | `name` `px` | `xs` `sm` `md` `lg` `xl` `2xl` | none (baked into coordinates) |
| `radius` | `name` `px` | `sm` `md` `lg` | none (OOXML has no radius) |
| `shadow` | `name` `dist` `dir` `blur` `color` `alpha` | `soft` | baked per shape; only the colour stays a reference |
| `motion` | `duration` `stagger` `delay` `transition` `transitionDuration` | (one optional object) | timing baked into each effect |

```json
{
  "color":  [{ "name": "accent3", "rgb": "15803D", "slot": "accent3", "ref": "accent3" }],
  "font":   [{ "name": "body", "latin": "Noto Sans", "ea": "Noto Sans CJK KR",
               "generic": "sans-serif", "scheme": "minor" }],
  "space":  [{ "name": "md", "px": 24 }],
  "radius": [{ "name": "md", "px": 12 }],
  "shadow": [{ "name": "soft", "dist": 4, "dir": 90, "blur": 16,
               "color": "fg", "alpha": 0.15 }],
  "motion": { "duration": 600, "stagger": 90, "delay": 150,
              "transition": "fade", "transitionDuration": 400 }
}
```

### Colours

A colour token's four fields are the token-colour-preservation contract: `name` is what authors
write, `rgb` is the colour (six hex digits, no `#`), `slot` is the `clrScheme` place it sits in,
`ref` is what shapes and runs reference it by — the two can differ (slot `dk1` is referenced as
`tx1`).

| Slot | Convention | `ref` |
|---|---|---|
| `dk1` | `fg` — **even in a dark theme** | `tx1` |
| `lt1` | `bg` — even in a dark theme | `bg1` |
| `dk2` | `muted` | `tx2` |
| `lt2` | `surface` | `bg2` |
| `accent1`…`accent6` | the six accents | same name |
| `hlink`, `folHlink` | unused by the built-ins — the only two free slots | same name |

A colour token **without** a slot is an error
(`… has no clrScheme slot — every color token must follow PPTX theme color edits (free slots: hlink, folHlink)`),
an unknown slot name is an error, and two tokens sharing a slot is an error
(`tokens.json clrScheme slot "accent1" is shared by color tokens "accent1" and "brand" — …`).
So a theme has at most twelve colour names, which is why child themes recolour rather than
extend. Unfilled slots borrow the nearest token (links → `accent1`) — a slot is never left
empty, or PowerPoint's colour editor offers an off-theme colour. Links in decks are always
`accent1`.

### Fonts

- `scheme` is `"major"` (headings), `"minor"` (body) or `""`. Exactly one `major` and one `minor`
  token are needed — otherwise the **PPTX build** fails (the loader does not catch it). Those two
  become PowerPoint's theme fonts and follow PowerPoint font changes; a `""` token (the
  built-ins' `mono`) is written as a literal typeface.
- `latin` is the Latin family, `ea` the East Asian one, `generic` the CSS generic family
  (`sans-serif`, `serif`, `monospace`); the web font stack is `latin, ea, generic`.
- **Both font slots must point at the same family style.** The browser asks the system for a
  substitute when a glyph is missing; PowerPoint and LibreOffice use the name in `ea` literally.
  This bites hardest on `mono`: a monospace Latin needs a **monospace CJK** `ea`
  (`Noto Sans Mono CJK KR`).
- Extra font tokens (`display`) are allowed but reachable only through a role's `font` — deck
  authors have no font attribute.

### Motion

The deck's entrance rhythm in ms, plus the default slide transition. The nth `enter` on a slide
starts at `delay + n × stagger` and runs for `duration` (> 0; `stagger`, `delay` ≥ 0).
`transition` (`fade` · `push` · `none`, omitted = none) is what every slide does unless it says
`transition=`. `transitionDuration` defaults to `duration`; the PPTX folds it to the nearest
speed (≤600ms fast, ≤875ms medium, else slow). It is one object, not an array, so a child that
writes it replaces it **whole**. A theme without `motion` loads, but any deck using `enter` then
fails (`<p> has enter="rise", but theme "corp" defines no motion — add "motion": { "duration": 600, "stagger": 90, "delay": 150 } to its tokens.json`).
All built-ins carry 600 / 90 / 150 and no transition.

## `components.json`

All five text roles must be present and complete. Every colour in the file — roles, defaults,
variants, images, tables, code — must be a reference to **this theme's** tokens: a literal
`#RRGGBB` (or a gradient with a literal stop) is a load error even when no slide uses that
variant, because it cuts the PPTX theme reference.

```json
{
  "text": {
    "roles": [
      { "name": "title",   "font": "heading", "size": 56, "weight": 700,
        "lineHeight": 1.2, "color": "fg", "align": "start" },
      { "name": "heading", "font": "heading", "size": 36, "weight": 700,
        "lineHeight": 1.3, "color": "fg", "align": "start" },
      { "name": "body",    "font": "body", "size": 24, "weight": 400,
        "lineHeight": 1.5, "color": "fg", "align": "start" },
      { "name": "caption", "font": "body", "size": 18, "weight": 400,
        "lineHeight": 1.4, "color": "muted", "align": "start" },
      { "name": "code",    "font": "mono", "size": 20, "weight": 400,
        "lineHeight": 1.5, "color": "fg", "align": "start" }
    ],
    "componentDefaults": [
      { "tag": "h3", "size": 28 },
      { "tag": "blockquote", "color": "muted" },
      { "layout": "quote", "slot": "quote", "size": 40, "color": "fg" },
      { "layout": "stat",  "slot": "value", "size": 96, "weight": 700, "color": "accent1" },
      { "layout": "stat",  "slot": "label", "size": 28, "weight": 700 }
    ],
    "variants": [
      { "name": "kicker", "size": 18, "weight": 700, "color": "accent1",
        "tracking": 0.1, "case": "upper" },
      { "name": "lead",   "size": 32 }
    ],
    "mark": { "highlight": "tint(accent4, 75%)" }
  },
  "shape": {
    "default": { "fill": "surface", "border": "1 solid muted", "radius": "md" },
    "variants": [
      { "name": "card",          "fill": "bg", "radius": "lg" },
      { "name": "stat-card",     "border": "none", "radius": "lg", "shadow": "soft" }
    ]
  },
  "connector": {
    "default": { "line": "2 solid muted", "start": "none", "end": "arrow" },
    "variants": [
      { "name": "dashed", "line": "2 dashed muted" },
      { "name": "accent", "line": "2 solid accent1" },
      { "name": "plain",  "end": "none" }
    ]
  },
  "image": {
    "default": {},
    "variants": [
      { "name": "framed", "border": "1 solid muted", "radius": "md", "shadow": "soft" }
    ]
  },
  "table": {
    "cell":   { "border": "1 solid muted" },
    "header": { "fill": "accent1", "color": "bg", "weight": 700 },
    "stripe": "surface"
  },
  "code": {
    "tabSize": 2,
    "tokens": [
      { "class": "keyword", "color": "accent6" },
      { "class": "name",    "color": "accent1" },
      { "class": "attr",    "color": "accent2" },
      { "class": "string",  "color": "accent3" },
      { "class": "number",  "color": "accent4" },
      { "class": "comment", "color": "muted" }
    ]
  }
}
```

### Allowed keys

| Section | Keys |
|---|---|
| `text.roles[]` | `name` `font` `size` `weight` `lineHeight` `color` `align` `tracking` `case` |
| `text.componentDefaults[]` | `tag` **or** `layout` + `slot`; then `size` `weight` `color` `tracking` `case` |
| `text.variants[]` | `name` `size` `weight` `color` `tracking` `case` |
| `text.mark` | `highlight` |
| `shape.default` | `fill` `border` `radius` |
| `shape.variants[]` | `name` `fill` `border` `radius` `shadow` |
| `image.default` | none — it must be `{}` |
| `image.variants[]` | `name` `border` `radius` `shadow` |
| `table.cell`, `table.header` | `fill` `border` `color` `weight` |
| `table.stripe` | a colour token |
| `code.tabSize` | integer ≥ 1 |
| `code.tokens[]` | `class` `color` |
| `connector.default` | `line` `start` `end` |
| `connector.variants[]` | `name` `line` `start` `end` |

Font and line height belong to roles only; a variant or default that wants another face uses
another role on the author's side. Shadow is a shape *variant* property, not a default. There are
no table variants in the file: `variant="striped"` is built into the language and paints
`table.stripe` on every other body row (an empty `stripe` makes it do nothing).

### Value rules

- **Roles**: `size` > 0, `lineHeight` > 0 (a multiplier), `align` present (`start` `center` `end`
  `justify`). `weight` is a CSS integer weight; the PPTX turns weights ≥ 600 into bold and below
  into regular — so 500 is medium on the web and regular in PowerPoint. Use 400, 600, 700.
  `font`/`color` are token names (no functions).
- **Typography**: `tracking` is letter spacing as a multiple of the font size (−0.1 to 0.5);
  `case` is `upper`, or `none` to turn an inherited `upper` off. Upper case is display-only: the
  text stays as written, so editing in PowerPoint shows the original case.
- **Colours** in defaults and variants use the authoring grammar: a token, `tint(c, p%)`,
  `shade(c, p%)` (HSL lightness; `0 < p ≤ 100`), `alpha(c, 0–1)`, `gradient(c1 c2 … [, Ndeg])`,
  `radial(c1 c2 … [, at X% Y%])`, stop opacity `c/0–100`. Functions don't nest. Shape fills may be
  `pattern(<pattern> <fg> <bg>)` (patterns: `horizontal` `vertical` `diagonal-up` `diagonal-down`
  `wide-diagonal-up` `wide-diagonal-down` `cross` `diagonal-cross` `dots`) — nowhere else.
- **Single token names only** for role colours, code token colours, table `fill`/`color`/`stripe`
  (or `none` for a fill), and `text.mark.highlight` (which also allows `tint`/`shade`/`alpha` of
  one token, but no gradient — PowerPoint highlights hold one colour).
- **Borders**: `none`, `<width> <solid|dashed|dotted> <colour>`, or `<width> gradient(…)`. Table
  borders are the three-part form or `none`.
- **`radius`/`shadow`** name tokens of this theme. A radius only applies to `rounded-rect` shapes
  (a theme default radius is dropped silently on other presets) and to images. Don't make card
  radii much larger than the built-ins: text is inset by the radius.
- **Unchecked at load**: unknown radius/shadow token names in defaults and variants (the field is
  dropped), unknown `tag` names in `componentDefaults`, and a role `align` outside the four
  values. Check these by eye.

### Things worth knowing about this file

- **`body` is 24px / 1.5 in every built-in theme.** The 720px stage is exactly 20 body lines; that
  is a product fact, not a theme choice. What themes vary is the heading hierarchy and the fonts.
- **Separate variant namespaces** — text, shape, image, connector. They do not know each other.
- **A component default belongs to a tag or to a layout slot**, never both (no owner and two
  owners are both errors; the layout + slot must exist in the slot table above). A slot default
  sits above the tag default and below variants: `<blockquote slot="quote">` gets quote size, a
  `<blockquote>` in a flow slide stays body size. The built-in themes keep three slot contracts:
  `quote/quote` ≥ 32px, `stat/value` > title, `stat/label` ≥ body. A `{ "tag": "connector", … }`
  default styles connector labels (otherwise they are `caption`).
- **A shape variant does not style its children.** `card-header` will not embolden the text
  inside it; deck authors put the typography on the child.
- **`image.default` is deliberately empty.** Pictures have their own pixels; a default border
  would double-frame every screenshot. The frame belongs to `variant="framed"`.
- **`fit` is not here** — its default (`contain`) belongs to the language, so a theme swap never
  starts cropping faces.
- **`<mark>`** needs `text.mark.highlight`; without it any `<mark>` in a deck is an error
  (`<mark> needs a highlight color, and theme "corp" defines none — …`). On a dark theme,
  `shade(accent4, 55%)` (as in `ink`) keeps the light text readable.
- **`connector`** sets the cascade for `<connector>`: inline > variant > `connector.default`.
  `line` is one colour (no gradient, no `none`); `start`/`end` are `none`, `arrow`, or `dot`.
  With no default `line`, a connector without an inline `line=` is a build error.
- **Code**: classes are a closed set — `keyword` `name` `attr` `string` `number` `comment`
  `punct`, none repeated. A class left out inherits the code block's colour. `tabSize` applies to
  every `<pre>`; a standalone theme must set it (≥ 1). Pick token colours by eye, not by accent
  number: comment recedes to `muted`, and name/attr/string/number take four accents that sit
  apart on the wheel.

## `layouts/*.css`

Two files per layout, both **selector-less declaration lists** injected into the slide's rule:

- `<name>.css` — flow mode. The flow stage is a vertical flex column, so `justify-content` is
  vertical. Often one line; an empty (or comment-only) file means "the default stack":

  ```css
  /* title — centre the title group vertically */
  justify-content: center;
  ```

- `<name>.slots.css` — slot mode, and it must be **self-sufficient**: the flow rules do not
  apply in slot mode, so start from `display: grid`:

  ```css
  display: grid;
  grid-template-columns: 1fr;
  grid-template-rows: auto 1fr;
  grid-template-areas: "title" "body";
  align-content: start;
  ```

Rules:

- **The declared `layouts` list and the `*.css` file set must match both ways.** Declared with
  no file is an error; a file with no declaration is an error. Slot files are checked one way
  only — a layout may have no slot file (`canvas` has no slots), but a slot file for an
  undeclared layout is an error. A slot-mode slide whose layout has no slot file warns and
  renders in flow mode.
- **Placement only.** Allowed: `display` `flex` `flex-*` `grid` `grid-*` `gap` `row-gap`
  `column-gap` `padding*` `margin*` `width` `min-width` `max-width` `height` `min-height`
  `max-height` `box-sizing` `align-*` `justify-*` `place-*` `order` `aspect-ratio` `position`
  `inset*` `top` `right` `bottom` `left` `translate`. Anything else — colour, font,
  `text-align` — is a warning (`layout CSS declaration "color" is not layout — …`) and never
  reaches the PPTX master. A malformed declaration (no colon, no value) is an error. Comments are
  fine.
- **Vertical rhythm in slot mode is yours** (`grid-template-rows`, `align-content`) — that is
  why `justify` is an error on a slot-mode slide.
- **Horizontal: centre only.** `justify-items: center` where centring is the meaning; write
  nothing for left. `justify-items: start` changes no picture but shrinks items to content
  width, which destroys the PPTX slack.
- **Use literal px** for gaps inside layout CSS (`grid-template-rows: auto auto 56px auto`,
  `row-gap: 12px`) rather than `var(--balpyo-space-…)` — the space variables are not declared on
  every deck.
- Empty rows are spelled with `"."` in `grid-template-areas`, e.g.
  `"." "kicker" "title" "subtitle" "author" "date"` with `grid-template-rows: 1fr auto auto auto auto auto`
  for a cover that sits on the floor.
- The stage keeps its 64px padding in every layout except where the theme changes it
  (`image-full` uses `padding: 0` for a full bleed).

## `extends` — the merge rule

**Name matches → the child wins; new names append after the parent's order.** Parent order is
part of the rule because order is visible in the output (`clrScheme` slots, CSS variable order,
the list in an unknown-`variant` warning).

| Merge unit | What | If the child writes it |
|---|---|---|
| **name** | colour/font/space/radius/shadow tokens, text roles, tag and slot defaults, variants, code tokens (`class`), chrome declarations (`kind`) | that **whole entry** is the child's — none of the parent's other fields survive |
| **whole** | `tokens.json`'s `motion` | the child's object replaces the parent's entirely |
| **field** | the unnamed sections — `shape.default`, `connector.default`, table `cell`/`header`/`stripe`, `code.tabSize`, `text.mark` | only the written fields change; the rest is inherited (`""` and `0` mean "unset") |
| **file** | `layouts/<name>.css`, `layouts/<name>.slots.css`, `assets/<path>` | that **one file** is the child's; files not present stay the parent's |
| **list** | `theme.json`'s `layouts` and `backgrounds` | if written it wins (`layouts` must then match the merged file set again); if omitted the parent's list is inherited |

So re-declaring role `title` means you own every field of it — the parent's 56px and `fg` do
not remain; re-declaring variant `kicker` without `tracking`/`case` drops the parent's tracked
upper case. **There is no delete syntax**: a name the parent gave cannot be removed, only
overridden.

A child's `tokens.json`, `components.json`, and `layouts/` may be partial **or absent**. A
theme with no `extends` gets the full treatment: all three files present and complete.

## Worked example: `corp`, a small inheriting theme

Four files re-skin `minimal`: a purple accent and lavender surface, serif headings, a larger
coloured title, a bottom-left cover, a centred footer, and a `hero` gradient variant. The other
ten layouts, four roles, and the code colours stay `minimal`'s.

`themes/corp/theme.json` — replaces the footer; `pageNumber` and `layouts` are inherited:

```json
{
  "name": "corp",
  "extends": "minimal",
  "chrome": [
    { "kind": "footer", "align": "center", "role": "caption",
      "skipLayouts": ["title", "section-divider", "image-full", "end"] }
  ]
}
```

`themes/corp/tokens.json` — recolours two names, swaps one font, adds one radius:

```json
{
  "color": [
    { "name": "accent1", "rgb": "7C3AED", "slot": "accent1", "ref": "accent1" },
    { "name": "surface", "rgb": "F1EDFB", "slot": "lt2", "ref": "bg2" }
  ],
  "font": [
    { "name": "heading", "latin": "Noto Serif", "ea": "Noto Serif CJK KR", "generic": "serif", "scheme": "major" }
  ],
  "radius": [
    { "name": "xl", "px": 36 }
  ]
}
```

`themes/corp/components.json` — one role (written whole), variants on top of the parent's, a
field-level patch to the shape default, one code colour:

```json
{
  "text": {
    "roles": [
      { "name": "title", "font": "heading", "size": 64, "weight": 700, "lineHeight": 1.15,
        "color": "accent1", "align": "start" }
    ],
    "variants": [
      { "name": "kicker", "size": 16, "weight": 700, "color": "muted", "tracking": 0.1, "case": "upper" },
      { "name": "tagline", "size": 28, "color": "accent1" }
    ]
  },
  "shape": {
    "default": { "fill": "surface", "radius": "xl" },
    "variants": [
      { "name": "hero", "fill": "gradient(accent1 accent6, 135deg)",
        "border": "4 gradient(accent6 accent1)", "radius": "xl" }
    ]
  },
  "code": {
    "tokens": [ { "class": "keyword", "color": "accent1" } ]
  }
}
```

`themes/corp/layouts/title.slots.css` — the cover sits bottom-left (one file replaced whole):

```css
display: grid;
grid-template-columns: 1fr;
grid-template-rows: 1fr auto auto auto auto auto;
grid-template-areas: "." "kicker" "title" "subtitle" "author" "date";
```

Here `shape.default` keeps `minimal`'s `1 solid muted` border (field merge), `code.tabSize`
stays 2, and the new `xl` radius and `tagline`/`hero` variants are appended after the parent's.
(`#7C3AED` on white is 5.7:1 — check the other rules for your own palette.)

## Worked example: backgrounds, overlay, logo, and rhythm

`themes/summit/theme.json` — a corner glow on three layouts, a fraction page number:

```json
{
  "name": "summit",
  "extends": "minimal",
  "backgrounds": [
    { "layouts": ["title", "end", "section-divider"], "fill": "bg",
      "overlay": "radial(accent1/35 bg/0, at 0% 100%)" }
  ],
  "chrome": [
    { "kind": "footer", "align": "start", "role": "caption",
      "skipLayouts": ["title", "section-divider", "image-full", "end"] },
    { "kind": "pageNumber", "align": "end", "role": "caption", "format": "{n} / {total}",
      "skipLayouts": ["title", "section-divider", "image-full", "end"] }
  ]
}
```

`themes/summit/tokens.json` (excerpt) — motion with a fade between slides:

```json
{
  "motion": { "duration": 600, "stagger": 90, "delay": 150, "transition": "fade", "transitionDuration": 400 }
}
```

`themes/summit/components.json` (excerpt) — tracked kickers and a highlighter:

```json
{
  "text": {
    "mark": { "highlight": "tint(accent1, 75%)" },
    "variants": [
      { "name": "kicker", "size": 14, "weight": 700, "color": "accent1", "tracking": 0.16, "case": "upper" },
      { "name": "lead",   "size": 26, "color": "muted" }
    ]
  }
}
```

`themes/summit/layouts/title.slots.css` — a left cover of limited width, centred vertically:

```css
display: grid;
grid-template-columns: minmax(0, 760px);
grid-template-rows: auto auto auto 56px auto auto;
grid-template-areas: "kicker" "title" "subtitle" "." "author" "date";
row-gap: 12px;
align-content: center;
```

A brand theme with a picture cover and a logo (`themes/brand/theme.json`, with
`assets/cover.png` and `assets/logo.png`):

```json
{
  "name": "brand",
  "extends": "minimal",
  "backgrounds": [
    { "layouts": ["title"], "image": "assets/cover.png" },
    { "layouts": ["section-divider"], "fill": "gradient(accent1 accent6, 135deg)" },
    { "fill": "surface" }
  ],
  "chrome": [
    { "kind": "logo", "align": "center", "image": "assets/logo.png", "height": 24,
      "alt": "Brand logo", "skipLayouts": ["title"] }
  ]
}
```

## README example

```markdown
# summit

A cloud-keynote look built on `minimal`: dark text on white, one orange accent, tracked
uppercase kickers, a soft orange glow in the corner of title, section and end slides, a fade
between slides and an `n / total` page number.

summit only renames colors and sets rhythm. Layouts, roles and variant names are `minimal`'s — run
`balpyo theme info minimal` for what each variant is for.

## Works best for

- Product and service introductions, architecture walkthroughs, launch-style talks.
- Decks that alternate section dividers with dense content slides.

## Using it

- Let the orange (`accent1`) mark one thing per slide: the key number, the service being
  introduced, the main path in a diagram (`<connector variant="accent">`).
- Use the other accents for categories in diagrams and charts, not for emphasis.
- Do not add your own background to `title`, `section-divider` or `end` slides.

## Assets

None — summit ships no pictures. Put service icons and diagrams in the deck's `assets/`.
```

The built-in READMEs use the same shape: a first paragraph for the voice, then `## Works best
for`, `## Look`, `## Variants — what each one is for`, `## Colors — how to use them`, and
`## Assets` (each picture's `theme:` path and purpose).
