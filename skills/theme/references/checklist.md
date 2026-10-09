# Theme checklist

Three tiers, by who catches the mistake. Tier 1 the loader catches for you. Tier 2 only the
repository's own test suite checks, and it runs on **built-in themes only** — in a project-local
theme those are yours to police. Tier 3 nothing checks; it shows up in the output.

Work top to bottom before telling the user a theme is done.

## Tier 1 — the loader stops the build (before any slide renders)

Run `balpyo check --project <dir> --theme <name> --check-layout --strict` and get exit 0.

Identity and resolution

- [ ] `theme.json` has a non-empty `name`, and it **equals the directory name**.
- [ ] The local theme does not shadow a built-in name (`themes/minimal/` is an error even when
      building another theme).
- [ ] `extends` names an existing theme (built-in ∪ project-local ∪ installed) and the chain does not cycle.
- [ ] `balpyo.json`'s `theme` (or `--theme`) is this name. An unknown name is an error, never a
      silent fallback to `minimal`.

Layouts

- [ ] The `layouts` declaration and the `layouts/*.css` file set match **both ways** —
      declared-without-file and file-without-declaration are both errors.
- [ ] No `*.slots.css` for an undeclared layout (the one-way half of the same check).
- [ ] A standalone theme declares all eleven standard layouts; an inheriting theme omits the
      list entirely or restates all eleven.

Chrome

- [ ] Every declaration has `kind` and `align`; `footer`/`pageNumber` also need `role`.
- [ ] A `logo` has `image` and `height` (and should have `alt`); text kinds carry none of the
      three.
- [ ] No `kind` declared twice.
- [ ] Every `skipLayouts` name is a layout of *this* theme.

Backgrounds and assets

- [ ] Each `backgrounds` entry has `fill` **or** `image`, not both and not neither.
- [ ] No layout in two entries, at most one default (an entry without `layouts`), and every
      listed name is a layout of this theme.
- [ ] Every background and logo picture is a real PNG/JPEG under the theme's `assets/`.
- [ ] Pictures meant for slides (`theme:` paths) are PNG/JPEG with matching content, and a child
      theme that overrides one keeps the parent's path.

Tokens

- [ ] All required names present: colours `fg` `bg` `muted` `surface` `accent1`…`accent6`;
      fonts `heading` `body` `mono`; spaces `xs` `sm` `md` `lg` `xl` `2xl`; radii `sm` `md` `lg`;
      shadow `soft`.
- [ ] Every colour token has a `clrScheme` `slot`, and no two share one.
- [ ] Adding a *new* colour name only where a slot is free — built-ins use 10 of 12, leaving
      `hlink` and `folHlink`. Otherwise recolour an existing name.

Components

- [ ] All five text roles present: `title` `heading` `body` `caption` `code`.
- [ ] Each role has `size` > 0, `lineHeight` > 0, and an `align`.
- [ ] Every `font` and `color` on a role names a token **this theme has**; no literal colours.
- [ ] Every colour in defaults, variants, images, and tables is a token reference (a name,
      `tint`/`shade`/`alpha` of one, or a gradient of tokens) — table cells single tokens only.
      This is checked even for variants no slide uses.
- [ ] A component default names a `tag` **or** a `layout` + `slot`, never both.
- [ ] No unknown keys anywhere in `components.json` (`$comment` is the only exception).
- [ ] `code.tabSize` ≥ 1; every `code.tokens` class is one of `keyword` `name` `attr` `string`
      `number` `comment` `punct`, none repeated, each colour a token name.

Warnings (which `--strict` promotes to errors)

- [ ] No non-placement property in `layouts/*.css` — colour and font there never reach the PPTX.
- [ ] Chrome the config turns on is declared by the theme, or accept the warning knowingly.
- [ ] Two chrome kinds (e.g. `logo` and `pageNumber`) do not share an `align` — they overlap.
- [ ] Every font family the tokens name is installed on the machine that builds (and on CI).
      Not a loader check: each measuring build warns per missing family, and `--strict` fails.

## Tier 2 — only tested for built-ins, so check these yourself

- [ ] **All eleven layouts implemented.** A missing layout does not fall back to another theme.
- [ ] **Slot CSS for all ten slotted layouts** (`canvas` has none), and each file carries its
      own `display: grid` plus every slot name of that layout in `grid-template-areas`.
- [ ] **The six accents satisfy the three numbers** — bg contrast ≥ 4.5:1, pairwise hue gap
      ≥ 25°, relative-luminance band (max/min) ≤ 1.25, by WCAG 2.1 relative luminance. Compute
      and show them; `core/palette_test.go` will not do it for a local theme.
- [ ] **The variant vocabulary matches the built-ins** — names *and* order: text `kicker`
      `lead`; shape `card` `card-header` `card-body` `card-backdrop` `stat-card` `badge`;
      image `framed`; table `striped`. Add on top; never rename, or a deck that moves between
      themes starts emitting warnings.
- [ ] **`body` is 24px / line-height 1.5.** The 720px stage is exactly 20 body lines.
- [ ] **The three slot contracts hold**: `quote`/`quote` ≥ 32px, `stat`/`value` larger than
      the title, `stat`/`label` ≥ body. A standalone theme must write these slot defaults
      itself, or the quote renders at body size and the stat label undercuts the body.

## Tier 3 — nothing checks these; they surface in the output

- [ ] The `mono` token's `ea` is a **monospace CJK** face (`Noto Sans Mono CJK KR`). A
      proportional face draws a Korean comment line 7.9% narrower in the browser than in
      PowerPoint.
- [ ] `latin` and `ea` of every font token point at the same family.
- [ ] **No `justify-items: start`** in any slot CSS. It changes nothing visually and zeroes the
      text box slack, so PowerPoint re-breaks the line and `normAutofit` shrinks it.
- [ ] `justify-items: center` only where centring *is* the meaning (cover, a single `stat`
      number, the closing slide) — and tell the user that one-line text in those slots must stay
      short.
- [ ] `clrScheme` slots follow the convention, not brightness: a **dark** theme still puts `fg`
      in `dk1` and `bg` in `lt1`, or PowerPoint's theme-colour editor comes out inverted.
- [ ] On a dark background, fills and lines use `surface`, not `muted` (as `ink`'s table borders
      do) — grey on black outshines the text.
- [ ] Code-token colours were chosen **by eye**, not by accent number: comment recedes to
      `muted`, and name/attr/string/number take four accents that sit apart on the wheel.

## Final pass — look at it

- [ ] `balpyo check --theme <name> --check-layout --strict` exits 0 on a real deck. Values move
      geometry: a bigger title adds a line and the line overflows.
- [ ] `balpyo preview --theme <name> --gallery preview-<name>` (the gallery may not overlap
      `--out`), and the same deck under the parent theme in a second gallery. What is identical
      is what you did not claim. The PPTX column needs OnlyOffice or LibreOffice + `pdftoppm`.
- [ ] Backgrounds and the logo show on the layouts you meant, and nowhere in `skipLayouts`.
- [ ] Open the PPTX in PowerPoint or Keynote and check the theme colour editor shows your
      palette in sane places.
- [ ] Only then write the name into `balpyo.json`.
