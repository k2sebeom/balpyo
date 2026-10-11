# Theme checklist

Three tiers, by who catches the mistake. Tier 1 the loader catches for you. Tier 2 is what the
built-in themes are held to but **nothing checks for a project or installed theme** — those are
yours to police. Tier 3 nothing checks; it shows up in the output.

Work top to bottom before telling the user a theme is done.

## Tier 1 — the loader stops the build (before any slide renders)

Run `balpyo check --project <dir> --theme <name> --check-layout --strict` and get exit 0. Run
`balpyo build --project <dir> --theme <name> --target html,pptx --strict` once as well: a few
font and colour-reference problems only surface when the PPTX is assembled.

Identity and resolution

- [ ] `theme.json` has a non-empty `name`, and it **equals the directory name**.
- [ ] The local theme does not shadow a built-in name (`themes/minimal/` is an error even when
      building another theme), and no installed theme has the same name.
- [ ] `extends` names an existing theme and the chain does not cycle (an installed theme may
      extend only built-in or installed themes).
- [ ] `balpyo.json`'s `theme` (or `--theme`) is this name. An unknown name is an error, never a
      silent fallback to `minimal`.

Layouts

- [ ] The `layouts` declaration and the `layouts/*.css` file set match **both ways** —
      declared-without-file and file-without-declaration are both errors.
- [ ] No `*.slots.css` for an undeclared layout (the one-way half of the same check).
- [ ] An inheriting theme omits `layouts` entirely, or restates the full list.
- [ ] No malformed declaration (missing colon or value) in any layout CSS.

Chrome

- [ ] Every declaration has `kind` (`footer` `pageNumber` `logo`) and `align`; `footer`/
      `pageNumber` also need `role`, and `logo` must not have one.
- [ ] A `logo` has `image` and `height` (and should have `alt`); text kinds carry none of the
      three.
- [ ] A `pageNumber` `format`, if any, holds `{n}` exactly once (and at most uses `{total}`);
      only `pageNumber` has a format.
- [ ] No `kind` declared twice.
- [ ] Every `skipLayouts` name is a layout of *this* theme.

Backgrounds and assets

- [ ] Each `backgrounds` entry has `fill` **or** `image`, not both and not neither; neither
      `fill` nor `overlay` is `none`.
- [ ] No layout in two entries, at most one default (an entry without `layouts`), and every
      listed name is a layout of this theme.
- [ ] An `overlay` is a colour value (usually a gradient fading to `…/0`); stop opacities are
      0–100 with no `%`.
- [ ] Every background and logo picture is a real PNG/JPEG under the theme's `assets/` (paths
      written `assets/…`, no SVG), with an extension matching its content.
- [ ] Pictures meant for slides (`theme:` paths) are PNG/JPEG/SVG with matching content, and a
      child theme that overrides one keeps the parent's path. SVGs are self-contained and give
      px `width`/`height` on the root `<svg>`.

Tokens

- [ ] All required names present: colours `fg` `bg` `muted` `surface` `accent1`…`accent6`;
      fonts `heading` `body` `mono`; spaces `xs` `sm` `md` `lg` `xl` `2xl`; radii `sm` `md` `lg`;
      shadow `soft` (inherited ones count).
- [ ] Every colour token has a known `clrScheme` `slot`, and no two share one.
- [ ] Adding a *new* colour name only where a slot is free — built-ins use 10 of 12, leaving
      `hlink` and `folHlink`. Otherwise recolour an existing name.
- [ ] Exactly one font token with `"scheme": "major"` and one with `"minor"` (PPTX build).
- [ ] `motion`, if written, has integer `duration` > 0 and `stagger`, `delay` ≥ 0 (ms), and
      `transition` is `fade`, `push`, `none`, or absent.
- [ ] No unknown keys anywhere in `tokens.json` (keys starting with `$comment` excepted).

Components

- [ ] All five text roles present: `title` `heading` `body` `caption` `code`.
- [ ] Each role has `size` > 0, `lineHeight` > 0, and an `align`.
- [ ] Every `font` and `color` on a role names a token **this theme has**; no literal colours.
- [ ] Every colour in defaults, variants, images, and tables is a token reference (a name,
      `tint`/`shade`/`alpha` of one, or a gradient of tokens) — table cells, roles, and code
      tokens single token names only. This is checked even for variants no slide uses.
- [ ] A component default names a `tag` **or** a `layout` + `slot` (a real slot of that
      layout), never both and never neither.
- [ ] Text variants and defaults carry only `size` `weight` `color` `tracking` `case` (no
      `font`, `lineHeight`, `align`).
- [ ] `shape.default` has only `fill` `border` `radius`; `image.default` is `{}`; image variants
      have no `fill`.
- [ ] Every `tracking` is within −0.1 … 0.5 and every `case` is `upper` or `none`.
- [ ] `text.mark.highlight` is one colour, not a gradient.
- [ ] `connector` lines use the border grammar with a token colour (no gradient, no `none`);
      `start`/`end` are `none`, `arrow`, or `dot`.
- [ ] `pattern(…)` appears only in shape fills (`shape.default` / shape variants).
- [ ] No `table.variants` (striping is built in; only `table.stripe` is a theme value).
- [ ] No unknown keys anywhere in `components.json`.
- [ ] `code.tabSize` ≥ 1 (a standalone theme must write it); every `code.tokens` class is one of
      `keyword` `name` `attr` `string` `number` `comment` `punct`, none repeated, each colour a
      token name.

Warnings (which `--strict` promotes to errors)

- [ ] No non-placement property in `layouts/*.css` (including `text-align`) — colour and font
      there never reach the PPTX.
- [ ] Chrome the config turns on is declared by the theme, or accept the warning knowingly.
- [ ] Two chrome kinds (e.g. `logo` and `pageNumber`) do not share an `align` — they overlap.
- [ ] Every font family the tokens name is installed on the machine that builds (and on CI).
      Not a loader check: each measuring build warns per missing family, and `--strict` fails.

## Tier 2 — what the built-ins are held to, unchecked for your theme

- [ ] **All eleven layouts implemented.** The loader accepts fewer, but a slide using a missing
      layout fails, and nothing falls back to another theme.
- [ ] **Slot CSS for all ten slotted layouts** (`canvas` has none), and each file carries its
      own `display: grid` plus every slot name of that layout in `grid-template-areas`.
- [ ] **The six accents satisfy the three numbers** — bg contrast ≥ 4.5:1, pairwise hue gap
      ≥ 25°, relative-luminance band (max/min) ≤ 1.25, by WCAG 2.1 relative luminance. Compute
      and show them.
- [ ] **The variant vocabulary matches the built-ins** — names *and* order: text `kicker`
      `lead`; shape `card` `card-header` `card-body` `card-backdrop` `stat-card` `badge`;
      image `framed`; connector `dashed` `accent` `plain`; plus a `table.stripe` colour. Add on
      top; never rename, or a deck that moves between themes starts emitting warnings.
- [ ] **The sections decks depend on are present** (inherited or written): `motion` in
      `tokens.json`, `text.mark.highlight`, and `connector.default` with a `line`. The loader
      accepts a theme without them, but every deck using `enter`, `<mark>`, or `<connector>`
      then fails to build.
- [ ] **`body` is 24px / line-height 1.5.** The 720px stage is exactly 20 body lines.
- [ ] **The three slot contracts hold**: `quote`/`quote` ≥ 32px, `stat`/`value` larger than
      the title, `stat`/`label` ≥ body. A standalone theme must write these slot defaults
      itself, or the quote renders at body size and the stat label undercuts the body.
- [ ] **`image-full` is in the footer/page-number `skipLayouts`** (or the chrome is placed where
      it does not sit on a full-bleed picture and its caption).
- [ ] **Role weights are 400, 600, or 700** — PowerPoint draws anything below 600 as regular.
- [ ] Names the loader does not verify are right: radius/shadow tokens named in defaults and
      variants exist, `componentDefaults` `tag`s are real tags, role `align` is one of `start`
      `center` `end` `justify`, every `theme.json` key is spelt as in `package.md`.

## Tier 3 — nothing checks these; they surface in the output

- [ ] The theme directory has a `README.md` (exact name) saying what decks it suits, what each
      variant is for, and the `theme:` path and purpose of every picture under `assets/`.
      `balpyo theme info <name>` prints it — read it back once as a deck author would.
- [ ] The `mono` token's `ea` is a **monospace CJK** face (`Noto Sans Mono CJK KR`). A
      proportional face draws a Korean comment line 7.9% narrower in the browser than in
      PowerPoint.
- [ ] `latin` and `ea` of every font token are the same family style (sans with sans, serif with
      serif, mono with mono).
- [ ] **No `justify-items: start`** in any slot CSS. It changes nothing visually and zeroes the
      text box slack, so PowerPoint re-breaks the line and shrinks it.
- [ ] `justify-items: center` only where centring *is* the meaning (cover, a single `stat`
      number, the closing slide) — and tell deck authors in the README that one-line text in
      those slots must stay short.
- [ ] `clrScheme` slots follow the convention, not brightness: a **dark** theme still puts `fg`
      in `dk1` and `bg` in `lt1`, or PowerPoint's theme-colour editor comes out inverted.
- [ ] On a dark background, fills and lines use `surface`, not `muted` (as `ink`'s table borders
      do) — grey on black outshines the text.
- [ ] The accents still read on an **inverted** slide (`<slide invert>` puts them on `fg`). The
      4.5:1 rule is against `bg` only: with a near-black `fg`, no colour reaches 4.5:1 against
      both (about 3.9:1 is the ceiling). Aim for ≥ 3:1 on `fg` — enough for headings, numbers,
      bars and lines — and tell deck authors in the README to keep small accent text off
      inverted slides.
- [ ] Variants that put `bg` text on an accent fill (`badge`, `card-header`, table header) have
      ≥ 4.5:1 there too.
- [ ] The `mark` highlight is **one colour**, so it cannot suit light and dark text equally. Pick
      it for the theme's normal text (≥ 4.5:1 between `fg` and the highlight); if it fails under
      the inverted slide's light text, say so in the README.
- [ ] An off-centre `radial(…, at X% Y%)` background or overlay was checked in PowerPoint:
      OnlyOffice ignores the focus and draws it from the centre.
- [ ] Code-token colours were chosen **by eye**, not by accent number: comment recedes to
      `muted`, and name/attr/string/number take four accents that sit apart on the wheel.
- [ ] Card radii are not much larger than the built-ins' (text is inset by the radius, so card
      grids written for other themes would overflow).

## Final pass — look at it

- [ ] `balpyo check --theme <name> --check-layout --strict` exits 0 on a real deck. Values move
      geometry: a bigger title adds a line and the line overflows.
- [ ] `balpyo snap --theme <name>` and read the PNGs; then
      `balpyo preview --theme <name> --gallery preview-<name>` (the gallery may not overlap
      `--out`), and the same deck under the parent theme in a second gallery. What is identical
      is what you did not claim. The PPTX column needs OnlyOffice or LibreOffice + `pdftoppm`.
- [ ] Backgrounds and the logo show on the layouts you meant, and nowhere in `skipLayouts`.
- [ ] Build a deck that uses `enter`, `<mark>`, `<connector>`, a striped table, `<pre lang>`, and
      one `<slide invert>` under the theme — that exercises every section above (SKILL.md §4
      lists a full exercise deck).
- [ ] Open the PPTX in PowerPoint or Keynote and check the theme colour editor shows your
      palette in sane places.
- [ ] Only then write the name into `balpyo.json`.
