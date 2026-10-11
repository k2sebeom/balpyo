# Appearance: how a value is decided

Authors write meaning; the theme supplies values. This file says how the two meet, what the
three built-in themes actually set, and what survives into PowerPoint as an editable theme
reference. (The theme skill's `references/builtin-themes.md` has the built-in themes' complete
files.)

## The cascade

For every text and shape, the resolved look is:

**inline attribute > list container (`<li>` only) > variant > component default (slot default above tag default) > role / tokens**

- The **role** (`title` `heading` `body` `caption` `code`) sets the full baseline: font, size,
  weight, line height, colour, alignment. Each layer above patches only the fields it sets.
- **Tag defaults**: `<h3>` is heading at a smaller size; `<blockquote>` is body in `muted`;
  every other tag adds nothing (`<p>` is exactly `<text role="body">`).
- **Slot defaults** apply to the element in that slot: `quote`/`quote` is large (40px),
  `stat`/`value` is huge and coloured, `stat`/`label` is bold. A variant or an inline attribute
  beats them (`<blockquote slot="quote" size="48">`).
- **List attributes** (`size`, `color`, `align`, `anchor` on `ul`/`ol`) patch every `<li>`.
- **Inline runs** (`strong`, `span`, …) start from their block's resolved style.
- **Layout attributes have no cascade** — `gap`, `pad`, `w` are exactly what you wrote;
  variants never carry layout.
- **Shape variants do not style the text inside the shape.**

## What becomes a PowerPoint theme reference

| You write | PPTX gets | Survives a theme-colour change in PowerPoint? |
|---|---|---|
| a colour token (`accent1`) or a function of one (`tint(accent1, 40%)`) | `schemeClr` (+ lum/alpha modifiers) | yes |
| a literal `#RRGGBB` | `srgbClr` | no |
| role fonts from `heading` / `body` | theme major / minor font | yes |
| `mono` | a literal typeface | — |
| space and radius tokens | baked into coordinates and geometry | — |
| a shadow | baked per shape; only its colour stays a reference | colour only |

So prefer tokens: PowerPoint's Design → Variants → Colors moves every token-coloured object and
none of the literal ones.

## Built-in themes at a glance

`balpyo theme info <name>` prints each theme's README (what it is for). Same in all three: the
11 layouts, chrome placement (footer left, page number right, both caption, skipped on `title`,
`section-divider`, `image-full`, `end`), motion (600ms, 90ms stagger, 150ms delay, no
transition), the body/caption/code roles, the variant names, the code colours, the connector
look, the table look, and the `quote` and `stat` slot defaults' structure.

| | `minimal` (default) | `editorial` | `ink` |
|---|---|---|---|
| Voice | neutral business document | print page, serif headings that sit low | dark room |
| `fg` / `bg` | `1A1A1A` / `FFFFFF` | `1C1A17` / `FAF8F4` | `F2F4F7` / `15171B` |
| `muted` / `surface` | `6B7280` / `F3F4F6` | `6A6459` / `F0EBE3` | `9CA3AF` / `21252B` |
| `accent1`…`accent6` | `0F65F9` `0E8174` `15803D` `B45309` `DC2626` `9333EA` (blue teal green amber red purple) | `585CDC` `30776C` `487730` `7D6B32` `A25741` `B04774` | `3B96FF` `00A987` `00AD2B` `CB8700` `FF5E4F` `D75EFF` |
| `heading` font | Noto Sans | **Noto Serif** | Noto Sans |
| `body` / `mono` | Noto Sans / Noto Sans Mono (+ CJK KR faces) | same | same |
| `title` role | 56 / 700 / 1.2 | 60 / 400 / 1.2 | 52 / 600 / 1.2 |
| `heading` role | 36 / 700 / 1.3 | 32 / 700 / 1.3 | 34 / 600 / 1.3 |
| `body` / `caption` / `code` | 24/400/1.5 · 18/400/1.4 `muted` · 20/400/1.5 | same | same |
| `<h3>` | 28 | 26 | 28 |
| `quote` slot | 40 `fg` | 40 `fg` | 40 `fg` |
| `stat` value / label | 96 / 700 `accent1` · 28 / 700 | 96 / 700 `accent5` · 28 / 700 | 88 / 700 `accent1` · 28 / 700 |
| space `xs sm md lg xl 2xl` | 6 12 24 36 48 72 | 8 16 24 40 56 80 | 4 10 20 32 48 72 |
| radius `sm md lg` | 6 12 24 | 0 2 4 | 8 16 28 |
| `<mark>` | `tint(accent4, 75%)` | `tint(accent4, 70%)` | `shade(accent4, 55%)` |

Sizes are px / weight / line height. `body` is 24px / 1.5 in every theme: the 720px stage is
exactly 20 body lines.

### Variants

| Variant | `minimal` | `editorial` | `ink` |
|---|---|---|---|
| text `kicker` | 18 / 700 `accent1`, tracked 0.1em, upper case | 17 / 700 `accent5`, tracked, upper | 18 / 700 `accent1`, tracked, upper |
| text `lead` | 32 | 30 | 32 |
| shape (no variant) | fill `surface`, `1 solid muted`, radius `md` | fill `surface`, `1 solid muted`, radius `sm` | fill `surface`, no border, radius `md` |
| shape `card` | fill `bg`, radius `lg` | fill `bg`, radius `sm` | fill `surface`, radius `lg`, shadow |
| shape `card-header` | no border, radius `sm` | no border, radius `sm` | fill `accent1`, radius `sm` |
| shape `card-body` | fill `bg` | fill `bg` | fill `surface` |
| shape `card-backdrop` | no fill, `3 solid accent1`, radius `lg` | no fill, `2 solid accent5`, radius `sm` | no fill, `3 solid accent1`, radius `lg` |
| shape `stat-card` | no border, radius `lg`, shadow | no border, radius `md`, shadow | fill `surface`, radius `lg`, shadow |
| shape `badge` | `1 solid accent1`, radius `lg` | `1 solid accent5`, radius `lg` | no fill, `1 solid accent1`, radius `lg` |
| image `framed` | `1 solid muted`, radius `md`, shadow | `1 solid fg`, radius `sm`, shadow | no border, radius `md`, shadow |
| table cells / header / stripe | `1 solid muted` / `accent1` fill, `bg` bold text / `surface` | same | `1 solid surface` / same / same |
| connector default / `dashed` / `accent` / `plain` | `2 solid muted` + end arrow / dashed muted / solid `accent1` / no arrowhead | same | same |
| code colours | keyword `accent6`, name `accent1`, attr `accent2`, string `accent3`, number `accent4`, comment `muted`, punctuation inherits | same | same |

A variant fills only the fields it names; the rest come from the default (a `card-header` in
`minimal` keeps the default `surface` fill).

### Layout placement

| Layout | `minimal` | `editorial` | `ink` |
|---|---|---|---|
| `title` | vertically centred, **horizontally centred** | sits at the bottom, left | vertically centred, left |
| `section-divider` | centred vertically, left | at the bottom, left | vertically and **horizontally centred** |
| `content`, `code` | top-down stack | same | same |
| `two-column` | two equal columns | 1.4 : 1 | two equal columns |
| `image-text` | two equal columns, vertically centred | 1.2 : 1, top | 1 : 1.2, vertically centred |
| `quote` | vertically centred | same | same |
| `stat` | vertically and horizontally centred | same | same |
| `image-full` | full bleed (no margin), no chrome | same | same |
| `end` | vertically and **horizontally centred** | at the bottom, left | vertically centred, left |
| `canvas` | free | same | same |

**Centred slots have no slack** (see `pitfalls.md`): `minimal` `title` `stat` `end`;
`editorial` `stat`; `ink` `stat` `section-divider`. Keep their lines short.

## Engine values no theme changes

- Stage 1280×720 (16:9) or 960×720 (4:3); a 64px margin (also the band the footer, page
  number, and logo sit in); a 24px gap between a flow slide's children.
- Lists: 36px indent, `•` and `1.` markers, no gap between items.
- Links are `accent1`, underlined.
- `fit` defaults to `contain`.
- The slide background is the `bg` token, unless the theme declares a background for that
  layout.

## Colour notes for authors

- Accent numbers are **not a hierarchy**: `accent2` is not "the second most important colour".
  Use two or three accents on a deck, each with one meaning.
- Accents are tuned for ≥ 4.5:1 contrast against `bg`; on `surface` (cards) they drop a little
  (`minimal` `accent1` on `surface` is 4.49:1) — avoid small accent text on cards.
- `shade`/`tint` keep hue and saturation: `tint(accent3, 40%)` is a vivid light green, not
  grey.

## Accessibility

- `alt` names an `<image>` (`alt=""` = decorative), a `<shape>` (its text stays readable), a
  `<table>` and a `<chart>`.
- Reading order is document order on both targets. `order`, `at`, `x`/`y` and `<stack>` change
  only the picture — if the visual order matters to a screen reader, change the document order.

## Slots and PowerPoint placeholders

For reference when the user edits the PPTX: slotted single-frame content becomes placeholders —
`title` → centre title / subtitle / body; `content` → title / body; `two-column` → title / two
bodies; `image-full` and `image-text` → a picture placeholder for `media`; `quote`, `code`,
`stat`, `end` → title and bodies. Chrome (footer, page number, logo) is never a placeholder.
