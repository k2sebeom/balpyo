# editorial

A print voice: warm paper background, serif headings (Noto Serif) over a sans body, square
corners and a terracotta accent. It reads like a magazine page.

## Works best for

- Narrative talks, essays, research summaries, keynotes built around quotes and big statements.
- Decks with long titles — the serif heading role is light (400) and large (60px).
- Less suited to dense dashboards or many-box diagrams; the near-square corners and muted palette
  make busy slides feel heavy. Prefer `minimal` for those.

## Look

- Covers sit the title group at the **bottom left** of the stage, like a magazine cover —
  leave the top of the cover empty (or use `image-full`).
- Generous spacing (`space` up to 80px) — write less per slide than in `minimal`.
- Nearly square corners (`radius` md 2 · lg 4); `framed` images get a thin `fg` rule.
- The `quote` layout is the signature — use it for the lines you want remembered.

## Variants — what each one is for

The variant names are the vocabulary every built-in theme shares, so a deck that uses them
re-skins cleanly with `--theme`.

| Element | Variant | Use it for |
|---|---|---|
| `<text>` · `p` | `kicker` | The small uppercase line above a title (`slot="kicker"`). One per slide, never a sentence. |
| `<text>` · `p` | `lead` | The opening paragraph under a heading — one or two lines, larger than body. |
| `<shape>` | `card` | A grouped block of content in a `row`/`grid` of peers. Paper-colored with almost square corners — it reads as a printed box, so keep it to two or three per slide. |
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
| `accent1` … `accent6` | indigo · teal · green · ochre · terracotta · rose. All six have the same contrast and brightness, so the number is not a rank. |

`accent5` is this theme's signature accent — the kicker, badge and backdrop already use it. Give
the other accents a meaning per deck (say, `accent3` for "shipped", `accent4` for "at risk"), use two
or three of them, and keep each meaning the same on every slide.

## Assets

None — editorial ships no pictures. Photography with warm tones suits it best; put images in
your project's `assets/`.
