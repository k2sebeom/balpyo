# The theme package

Four file groups whose jobs do not overlap. The complete Korean original is
`${CLAUDE_PLUGIN_ROOT}/docs/reference/theme.md`; built-in themes to read are under
`${CLAUDE_PLUGIN_ROOT}/core/themes/`.

| File | Owns |
|---|---|
| `theme.json` | identity — name, parent, the layouts implemented, deck chrome |
| `tokens.json` | **the names of values** — what `accent1`, `md`, `heading` are |
| `components.json` | **appearance of roles and variants** — how big `role="body"` is, what colour a code keyword gets |
| `layouts/*.css` | **placement** — where a layout puts its children |

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
| `name` | string | must equal the directory name; empty is an error |
| `extends` | a theme name or `""` | resolved in the same vocabulary as `theme` (built-in ∪ project-local). Chains are allowed (`corp-dark` → `corp` → `minimal`); cycles are errors |
| `layouts` | list of layout names | must contain **all eleven** standard names. Inherited — omit it when extending. Names outside the catalogue are meaningless, since no slide can request them |
| `chrome` | array of declarations | below |

A standalone theme lists all eleven:

```json
["title","section-divider","content","two-column","image-full","image-text",
 "quote","code","stat","canvas","end"]
```

### Chrome declarations

Chrome is the footer and the page number. The vocabulary is **split between two files**: the
deck owns the content and the on/off (`balpyo.json`'s `footer` string and `pageNumber` bool),
the theme owns placement, alignment, role, and exclusions. Both are needed for anything to
render; config-only produces a warning and no chrome.

| Key | Values | Notes |
|---|---|---|
| `kind` | `footer` · `pageNumber` | declaring the same `kind` twice is an error |
| `align` | `start` · `center` · `end` | **mandatory** — position inside the band |
| `role` | one of the five text roles | **mandatory** — chrome has no size or colour of its own; reuse `caption` |
| `skipLayouts` | layout names of **this** theme (optional) | a name that is not a layout here is an error |

Chrome lives in the 64px band below the content box, so it never overlaps the body. Page
numbers count from 1 and **count excluded slides too** (the PPTX emits
`<a:fld type="slidenum">`, and no viewer can count a cover out). `<slide skip>` is different:
that slide leaves the deck entirely and the following numbers shift up.

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
               "color": "fg", "alpha": 0.15 }]
}
```

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

Unknown keys are errors (`$comment` excepted). All five text roles must be present and
complete; `size` and `lineHeight` must be > 0, `align` must be there, and `font`/`color` must
name tokens **this** theme has — a literal colour is rejected because it cuts the PPTX theme
reference.

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
      { "tag": "blockquote", "color": "muted" }
    ],
    "variants": [
      { "name": "kicker", "size": 18, "weight": 700, "color": "accent1" },
      { "name": "lead",   "size": 32 }
    ]
  },
  "shape": {
    "default": { "fill": "surface", "border": "1 solid muted", "radius": "md" },
    "variants": [
      { "name": "card",          "fill": "bg", "radius": "lg" },
      { "name": "stat-card",     "border": "none", "radius": "lg", "shadow": "soft" }
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
- **Three separate variant namespaces** — text, shape, image. They do not know each other.
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
| **field** | the unnamed sections — `shape.default`, `image.default`, table `cell`/`header`/`stripe`, `code.tabSize` | only the written fields change; the rest is inherited (`""` and `0` mean "unset") |
| **file** | `layouts/<name>.css`, `layouts/<name>.slots.css` | that **one file** is the child's; files not present stay the parent's |
| **list** | `theme.json`'s `layouts` | if written it wins (and must again be all eleven); if omitted the parent's eleven are inherited |

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
