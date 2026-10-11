# harbor

A standalone example theme: cool light-grey background, white cards that float on it, navy-black
text and heavy (800) headings. It is a complete design system on its own (`extends` is empty) and a
starting point for writing your own theme.

## Works best for

- Product updates, status reports, quarterly reviews — decks built out of cards, stats and short
  bullet lists.
- Dashboards-on-slides: `stat-card` rows and card grids are where it looks most at home.
- Less suited to long-form prose or full-bleed photography; use `editorial` for those.

## Look

- Covers start the title group at the **top left** of the stage, not the middle — keep cover
  titles short so the empty space below reads as intentional.
- The `lead` paragraph is `muted`, so it steps back from the heading above it.
- Medium corners (`radius` sm 4 · md 8 · lg 16).
- Install it with `balpyo theme install <path to examples/themes/harbor>` or copy the directory to
  your project's `themes/harbor/`, then set `"theme": "harbor"` in `balpyo.json`.

## Variants — what each one is for

harbor uses the same variant names as the built-in themes, so a deck written for `minimal` switches
to harbor without warnings.

| Element | Variant | Use it for |
|---|---|---|
| `<text>` · `p` | `kicker` | The small uppercase line above a title (`slot="kicker"`). One per slide, never a sentence. |
| `<text>` · `p` | `lead` | The opening paragraph under a heading — one or two lines, larger than body. |
| `<shape>` | `card` | A grouped block of content in a `row`/`grid` of peers. A white `surface` panel with a soft shadow that floats on the grey background — no border needed. |
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
| `accent1` … `accent6` | indigo · teal · green · amber · signal red · magenta. All six have the same contrast and brightness, so the number is not a rank. |

`accent1` (structure) and `accent5` (signal) are this theme's pair — headers and frames use indigo,
kickers and badges use red. Give the other accents a meaning per deck (say, `accent3` for
"shipped", `accent4` for "at risk"), use two or three of them, and keep each meaning the same on every slide.

## Assets

None — harbor ships no pictures. A theme that does would list each file under `assets/` here with
its `theme:` path and what it is for.
