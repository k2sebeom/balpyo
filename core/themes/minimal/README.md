# minimal

The default theme: white background, dark text, one blue accent, Noto Sans throughout. It stays
out of the way so the content carries the slide. When you are not sure which theme to use, use
this one.

## Works best for

- Technical talks, internal reviews, documentation-style decks, anything that will be edited
  further in PowerPoint.
- Decks with many diagrams, tables and code — the neutral palette keeps them readable.
- Not ideal for a dark room or a stage — use `ink` there.

## Look

- Covers (`title`, `section-divider`, `end`) center the title group vertically.
- Headings are bold sans; body text is 24px at 1.5 line height.
- Soft rounded corners (`radius` sm 6 · md 12 · lg 24) and a single soft shadow.
- The footer and page number sit along the bottom, hidden on covers.

## Variants — what each one is for

The variant names are the vocabulary every built-in theme shares, so a deck that uses them
re-skins cleanly with `--theme`.

| Element | Variant | Use it for |
|---|---|---|
| `<text>` · `p` | `kicker` | The small uppercase line above a title (`slot="kicker"`). One per slide, never a sentence. |
| `<text>` · `p` | `lead` | The opening paragraph under a heading — one or two lines, larger than body. |
| `<shape>` | `card` | A grouped block of content in a `row`/`grid` of peers. Filled with `bg` and rounded — on the white slide give it `fill="surface"` or a `border` so it reads as a panel. |
| `<shape>` | `card-header` | The label strip at the top of a `card`. Put a short `<text>` inside. |
| `<shape>` | `card-body` | The content area under a `card-header`. |
| `<shape>` | `card-backdrop` | An outlined frame around a whole group of cards or a diagram — the "this is one thing" box. |
| `<shape>` | `stat-card` | A tile holding one big number and its caption (with the `stat` layout or a row of KPIs). |
| `<shape>` | `badge` | A pill-shaped tag or label ("NEW", "v2", a status). Short text only. |
| `<connector>` | `accent` | The main path through a diagram. |
| `<connector>` | `dashed` | Optional, async or secondary relationships. |
| `<connector>` | `plain` | A line without an arrowhead — grouping, not flow. |
| `<image>` | `framed` | Screenshots and photos that sit on the slide rather than bleed off it. |

## Colors — how to use them

Write color names, never hex values — that is what lets a deck switch themes.

| Token | Use it for |
|---|---|
| `fg` · `bg` | Text and slide background. |
| `muted` | Captions, secondary text, de-emphasized lines. |
| `surface` | Panels and cards that sit on the background. |
| `accent1` … `accent6` | blue · teal · green · amber · red · violet. All six have the same contrast and brightness, so the number is not a rank. |

`accent1` is this theme's signature accent — the kicker, badge and backdrop already use it. Give
the other accents a meaning per deck (say, `accent3` for "shipped", `accent4` for "at risk"), use two
or three of them, and keep each meaning the same on every slide.

## Assets

None — minimal ships no pictures. Use your project's `assets/` for images and icons.
