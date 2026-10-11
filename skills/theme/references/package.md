# The theme package

Four file groups whose jobs do not overlap. The complete Korean original is
`${CLAUDE_PLUGIN_ROOT}/docs/reference/theme.md`; built-in themes to read are under
`${CLAUDE_PLUGIN_ROOT}/core/themes/`.

| File | Owns |
|---|---|
| `theme.json` | identity — name, parent, the layouts implemented, deck chrome, slide backgrounds |
| `tokens.json` | **the names of values** — what `accent1`, `md`, `heading` are |
| `components.json` | **appearance of roles and variants** — how big `role="body"` is, what colour a code keyword gets |
| `layouts/*.css` | **placement** — where a layout puts its children |
| `README.md` | optional, not loaded — the theme's usage guide for deck authors, printed verbatim by `balpyo theme info` |
| `assets/` | optional pictures the theme owns — background images and the logo (PNG/JPEG), plus pictures slides use via `theme:` (PNG/JPEG/SVG) |

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
| `name` | string | must equal the directory name under a project's `themes/`; for an installed theme it *is* the installed name (the source directory name is irrelevant); empty is an error |
| `extends` | a theme name or `""` | resolved in the same vocabulary as `theme` (built-in ∪ project-local ∪ installed). Chains are allowed (`corp-dark` → `corp` → `minimal`); cycles are errors |
| `layouts` | list of layout names | must contain **all eleven** standard names. Inherited — omit it when extending. Names outside the catalogue are meaningless, since no slide can request them |
| `chrome` | array of declarations | below |
| `backgrounds` | array of background declarations (optional) | below; inherited as a list — if the child writes it, the child's list wins |

A standalone theme lists all eleven:

```json
["title","section-divider","content","two-column","image-full","image-text",
 "quote","code","stat","canvas","end"]
```

### Chrome declarations

Chrome is the footer, the page number, and the logo. For the two text kinds the vocabulary is
**split between two files**: the deck owns the content and the on/off (`balpyo.json`'s `footer`
string and `pageNumber` bool), the theme owns placement, alignment, role, and exclusions. Both
are needed for anything to render; config-only produces a warning and no chrome. **The logo is
the theme's alone** — declaring it turns it on for every slide outside `skipLayouts`.

| Key | Values | Notes |
|---|---|---|
| `kind` | `footer` · `pageNumber` · `logo` | declaring the same `kind` twice is an error |
| `align` | `start` · `center` · `end` | **mandatory** — position inside the band. Two chrome kinds on the same `align` overlap and warn |
| `role` | one of the five text roles | **mandatory** for `footer`/`pageNumber`, not used by `logo` — chrome has no size or colour of its own; reuse `caption` |
| `skipLayouts` | layout names of **this** theme (optional) | a name that is not a layout here is an error |
| `image` · `height` · `alt` | `logo` only — a picture under `assets/`, logical px height, alt text | `image` and `height` are required; width follows the picture's ratio. On a text kind these keys are an error |
| `format` | `pageNumber` only (optional) — text holding `{n}` exactly once, optionally `{total}` | `"{n} / {total}"` renders `3 / 13` |

```json
{ "kind": "logo", "align": "center", "image": "assets/logo.png", "height": 24,
  "alt": "Brand logo", "skipLayouts": ["title"] }
```

Chrome lives in the 64px band below the content box, so it never overlaps the body. Page
numbers count from 1 and **count excluded slides too** (the PPTX emits
`<a:fld type="slidenum">`, and no viewer can count a cover out). `<slide skip>` is different:
that slide leaves the deck entirely and the following numbers shift up.

With `format`, only `{n}` stays a live PPTX field. The text around it, `{total}` included, is
written at build time, so a slide added in PowerPoint renumbers but the total stays stale until
the next build (DrawingML has no slide-count field). There is no zero-padding (`03`): a field
cannot pad, and padding only the web would make the two outputs disagree. A bad format is a
load error: `pageNumber format "{page}" needs {n} exactly once and may use {total} — …`.

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
| `fill` | a token, a token variant, or a token gradient | the slide's base fill |
| `image` | a picture under `assets/` | covers the slide (aspect kept, centre-cropped — like `fit="cover"`) |
| `overlay` | a colour value — usually a gradient fading to transparent (optional) | one layer painted **over** the fill or picture: a dark band over a photo, a corner glow |

- One declaration is `fill` **or** `image`, never both — a PPTX slide background holds one
  fill, so a fill under a picture would vanish there.
- A layout appears in **one** declaration only, and there is one default. Overlaps are errors —
  there is no precedence rule to remember.
- **`overlay` is how a picture gets a tint.** `{ "image": "assets/cover.jpg", "overlay":
  "gradient(fg fg/0, 90deg)" }` darkens the photo from the left. On the web it is the top
  background layer. In the PPTX it is a **locked full-slide rectangle** behind all content (the
  slide background holds one fill only), and it cannot be selected or moved.
- Gradient stops take an opacity as `colour/percent` (`fg/0`, `accent1/35` — 0–100, no `%`), and
  `radial(c1 c2 … [, at X% Y%])` is a radial gradient (first colour at the centre). An
  off-centre radial is drawn correctly by PowerPoint but **from the centre by OnlyOffice**, so
  check corner glows in PowerPoint.
- On a `<slide invert>` the background stays, but a token fill shows that token's swapped colour.
- A layout with no background stays the `bg` colour.
- On the web it is the slide's CSS background; in the PPTX it is the slide's `p:bg` ("Format
  Background" in PowerPoint), not a movable shape. Colours stay `schemeClr`.

Pictures (backgrounds and the logo) must live under the theme's `assets/`, be real PNG/JPEG
files (SVG is not allowed there) whose extension matches the content, and are judged like `<image src>`. Only referenced
pictures are shipped. An inherited declaration's picture belongs to the theme that has it; a
child file at the same path wins.

Slides may use the same pictures: `<image src="theme:icons/a.png">` is `assets/icons/a.png`
looked up through the active theme's `extends` chain, child first — for built-in, project, and
installed themes alike. This is how a theme ships an icon or illustration library: a child theme
overrides one icon by placing a file at the same path, and the deck's `src` never changes. The
paths under `assets/` become public vocabulary — renaming or removing a file breaks every deck
that uses it. Loading only indexes `assets/`; bytes are read for referenced files only, so a
large library costs nothing until used. Icons may be **SVG**: self-contained (no script, no
external references), with px `width`/`height` on the root `<svg>`. The PPTX carries the
vector plus a browser-rendered PNG fallback.

## `tokens.json`

Five **ordered arrays** — order determines CSS variable order, `clrScheme` assembly, and the
order names appear in diagnostics. Required *names* (the loader checks names, not values):

| Category | Required names | PPTX slot |
|---|---|---|
| `color` | `fg` `bg` `muted` `surface` `accent1`…`accent6` | `clrScheme`, 12 places |
| `font` | `heading` `body` `mono` | `fontScheme` major/minor |
| `space` | `xs` `sm` `md` `lg` `xl` `2xl` | none |
| `radius` | `sm` `md` `lg` | none (OOXML has no radius) |
| `shadow` | `soft` | none (baked per shape; only the colour stays a reference) |

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

**`motion` is the one optional section** — the deck's entrance rhythm in ms, plus the default
slide transition. The nth `enter` on a slide starts at `delay + n × stagger` and runs for
`duration`. `transition` (`fade` · `push`, omitted = none) is what every slide does unless it
says `transition=`. `transitionDuration` defaults to `duration`; the PPTX folds it to the
nearest speed (≤600ms fast, ≤875ms med, else slow). It is one object, not an array, so a child
that writes it replaces it **whole**. A theme without `motion` loads, but any deck using
`enter` then fails (`… defines no motion — add "motion": …`). All built-ins carry 600 / 90 / 150
and no transition. The PPTX theme part has no place for it: each effect gets the timing baked in.

A colour token's four fields are the token-colour-preservation contract: `name` is what authors
write, `rgb` is the colour, `slot` is the `clrScheme` place it sits in, `ref` is what shapes and
runs reference it by — the two can differ (slot `dk1` is referenced as `tx1`).

| Slot | Convention | Referenced as |
|---|---|---|
| `dk1` | `fg` — **even in a dark theme** | `tx1` |
| `lt1` | `bg` — even in a dark theme | `bg1` |
| `dk2` | `muted` | `tx2` |
| `lt2` | `surface` | `bg2` |
| `accent1`…`accent6` | the six accents | same name |
| `hlink`, `folHlink` | unused by the built-ins — the only two free slots | same name |

A colour token **without** a slot is an error, and two tokens sharing a slot is an error. So a
theme has at most twelve colour names, which is why child themes recolour rather than extend.
Unfilled slots borrow the nearest token (`accent2`–`accent6` → `accent1`, links → `accent1`) —
a slot is never left empty, or PowerPoint's colour editor offers an off-theme colour.

**Both font slots must point at the same family.** The browser asks fontconfig when a glyph is
missing; LibreOffice and PowerPoint use the name in `a:ea` literally. This bites hardest on
`mono`: a monospace Latin needs a **monospace CJK** `ea` (`Noto Sans Mono CJK KR`).

## `components.json`

Unknown keys are errors (keys starting with `$comment` excepted — the same rule as
`tokens.json`); the error names the file, line:col and JSON path. All five text roles must be present and
complete; `size` and `lineHeight` must be > 0, `align` must be there, and `font`/`color` must
name tokens **this** theme has. Every other colour in the file — defaults, variants, images,
tables — is checked at load too: a literal `#RRGGBB` (or a gradient with a literal stop) is an
error even when no slide uses that variant, because it cuts the PPTX theme reference.

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

Things worth knowing about this file:

- **`body` is 24px / 1.5 in every theme.** The 720px stage is exactly 20 body lines; that is a
  product fact, not a theme choice. What themes vary is the heading hierarchy and the fonts.
- **Separate variant namespaces** — text, shape, image, table, connector. They do not know
  each other.
- **Tracking and case are theme typography**, with no author attribute, so every kicker in a
  deck matches. Roles, component defaults, and variants take `tracking` (letter spacing as a
  multiple of the font size, −0.1 to 0.5) and `case` (`upper`, or `none` to turn an inherited
  `upper` off). Upper case is display-only (CSS `text-transform`, PPTX `cap="all"`): the text
  stays as written, so editing in PowerPoint shows the original case. Out-of-range values are
  load errors.
- **`text.mark.highlight`** is the colour of `<mark>`, a single colour (token or tint/shade/alpha
  of one; a gradient is an error because PPTX `a:highlight` holds one colour). Without it, any
  `<mark>` in a deck is a build error. On a dark theme, `shade(accent4, 55%)` (as in `ink`) keeps
  the light text readable.
- **`connector`** sets the cascade for `<connector>`: inline > variant > `connector.default`.
  `line` uses the border grammar (no gradient, no `none`). `start`/`end` are `none`, `arrow`,
  or `dot`. With no default `line`, a connector without an inline `line=` is a build error.
  Connector **labels** are `caption` text plus an optional `{ "tag": "connector", … }` entry in
  `text.componentDefaults`.
- **Shape fills may be pattern hatches**: `"fill": "pattern(wide-diagonal-up accent5/60 bg/0)"`
  in `shape.default` or a shape variant (patterns: `horizontal` `vertical` `diagonal-up`
  `diagonal-down` `wide-diagonal-up` `wide-diagonal-down` `cross` `diagonal-cross` `dots`). They
  are a shape fill only: on text, borders, connectors, table cells, or backgrounds they are an
  error.
- **Colors in components.json use the authoring grammar.** A theme can own gradients and color
  variants (`"fill": "gradient(accent1 accent6, 135deg)"`, `"border": "4 gradient(accent6 accent1)"`,
  `"color": "tint(accent1, 40%)"`) so the deck only says `variant="hero"`. Every color must stay a
  token reference — no `#RRGGBB`, no literal gradient stops. Table cells take single tokens only.
- **Themes own pictures.** Put PNG/JPEG files under `assets/` and point at them from `theme.json`:
  `backgrounds` (per layout, `fill` *or* `image` — never both; an entry without `layouts` is the
  default) and a `logo` chrome entry (`image`, `height`, `alt`; it is on whenever declared).
  The deck never mentions them, so switching themes switches the backgrounds and the logo.
  Slides reach any other picture there with `<image src="theme:<path under assets/>">`.
- **A component default belongs to a tag or to a layout slot**, never both. A slot default sits
  above the tag default and below variants: `<blockquote slot="quote">` gets quote size, a
  `<blockquote>` in a flow slide stays body size. The built-in themes keep three slot contracts:
  `quote/quote` ≥ 32px, `stat/value` > title, `stat/label` ≥ body.
- **A shape variant does not style its children.** `card-header` will not embolden the text
  inside it; put the typography on the child (`<h3>`, `<text role="heading">`).
- **`image.default` is deliberately empty.** Pictures have their own pixels; a default border
  would double-frame every screenshot. The frame belongs to `variant="framed"`.
- **`fit` is not here** — its default (`contain`) belongs to the language, so a theme swap never
  starts cropping faces.
- **Code token classes are a closed set**: `keyword` `name` `attr` `string` `number` `comment`
  `punct`. Colours must be token names; a repeated class is an error; `tabSize` must be ≥ 1.
  A class left out inherits the resolved `pre` colour. Because the colours are references, code
  highlighting follows the palette automatically.

## `layouts/*.css`

Two files per layout, both **selector-less declaration lists** injected into the stage rule:

- `<name>.css` — flow mode. Often one line:

  ```css
  /* title — centre the title group vertically */
  justify-content: center;
  ```

- `<name>.slots.css` — slot mode, and it must be **self-sufficient**: the flow rules exclude
  slot mode (`:not(.balpyo-slots)`), so start from `display: grid`:

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
  undeclared layout is an error.
- **Placement only.** Colour and font declarations here are a warning, and they never reach the
  PPTX master. Appearance belongs to `components.json`.
- **Vertical rhythm in slot mode is yours** (`grid-template-rows`, `align-content`) — that is
  why `justify` is an error on a slot-mode slide.
- **Horizontal: centre only.** `justify-items: center` where centring is the meaning; write
  nothing for left. `justify-items: start` changes no picture but shrinks items to content
  width, which destroys the PPTX slack.
- Empty rows are spelled with `"."` in `grid-template-areas`, e.g.
  `"." "kicker" "title" "subtitle" "author" "date"` for a cover that sits on the floor.

## `extends` — the merge rule

**Name matches → the child wins; new names append after the parent's order.** Parent order is
part of the rule because order is visible in the output (`clrScheme` slots, CSS variable order,
the list in an unknown-`variant` warning).

| Merge unit | What | If the child writes it |
|---|---|---|
| **name** | colour/font/space/radius/shadow tokens, text roles, tag defaults, variants, code tokens (`class`), chrome declarations (`kind`) | that **whole entry** is the child's — none of the parent's other fields survive |
| **whole** | `tokens.json`'s `motion` | the child's object replaces the parent's entirely |
| **field** | the unnamed sections — `shape.default`, `image.default`, `connector.default`, table `cell`/`header`/`stripe`, `code.tabSize`, `text.mark` | only the written fields change; the rest is inherited (`""` and `0` mean "unset") |
| **file** | `layouts/<name>.css`, `layouts/<name>.slots.css`, `assets/<path>` | that **one file** is the child's; files not present stay the parent's |
| **list** | `theme.json`'s `layouts` and `backgrounds` | if written it wins (`layouts` must again be all eleven); if omitted the parent's list is inherited |

So re-declaring role `title` means you own every field of it — the parent's 56px and `fg` do
not remain. **There is no delete syntax**: a name the parent gave cannot be removed, only
overridden.

A child's `tokens.json`, `components.json`, and `layouts/` may be partial **or absent**. A
theme with no `extends` gets the full built-in treatment: all three files present and complete.

A complete worked example is `${CLAUDE_PLUGIN_ROOT}/testdata/fixtures/15_local_theme` — a
three-slide deck whose `themes/corp/` inherits `minimal` through four files
(`theme.json`, partial `tokens.json`, partial `components.json`, one `layouts/title.slots.css`)
and re-skins the cover, the title role, an accent, and the footer position. The other ten
layouts, four roles, and five code colours are identical to the parent in the goldens — that
sameness is the witness that inheritance worked.

`${CLAUDE_PLUGIN_ROOT}/examples/decks/ec2-intro/themes/summit/` shows the newer keys in one small
inheriting theme: a radial corner-glow `overlay`, a `"{n} / {total}"` page number, `motion` with
a fade transition, tracked upper-case kickers, and a `text.mark` highlight.
